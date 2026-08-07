package svcoidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/iam/config"
	"github.com/morehao/ark-iam/iam/internal/dto/dtooidc"
	"github.com/morehao/ark-iam/iam/internal/service/svcauth"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/code"
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
		if block == nil || block.Type != "RSA PRIVATE KEY" {
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
		if block == nil || block.Type != "RSA PRIVATE KEY" {
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

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", err
	}
	return privateKey, "auto-key", nil
}

func loadEncryptionKey() ([32]byte, string) {
	encKeyStr := "test-key-32-bytes-for-oidc-encrp"
	encKeyID := "enc-key-1"
	if appconfig.Conf != nil {
		if appconfig.Conf.OIDC.EncryptionKey != "" {
			encKeyStr = appconfig.Conf.OIDC.EncryptionKey
		}
		if appconfig.Conf.OIDC.EncryptionKeyID != "" {
			encKeyID = appconfig.Conf.OIDC.EncryptionKeyID
		}
	}
	return sha256.Sum256([]byte(encKeyStr)), encKeyID
}

func SetupOIDCProvider(issuer string) (*OIDCProvider, error) {
	privateKey, keyID, err := loadSigningKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load OIDC signing key: %w", err)
	}

	storage := NewOIDCStorage(NewRedisProtocolStateStore(), NewPersistentStore(), privateKey, keyID)

	encKey, encKeyID := loadEncryptionKey()

	opConfig := &op.Config{
		CryptoKey:                encKey,
		CryptoKeyId:              encKeyID,
		DefaultLogoutRedirectURI: pathLoggedOut,
		CodeMethodS256:           true,
		AuthMethodPost:           true,
		GrantTypeRefreshToken:    true,
		SupportedUILocales:       []language.Tag{language.Chinese, language.English},
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
	SelectTenant(ctx context.Context, authRequestID string, tenantID uint) (*dtooidc.OIDCLoginResp, error)
	CompleteLoginBySession(ctx context.Context, authRequestID string, sessionID string) (string, error)
}

type passwordAuthenticator interface {
	AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error)
	TenantsForPerson(ctx *gin.Context, personID uint) ([]objauth.TenantOption, error)
}

type oidcAuthSvc struct {
	provider        *OIDCProvider
	authSvc         passwordAuthenticator
	ssoSessionStore SSOSessionStore
}

func NewOIDCAuthSvc(provider *OIDCProvider) OIDCAuthSvc {
	return &oidcAuthSvc{
		provider:        provider,
		authSvc:         svcauth.NewAuthSvc(),
		ssoSessionStore: NewSSOSessionStore(),
	}
}

func (svc *oidcAuthSvc) CompleteLogin(ctx *gin.Context, req *dtooidc.OIDCLoginReq) (*dtooidc.OIDCLoginResp, error) {
	if _, err := svc.provider.Storage.AuthRequestByID(ctx.Request.Context(), req.AuthRequestID); err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	personEntity, userEntity, tenants, err := svc.authSvc.AuthenticatePassword(ctx, req.Identifier, req.Password)
	if err != nil {
		return nil, err
	}
	authTime := time.Now()
	subject := buildOIDCSubject(personEntity.ID)
	// 多租户：暂不 done、不发 code，需用户先选租户（SSO 会话 defer 到 selectTenant 完成后再建）
	if len(tenants) > 1 {
		if err := svc.provider.Storage.CompleteAuthRequest(req.AuthRequestID, subject, authTime, []string{"pwd"}, "", 0, false); err != nil {
			return nil, code.GetError(code.OIDCSessionNotFound)
		}
		return &dtooidc.OIDCLoginResp{
			RequiresTenantSelection: true,
			Tenants:                 tenants,
		}, nil
	}
	// 单租户：自动选租户，done，发 code，并建 SSO 会话
	tenantID := userEntity.TenantID
	if err := svc.provider.Storage.CompleteAuthRequest(req.AuthRequestID, subject, authTime, []string{"pwd"}, "", tenantID, true); err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}

	resp := &dtooidc.OIDCLoginResp{
		ContinueURL: svc.provider.BuildAuthCallbackURL(ctx.Request.Context(), req.AuthRequestID),
		TenantID:    tenantID,
		Tenants:     tenants,
	}

	if svc.ssoSessionStore != nil {
		sessionID, err := svc.ssoSessionStore.CreateSession(ctx.Request.Context(), personEntity.ID)
		if err != nil {
			glog.Warnf(ctx, "[oidcAuthSvc.CompleteLogin] failed to create sso session: %v", err)
		} else {
			resp.SessionID = sessionID
		}
	}

	return resp, nil
}

func (svc *oidcAuthSvc) SelectTenant(ctx context.Context, authRequestID string, tenantID uint) (*dtooidc.OIDCLoginResp, error) {
	authReq, err := svc.provider.Storage.AuthRequestByID(ctx, authRequestID)
	if err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	personID, perr := parseOIDCSubject(authReq.GetSubject())
	if perr != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	tenants, terr := svc.authSvc.TenantsForPerson(ginContextFromContext(ctx), personID)
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
	if err := svc.provider.Storage.CompleteAuthRequest(authRequestID, authReq.GetSubject(), authReq.GetAuthTime(), authReq.GetAMR(), "", tenantID, true); err != nil {
		return nil, code.GetError(code.OIDCSessionNotFound)
	}
	resp := &dtooidc.OIDCLoginResp{
		ContinueURL: svc.provider.BuildAuthCallbackURL(ctx, authRequestID),
		TenantID:    tenantID,
		Tenants:     tenants,
	}
	if svc.ssoSessionStore != nil {
		if sessionID, sErr := svc.ssoSessionStore.CreateSession(ctx, personID); sErr == nil {
			resp.SessionID = sessionID
		}
	}
	return resp, nil
}

func (svc *oidcAuthSvc) CompleteLoginBySession(ctx context.Context, authRequestID string, sessionID string) (string, error) {
	personID, err := svc.ssoSessionStore.ValidateSession(ctx, sessionID)
	if err != nil {
		return "", err
	}

	if _, err := svc.provider.Storage.AuthRequestByID(ctx, authRequestID); err != nil {
		return "", err
	}

	tenantID := uint(0)
	if tenants, tErr := svc.authSvc.TenantsForPerson(ginContextFromContext(ctx), personID); tErr == nil && len(tenants) > 0 {
		tenantID = tenants[0].TenantID
	}

	authTime := time.Now()
	if err := svc.provider.Storage.CompleteAuthRequest(authRequestID, buildOIDCSubject(personID), authTime, []string{"sso"}, "", tenantID, true); err != nil {
		return "", err
	}

	return svc.provider.BuildAuthCallbackURL(ctx, authRequestID), nil
}

func ginContextFromContext(ctx context.Context) *gin.Context {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(ctx)
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = req
	return ginCtx
}
