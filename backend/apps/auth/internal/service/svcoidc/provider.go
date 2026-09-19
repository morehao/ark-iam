package svcoidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/core/oidcop"
	"github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/text/language"
)

const (
	pathLoggedOut = "/oidc/logged-out"
	// defaultIssuerPort 未配置 issuer 时推导本地 OP 地址使用的端口。
	defaultIssuerPort = "8099"
)

type OIDCProvider struct {
	Provider *op.Provider
	Storage  *oidcop.OIDCStorage
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
// middleware.LoadSigningPublicKey 的宽容解析保持一致。
func isSupportedRSAPrivateKeyBlock(blockType string) bool {
	return blockType == "RSA PRIVATE KEY" || blockType == "PRIVATE KEY"
}

// LoadedSigningKeys 是一次加载结果：当前 active 私钥 + 需对外发布的全部公钥。
type LoadedSigningKeys struct {
	// ActiveKey / ActiveKeyID 为当前签发用密钥（恰好一个）。
	ActiveKey   *rsa.PrivateKey
	ActiveKeyID string
	// Published 是需经 /oidc/keys 发布的全部公钥（kid → 公钥），含过渡期旧 key。
	Published map[string]*rsa.PublicKey
}

// loadSigningKeys 加载**全部**配置的签名密钥（最小多 key）。
//
// 配置来源：`oidc.keys` 列表（kid + privateKeyPath/PEM + active）；未配置列表时
// 回退到单 key 三元组（等价于只有一项且 active），保证零破坏迁移。
//
// 规则：
//   - 列表内 kid 必须唯一且非空（空 kid 由公钥派生 thumbprint 补齐），
//     否则"换 key 忘改 kid"会让校验方按 kid 取到错误公钥；
//   - 恰好一个 active；0 个或多个都视为配置错误（fail-closed，不猜测）；
//   - 非 dev 环境缺 key 一律 fail-closed，不自动生成（避免重启导致 token 全量失效）。
func loadSigningKeys() (*LoadedSigningKeys, error) {
	if appconfig.Conf == nil {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		kid := DeriveKeyID(&privateKey.PublicKey)
		return &LoadedSigningKeys{
			ActiveKey:   privateKey,
			ActiveKeyID: kid,
			Published:   map[string]*rsa.PublicKey{kid: &privateKey.PublicKey},
		}, nil
	}
	return loadSigningKeysFromConfig(&appconfig.Conf.OIDC)
}

func loadSigningKeysFromConfig(cfg *config.OIDC) (*LoadedSigningKeys, error) {
	configured := cfg.EffectiveSigningKeys()
	if len(configured) == 0 {
		if !isDevEnv() {
			return nil, fmt.Errorf("oidc signing key not configured (set oidc.keys or signingPrivateKeyPath/signingPrivateKeyPEM)")
		}
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		kid := DeriveKeyID(&privateKey.PublicKey)
		return &LoadedSigningKeys{
			ActiveKey:   privateKey,
			ActiveKeyID: kid,
			Published:   map[string]*rsa.PublicKey{kid: &privateKey.PublicKey},
		}, nil
	}
	result := &LoadedSigningKeys{Published: make(map[string]*rsa.PublicKey, len(configured))}
	for i := range configured {
		item := configured[i]
		privateKey, err := loadSingleSigningKey(cfg, &item, i)
		if err != nil {
			return nil, err
		}
		kid := item.Kid
		if kid == "" {
			kid = DeriveKeyID(&privateKey.PublicKey)
		}
		if _, dup := result.Published[kid]; dup {
			return nil, fmt.Errorf("oidc.keys[%d]: duplicate kid %q (kid 必须唯一，否则验签方会取到错误公钥)", i, kid)
		}
		result.Published[kid] = &privateKey.PublicKey
		if item.Active {
			if result.ActiveKey != nil {
				return nil, fmt.Errorf("oidc.keys: multiple active keys (kid %q and %q)", result.ActiveKeyID, kid)
			}
			result.ActiveKey = privateKey
			result.ActiveKeyID = kid
		}
	}
	if result.ActiveKey == nil {
		return nil, fmt.Errorf("oidc.keys: no active signing key (exactly one entry must set active: true)")
	}
	return result, nil
}

// loadSingleSigningKey 解析单个 key 条目；keyIndex 仅用于报错定位。
func loadSingleSigningKey(cfg *config.OIDC, item *config.SigningKeyConfig, keyIndex int) (*rsa.PrivateKey, error) {
	switch {
	case item.PrivateKeyPEM != "":
		return parseRSAPrivateKeyPEM([]byte(item.PrivateKeyPEM), fmt.Sprintf("oidc.keys[%d].privateKeyPEM", keyIndex))
	case item.PrivateKeyPath != "":
		pemData, err := os.ReadFile(item.PrivateKeyPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			// H4：非 dev 环境显式配置了路径但文件缺失时 fail-closed——
			// 自动生成新密钥会静默更换 kid，导致所有已签发 token 失效且各 RP 公钥失同步。
			if !isDevEnv() {
				return nil, fmt.Errorf("signing private key file not found: %s (refusing to auto-generate in non-dev)", item.PrivateKeyPath)
			}
			privateKey, genErr := rsa.GenerateKey(rand.Reader, 2048)
			if genErr != nil {
				return nil, fmt.Errorf("failed to generate signing key: %w", genErr)
			}
			encoded := pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
			})
			if keyDir := filepath.Dir(item.PrivateKeyPath); keyDir != "." {
				if mkErr := os.MkdirAll(keyDir, 0755); mkErr != nil {
					return nil, fmt.Errorf("failed to create key directory: %w", mkErr)
				}
			}
			if wErr := os.WriteFile(item.PrivateKeyPath, encoded, 0600); wErr != nil {
				return nil, fmt.Errorf("failed to write signing key: %w", wErr)
			}
			return privateKey, nil
		}
		return parseRSAPrivateKeyPEM(pemData, item.PrivateKeyPath)
	default:
		return nil, fmt.Errorf("oidc.keys[%d]: neither privateKeyPath nor privateKeyPEM set", keyIndex)
	}
}

// parseRSAPrivateKeyPEM 宽容解析 RSA 私钥 PEM（PKCS#8 优先、PKCS#1 兜底）。
func parseRSAPrivateKeyPEM(pemData []byte, source string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || !isSupportedRSAPrivateKeyBlock(block.Type) {
		return nil, fmt.Errorf("invalid RSA private key PEM: %s", source)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		parsed, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
		}
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA: %T", parsed)
	}
	return rsaKey, nil
}

// DeriveKeyID 由公钥派生稳定的 kid（RFC 7638 JWK thumbprint 风格，base64url(sha256(n || e))）。
// 供"未显式配置 kid"时使用，避免换 key 忘改 kid。
func DeriveKeyID(pub *rsa.PublicKey) string {
	if pub == nil {
		return ""
	}
	sum := sha256.Sum256(pub.N.Bytes())
	return base64.RawURLEncoding.EncodeToString(sum[:16])
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

// protocolStateTTLOptions 将应用配置的协议态 TTL 转为 oidcop 选项；未配置时使用 oidcop 默认值。
func protocolStateTTLOptions() []oidcop.ProtocolStateOption {
	if appconfig.Conf == nil {
		return nil
	}
	var opts []oidcop.ProtocolStateOption
	if v := appconfig.Conf.OIDC.AuthRequestTTL; v > 0 {
		opts = append(opts, oidcop.WithAuthRequestTTL(time.Duration(v)*time.Second))
	}
	if v := appconfig.Conf.OIDC.AuthCodeTTL; v > 0 {
		opts = append(opts, oidcop.WithAuthCodeTTL(time.Duration(v)*time.Second))
	}
	if v := appconfig.Conf.OIDC.SpentCodeTTL; v > 0 {
		opts = append(opts, oidcop.WithSpentCodeTTL(time.Duration(v)*time.Second))
	}
	return opts
}

// resolveIssuer 从应用配置解析 OP issuer；未配置时按本地端口推导（缺省 8099）。
func resolveIssuer() string {
	if appconfig.Conf == nil {
		return fmt.Sprintf("http://localhost:%s/oidc", defaultIssuerPort)
	}
	if issuer := appconfig.Conf.OIDC.Issuer; issuer != "" {
		return issuer
	}
	port := appconfig.Conf.Server.Port
	if port == "" {
		port = defaultIssuerPort
	}
	return fmt.Sprintf("http://localhost:%s/oidc", port)
}

// SetupOIDCProvider 按应用配置自装配 OP provider：issuer 解析、签名/加密密钥、
// 协议态 storage 与 op.Provider。
func SetupOIDCProvider() (*OIDCProvider, error) {
	issuer := resolveIssuer()
	keys, err := loadSigningKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to load OIDC signing key: %w", err)
	}

	storage := oidcop.NewOIDCStorage(
		oidcop.NewRedisProtocolStateStore(protocolStateTTLOptions()...),
		oidcop.NewPersistentStore(oidcop.WithIssuer(issuer)),
		keys.ActiveKey, keys.ActiveKeyID, keys.Published,
	)

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

// SigningKey 返回 OP 的签名私钥与 keyID（供 back-channel logout 发送器使用）。
func (p *OIDCProvider) SigningKey(ctx context.Context) (*rsa.PrivateKey, string, error) {
	if p == nil || p.Storage == nil {
		return nil, "", fmt.Errorf("oidc provider storage not initialized")
	}
	signingKey, err := p.Storage.SigningKey(ctx)
	if err != nil {
		return nil, "", err
	}
	privKey, ok := signingKey.Key().(*rsa.PrivateKey)
	if !ok {
		return nil, "", fmt.Errorf("oidc signing key is not RSA private key: %T", signingKey.Key())
	}
	return privKey, signingKey.ID(), nil
}

// StartLogoutWorker 启动 back-channel logout 发送器（异步消费登出队列，发送 logout_token）。
func (p *OIDCProvider) StartLogoutWorker(ctx context.Context) error {
	privKey, keyID, err := p.SigningKey(ctx)
	if err != nil {
		return err
	}
	worker := oidcop.NewLogoutWorker(privKey, keyID, p.issuer)
	go worker.Run(ctx)
	return nil
}

// PublicKey 返回 OP 签名公钥，供业务路由鉴权中间件校验本 OP 签发的 token。
func (p *OIDCProvider) PublicKey() (*rsa.PublicKey, error) {
	privKey, _, err := p.SigningKey(context.Background())
	if err != nil {
		return nil, err
	}
	return &privKey.PublicKey, nil
}

func (p *OIDCProvider) BuildAuthCallbackURL(ctx context.Context, authRequestID string) string {
	return op.AuthCallbackURL(p.Provider)(op.ContextWithIssuer(ctx, p.issuer), authRequestID)
}

// RedirectURIVerifier 返回基于 provider 客户端注册的回调地址校验器，
// 供静默登录中间件在 prompt=none 失败跳回 redirect_uri 前校验其确为该 client 注册的回调地址（L1）。
// provider 或其存储不可用（如单元测试中的空 provider）时返回 nil，跳过校验保持兼容。
func (p *OIDCProvider) RedirectURIVerifier() func(ctx *gin.Context, clientID, redirectURI string) bool {
	if p == nil || p.Storage == nil {
		return nil
	}
	storage := p.Storage
	return func(ctx *gin.Context, clientID, redirectURI string) bool {
		if clientID == "" || redirectURI == "" {
			return false
		}
		// client_id 全局唯一，且本校验发生在「租户确定之前」的授权请求入口：
		// 必须显式声明「全租户」作用域，跨租户可见性不能来自缺失作用域。
		client, err := storage.GetClientByClientID(dbclient.CrossTenantContext(ctx), clientID)
		if err != nil {
			return false
		}
		for _, u := range client.RedirectURIs() {
			if u == redirectURI {
				return true
			}
		}
		return false
	}
}
