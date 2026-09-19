package oidckit

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/sdk/rp"

	"github.com/morehao/golib/glog"
)

// KeySource 是按 kid 取公钥的最小接口（真源 sdk/rp.KeySource）。
//
// 全仓只保留这一个别名：back-channel logout 接收端与 OIDC 鉴权中间件都用它，
// 避免出现两个方法集相同却各自命名的重复别名。
type KeySource = rp.KeySource

// NewKeySourceFromConfig 依据配置构造 OP 公钥来源，供内置应用（RP）校验 access token。
//
// 取值优先级：
//  1. OIDC.JWKSURL（显式端点）或由 issuer 解析出的 JWKS 端点——
//     issuer 形态走「标准 discovery 的 jwks_uri → {issuer}/keys →
//     {issuer}/.well-known/jwks.json」，是本仓 OP 与外部 OP 都适用的
//     "可轮换"来源，支持未知 kid 限速刷新；生产多进程部署应走这条；
//  2. 本地配置的签名密钥（signingPrivateKeyPath / signingPrivateKeyPEM / oidc.keys）
//     ——导出全部公钥构造进程内 key set。**注意：这是启动时快照，OP 轮换后需要
//     重启本进程**，因此它只作为"没有可用的 JWKS 端点时的兜底"，不是推荐路径。
//
// 两者都没有时返回 error（调用方应 fail-closed，不得退化为不校验）。
func NewKeySourceFromConfig(conf *config.Config) (KeySource, error) {
	if conf != nil && strings.TrimSpace(conf.OIDC.JWKSURL) != "" {
		return rp.NewKeys(context.Background(), strings.TrimSpace(conf.OIDC.JWKSURL))
	}
	if conf != nil && conf.OIDC.Issuer != "" {
		// issuer 形如 {op}/oidc：JWKS 端点由 SDK 解析（discovery 优先，
		// 本仓 OP 的 jwks_uri = {issuer}/keys），并同步预取（失败即 fail-fast，
		// 不静默降级为不校验）。
		return rp.NewKeys(context.Background(), conf.OIDC.Issuer)
	}
	if keys := localPublicKeys(conf); len(keys) > 0 {
		glog.Warnf(context.Background(), "[oidckit.NewKeySourceFromConfig] 使用本地签名密钥快照作为 JWKS 来源：OP 轮换签名密钥后本进程需重启才会生效；生产建议配置 oidc.jwksURL")
		return rp.NewKeysFromSet(keys), nil
	}
	return nil, errors.New("oidc key source unavailable: set oidc.jwksURL, oidc.issuer, or local signing key")
}

// ResolveKeySource 返回本应用可用的 OP 公钥来源。
//
// injected 非空时直接使用：**gateway 单体部署**下 auth 的 OP 把自己的已发布公钥
// 作为进程内 key set 注入同进程的其它应用（platformadmin/tenantadmin/rpapi），
// 零网络调用、自动跟随多 key 轮换，也避免"进程启动期 HTTP 请求自己还没监听的
// JWKS 端点"这种必然失败的自举。injected 为 nil 时按本应用配置自建
// （见 NewKeySourceFromConfig），此时才可能发生网络预取。
func ResolveKeySource(injected KeySource, conf *config.Config) (KeySource, error) {
	if injected != nil {
		return injected, nil
	}
	return NewKeySourceFromConfig(conf)
}

// localPublicKeys 从本地配置的签名密钥（单 key 或 keys 列表）导出全部公钥。
func localPublicKeys(conf *config.Config) map[string]*rsa.PublicKey {
	if conf == nil {
		return nil
	}
	out := map[string]*rsa.PublicKey{}
	for i, item := range conf.OIDC.EffectiveSigningKeys() {
		var pemData []byte
		switch {
		case item.PrivateKeyPEM != "":
			pemData = []byte(item.PrivateKeyPEM)
		case item.PrivateKeyPath != "":
			data, err := os.ReadFile(item.PrivateKeyPath)
			if err != nil {
				continue
			}
			pemData = data
		default:
			continue
		}
		key, err := parseRSAPrivateKeyPEM(pemData)
		if err != nil {
			glog.Warnf(context.Background(), "[oidckit.localPublicKeys] parse signing key fail, index:%d, err:%v", i, err)
			continue
		}
		kid := item.Kid
		if kid == "" {
			kid = conf.OIDC.SigningKeyID
		}
		if kid == "" {
			// 与 auth 侧一致的 kid 派生（sha256(N)[:16] → base64url）：
			// kid 必须与 OP 实际签发的 kid 完全一致，否则全部请求 401。
			kid = deriveKeyID(&key.PublicKey)
		}
		out[kid] = &key.PublicKey
	}
	return out
}

// deriveKeyID 与 auth 侧 svcoidc.DeriveKeyID 保持同一算法（此处重复实现以避免
// pkg → apps 的反向依赖）。改动其一必须同步另一处：pkg 侧由 keys_test.go 的
// TestDeriveKeyID 锁定，apps 侧由 svcoidc 的 kid 派生测试锁定（同一 golden 口径）。
func deriveKeyID(pub *rsa.PublicKey) string {
	if pub == nil {
		return ""
	}
	sum := sha256.Sum256(pub.N.Bytes())
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}

// parseRSAPrivateKeyPEM 宽容解析 RSA 私钥 PEM（PKCS#8 优先、PKCS#1 兜底）。
func parseRSAPrivateKeyPEM(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("invalid RSA private key PEM")
	}
	if block.Type != "RSA PRIVATE KEY" && block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
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
