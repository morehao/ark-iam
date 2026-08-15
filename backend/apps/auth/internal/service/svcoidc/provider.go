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
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	appconfig "github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/oidcop"
	"github.com/zitadel/oidc/v3/pkg/op"
	"golang.org/x/text/language"
)

const (
	pathLoggedOut = "/oidc/logged-out"
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

func SetupOIDCProvider(issuer string) (*OIDCProvider, error) {
	privateKey, keyID, err := loadSigningKey()
	if err != nil {
		return nil, fmt.Errorf("failed to load OIDC signing key: %w", err)
	}

	storage := oidcop.NewOIDCStorage(
		oidcop.NewRedisProtocolStateStore(protocolStateTTLOptions()...),
		oidcop.NewPersistentStore(oidcop.WithIssuer(issuer)),
		privateKey, keyID,
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
		client, err := storage.GetClientByClientID(ctx.Request.Context(), clientID)
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
