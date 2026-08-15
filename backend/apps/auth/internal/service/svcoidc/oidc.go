package svcoidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/auth/internal/service/svcauth"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/ark-iam/pkg/iam/svcaudit"
	"github.com/morehao/golib/glog"
	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/text/language"
)

const (
	pathLoggedOut = "/oidc/logged-out"
)

type OIDCProvider struct {
	Provider *op.Provider
	Storage  *OIDCStorage
	issuer   string
}

// isDevEnv 判断当前环境是否为开发环境（dev/空）。
// 密钥 fail-closed 规则仅对非 dev 环境生效：生产必须显式配置签名/加密密钥。
func isDevEnv() bool {
	if appconfig.Conf == nil {
		return true
	}
	return appconfig.Conf.Server.Env == "" || appconfig.Conf.Server.Env == "dev"
}

// isSupportedRSAPrivateKeyBlock 判断 PEM 块是否为受支持的 RSA 私钥编码：
// PKCS#1（"RSA PRIVATE KEY"）或 PKCS#8（"PRIVATE KEY"）。两者后续均由
// ParsePKCS8PrivateKey 优先解析、ParsePKCS1PrivateKey 兜底，与 RP 侧
// oidcauth.LoadSigningPublicKey 的宽容解析保持一致。
func isSupportedRSAPrivateKeyBlock(blockType string) bool {
	return blockType == "RSA PRIVATE KEY" || blockType == "PRIVATE KEY"
}

func loadSigningKey() (*rsa.PrivateKey, string, error) {
	if appconfig.Conf == nil {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		return privateKey, "auto-key", err
	}
	cfg := &appconfig.Conf.OIDC

	if cfg.SigningPrivateKeyPath != "" {
		pemData, err := os.ReadFile(cfg.SigningPrivateKeyPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, "", err
			}
			// H4：非 dev 环境显式配置了路径但文件缺失时 fail-closed——
			// 自动生成新密钥会静默更换 kid，导致所有已签发 token 失效且各 RP 公钥失同步。
			if !isDevEnv() {
				return nil, "", fmt.Errorf("signing private key file not found: %s (refusing to auto-generate in non-dev)", cfg.SigningPrivateKeyPath)
			}
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return nil, "", fmt.Errorf("failed to generate signing key: %w", err)
			}
			der := x509.MarshalPKCS1PrivateKey(privateKey)
			encoded := pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: der,
			})
			keyDir := filepath.Dir(cfg.SigningPrivateKeyPath)
			if keyDir != "." {
				if err := os.MkdirAll(keyDir, 0755); err != nil {
					return nil, "", fmt.Errorf("failed to create key directory: %w", err)
				}
			}
			if err := os.WriteFile(cfg.SigningPrivateKeyPath, encoded, 0600); err != nil {
				return nil, "", fmt.Errorf("failed to write signing key: %w", err)
			}
			keyID := cfg.SigningKeyID
			if keyID == "" {
				keyID = "auto-key"
			}
			return privateKey, keyID, nil
		}
		block, _ := pem.Decode(pemData)
		if block == nil || !isSupportedRSAPrivateKeyBlock(block.Type) {
			return nil, "", fmt.Errorf("invalid RSA private key PEM: %s", cfg.SigningPrivateKeyPath)
		}
		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, "", fmt.Errorf("failed to parse RSA private key: %w", err)
			}
		}
		rsaKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, "", fmt.Errorf("key is not RSA: %T", privateKey)
		}
		keyID := cfg.SigningKeyID
		if keyID == "" {
			keyID = "config-key"
		}
		return rsaKey, keyID, nil
	}

	if cfg.SigningPrivateKeyPEM != "" {
		block, _ := pem.Decode([]byte(cfg.SigningPrivateKeyPEM))
		if block == nil || !isSupportedRSAPrivateKeyBlock(block.Type) {
			return nil, "", fmt.Errorf("invalid RSA private key PEM in config")
		}
		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, "", fmt.Errorf("failed to parse RSA private key: %w", err)
			}
		}
		rsaKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, "", fmt.Errorf("key is not RSA: %T", privateKey)
		}
		keyID := cfg.SigningKeyID
		if keyID == "" {
			keyID = "config-key"
		}
		return rsaKey, keyID, nil
	}

	// H4：未配置任何签名密钥时，仅 dev 环境允许生成临时密钥；
	// 非 dev 环境 fail-closed，避免重启后 token 全量失效与 RP 公钥失同步。
	if !isDevEnv() {
		return nil, "", fmt.Errorf("oidc signing key not configured (set signingPrivateKeyPath or signingPrivateKeyPEM)")
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
	}
	return privateKey, "auto-key", nil
}

func loadEncryptionKey() ([32]byte, string, error) {
	if appconfig.Conf == nil {
		return sha256.Sum256([]byte("test-key-32-bytes-for-oidc-encrp")), "enc-key-1", nil
	}
	cfg := &appconfig.Conf.OIDC
	encKeyStr := cfg.EncryptionKey
	encKeyID := cfg.EncryptionKeyID
	if encKeyID == "" {
		encKeyID = "enc-key-1"
	}
	// H4：非 dev 环境必须显式配置 encryptionKey；
	// 缺省测试密钥是公开常量，生产使用等于 auth code 可被伪造。
	if encKeyStr == "" {
		if !isDevEnv() {
			return [32]byte{}, "", fmt.Errorf("oidc encryptionKey not configured (required in non-dev)")
		}
		encKeyStr = "test-key-32-bytes-for-oidc-encrp"
	}
	return sha256.Sum256([]byte(encKeyStr)), encKeyID, nil
}

func SetupOIDCProvider(issuer string) (*OIDCProvider, error) {
	privateKey, keyID, err := loadSigningKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load OIDC signing key: %w", err)
	}

	storage := NewOIDCStorage(NewRedisProtocolStateStore(), NewPersistentStore(), privateKey, keyID)

	encKey, encKeyID, err := loadEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load OIDC encryption key: %w", err)
	}

	opConfig := &op.Config{
		CryptoKey:                         encKey,
		CryptoKeyId:                       encKeyID,
		DefaultLogoutRedirectURI:          pathLoggedOut,
		CodeMethodS256:                    true,
		AuthMethodPost:                    true,
		GrantTypeRefreshToken:             true,
		BackChannelLogoutSupported:        true,
		BackChannelLogoutSessionSupported: true,
		SupportedUILocales:                []language.Tag{language.Chinese, language.English},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	opts := []op.Option{
		op.WithCustomAuthEndpoint(op.NewEndpoint("authorize")),
		op.WithLogger(logger.WithGroup("op")),
	}
	allowInsecure := false
	if appconfig.Conf != nil {
		allowInsecure = appconfig.Conf.OIDC.AllowInsecure || appconfig.Conf.Server.Env == "dev"
	}
	if allowInsecure {
		opts = append(opts, op.WithAllowInsecure())
	}

	provider, err := op.NewProvider(opConfig, storage, op.StaticIssuer(issuer), opts...)
	if err != nil {
		return nil, err
	}

	return &OIDCProvider{
		Provider: provider,
		Storage:  storage,
		issuer:   issuer,
	}, nil
}

func (p *OIDCProvider) BuildAuthCallbackURL(ctx context.Context, authRequestID string) string {
	return op.AuthCallbackURL(p.Provider)(op.ContextWithIssuer(ctx, p.issuer), authRequestID)
}

type OIDCAuthSvc interface {
	CompleteLogin(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error)
	SelectTenant(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error)
	CompleteLoginBySession(ctx *gin.Context, authRequestID string, sessionID string) (string, error)
}

type passwordAuthenticator interface {
	AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error)
	TenantsForPerson(ctx *gin.Context, personID string) ([]objauth.TenantOption, error)
}

type oidcAuthSvc struct {
	provider             *OIDCProvider
	authSvc              passwordAuthenticator
	ssoSessionStore      sso.SSOSessionStore
	applicationClientDao func() *dao.ApplicationClientDao
	applicationDao       func() *dao.ApplicationDao
}

func NewOIDCAuthSvc(provider *OIDCProvider) OIDCAuthSvc {
	return &oidcAuthSvc{
		provider:             provider,
		authSvc:              svcauth.NewAuthSvc(),
		ssoSessionStore:      sso.NewSSOSessionStore(),
		applicationClientDao: func() *dao.ApplicationClientDao { return dao.NewApplicationClientDao() },
		applicationDao:       func() *dao.ApplicationDao { return dao.NewApplicationDao() },
	}
}

func (svc *oidcAuthSvc) CompleteLogin(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
	authReq, err := svc.provider.Storage.AuthRequestByID(ctx.Request.Context(), req.AuthRequestID)
	if err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	personEntity, userEntity, tenants, err := svc.authSvc.AuthenticatePassword(ctx, req.Identifier, req.Password)
	if err != nil {
		return nil, err
	}
	authTime := time.Now()
	subject := buildOIDCSubject(personEntity.ID)
	// 优先尊重 ?tenant hint（如 SSO 会话过期后回退到密码登录时），但仅当 hint 是 person 的成员租户时才采用
	resolvedTenant := ""
	if ar, ok := authReq.(*AuthRequest); ok {
		if hint := ar.GetTenantID(); hint != "" {
			for _, t := range tenants {
				if t.TenantID == hint {
					resolvedTenant = hint
					break
				}
			}
		}
	}
	// 多租户：除非有合法的 tenant hint，否则暂不 done、不发 code，需用户先选租户
	if resolvedTenant == "" && len(tenants) > 1 {
		if err := svc.provider.Storage.CompleteAuthRequest(ctx.Request.Context(), req.AuthRequestID, subject, authTime, []string{"pwd"}, "", "", false); err != nil {
			return nil, code.GetError(code.OIDCSessionNotFound)
		}
		return &dtooidc.OIDCLoginResp{
			RequiresTenantSelection: true,
			Tenants:                 tenants,
		}, nil
	}
	// 单租户（或在多租户但 hint 命中成员租户）：自动选租户，done，发 code，并建 SSO 会话
	tenantID := resolvedTenant
	if tenantID == "" {
		tenantID = userEntity.TenantID
	}
	if err := svc.provider.Storage.CompleteAuthRequest(ctx.Request.Context(), req.AuthRequestID, subject, authTime, []string{"pwd"}, "", tenantID, true); err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}

	allowPersonCreateTenant := false
	if cid := clientIDFromAuthRequest(authReq); cid != "" {
		allowPersonCreateTenant = svc.resolveAllowPersonCreateTenant(ctx, cid, len(tenants))
	}

	resp := &dtooidc.OIDCLoginResp{
		ContinueURL:             svc.provider.BuildAuthCallbackURL(ctx.Request.Context(), req.AuthRequestID),
		TenantID:                tenantID,
		Tenants:                 tenants,
		AllowPersonCreateTenant: allowPersonCreateTenant,
	}

	if svc.ssoSessionStore != nil {
		sessionID, err := svc.ssoSessionStore.CreateSession(sessionAuditContext(ctx.Request.Context(), tenantID), personEntity.ID, []string{"pwd"})
		if err != nil {
			glog.Warnf(ctx, "[oidcAuthSvc.CompleteLogin] failed to create sso session: %v", err)
		} else {
			resp.SessionID = sessionID
			if aErr := svc.provider.Storage.AssociateSession(ctx.Request.Context(), req.AuthRequestID, sessionID); aErr != nil {
				glog.Warnf(ctx, "[oidcAuthSvc.CompleteLogin] associate session fail, err:%v, authRequestID:%s, sessionID:%s", aErr, req.AuthRequestID, sessionID)
			}
		}
	}

	return resp, nil
}

func (svc *oidcAuthSvc) SelectTenant(ctx *gin.Context, authRequestID string, tenantID string) (*dtooidc.OIDCLoginResp, error) {
	reqCtx := ctx.Request.Context()
	authReq, err := svc.provider.Storage.AuthRequestByID(reqCtx, authRequestID)
	if err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	if authReq.Done() {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	personID, perr := parseOIDCSubject(authReq.GetSubject())
	if perr != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	tenants, terr := svc.authSvc.TenantsForPerson(ctx, personID)
	if terr != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	ok := false
	for _, t := range tenants {
		if t.TenantID == tenantID {
			ok = true
			break
		}
	}
	if !ok {
		return nil, code.GetError(code.TenantNotExistError)
	}
	if err := svc.provider.Storage.CompleteAuthRequest(reqCtx, authRequestID, authReq.GetSubject(), authReq.GetAuthTime(), authReq.GetAMR(), "", tenantID, true); err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	allowPersonCreateTenant := false
	if cid := authReq.GetClientID(); cid != "" {
		allowPersonCreateTenant = svc.resolveAllowPersonCreateTenant(ctx, cid, len(tenants))
	}
	resp := &dtooidc.OIDCLoginResp{
		ContinueURL:             svc.provider.BuildAuthCallbackURL(reqCtx, authRequestID),
		TenantID:                tenantID,
		Tenants:                 tenants,
		AllowPersonCreateTenant: allowPersonCreateTenant,
	}
	if svc.ssoSessionStore != nil {
		if sessionID, sErr := svc.ssoSessionStore.CreateSession(sessionAuditContext(reqCtx, tenantID), personID, authReq.GetAMR()); sErr == nil {
			resp.SessionID = sessionID
			if aErr := svc.provider.Storage.AssociateSession(reqCtx, authRequestID, sessionID); aErr != nil {
				glog.Warnf(ctx, "[oidcAuthSvc.SelectTenant] associate session fail, err:%v, authRequestID:%s, sessionID:%s", aErr, authRequestID, sessionID)
			}
		} else {
			glog.Warnf(ctx, "[oidcAuthSvc.SelectTenant] failed to create sso session: %v", sErr)
		}
	}
	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionTenantSwitch,
		TenantID:   tenantID,
		Result:     "success",
		TargetType: "tenant",
		TargetID:   tenantID,
	})
	return resp, nil
}

func (svc *oidcAuthSvc) CompleteLoginBySession(ctx *gin.Context, authRequestID string, sessionID string) (string, error) {
	reqCtx := ctx.Request.Context()
	personID, err := svc.ssoSessionStore.ValidateSession(reqCtx, sessionID)
	if err != nil {
		return "", err
	}

	authReq, err := svc.provider.Storage.AuthRequestByID(reqCtx, authRequestID)
	if err != nil {
		return "", err
	}

	tenantID := ""
	tenants, tErr := svc.authSvc.TenantsForPerson(ctx, personID)
	if tErr == nil {
		if ar, ok := authReq.(*AuthRequest); ok {
			if hint := ar.GetTenantID(); hint != "" {
				for _, t := range tenants {
					if t.TenantID == hint {
						tenantID = hint
						break
					}
				}
			}
		}
		// membership safety: never issue a token for a tenant hinted but not owned by the user
		if tenantID == "" && len(tenants) > 0 {
			tenantID = tenants[0].TenantID
		}
	}

	authTime := time.Now()
	// L7：amr 还原会话创建时的原始认证方法（如 ["pwd"]），不再使用非标准的 "sso"。
	amr := svc.ssoSessionStore.SessionAMR(reqCtx, sessionID)
	if len(amr) == 0 {
		amr = []string{"pwd"}
	}
	if err := svc.provider.Storage.CompleteAuthRequest(reqCtx, authRequestID, buildOIDCSubject(personID), authTime, amr, "", tenantID, true); err != nil {
		return "", err
	}

	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionLogin,
		TenantID:   tenantID,
		Result:     "success",
		TargetType: "person",
		TargetID:   personID,
	})

	if aErr := svc.provider.Storage.AssociateSession(reqCtx, authRequestID, sessionID); aErr != nil {
		glog.Warnf(ctx, "[oidcAuthSvc.CompleteLoginBySession] associate session fail, err:%v, authRequestID:%s, sessionID:%s", aErr, authRequestID, sessionID)
	}

	return svc.provider.BuildAuthCallbackURL(reqCtx, authRequestID), nil
}

// sessionAuditContext 将已解析的租户写入 context，供 CreateSession 落库 session 审计时读取 tenant_id。
func sessionAuditContext(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, sso.ContextKeyTenantID, tenantID)
}

func clientIDFromAuthRequest(authReq op.AuthRequest) string {
	if ar, ok := authReq.(*AuthRequest); ok {
		return ar.GetClientID()
	}
	return ""
}

// resolveAllowPersonCreateTenant reports whether the app backing the oauth client
// allows a zero-tenant person to self-create a tenant. Person with >=1 tenant => false.
func (svc *oidcAuthSvc) resolveAllowPersonCreateTenant(ctx *gin.Context, clientID string, tenantCount int) bool {
	if clientID == "" || tenantCount > 0 || svc.applicationClientDao == nil || svc.applicationDao == nil {
		return false
	}
	client, err := svc.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID})
	if err != nil || client == nil || client.AppID == "" {
		return false
	}
	app, err := svc.applicationDao().GetByID(ctx, client.AppID)
	if err != nil || app == nil || app.ID == "" {
		return false
	}
	var policy model.TenantPolicy
	if len(app.TenantPolicy) == 0 {
		return false
	}
	if err := json.Unmarshal(app.TenantPolicy, &policy); err != nil || policy.AllowPersonCreateTenant == nil {
		return false
	}
	return *policy.AllowPersonCreateTenant
}
