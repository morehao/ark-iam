package iam

import (
	"fmt"

	pkconfig "github.com/morehao/ark-iam/pkg/config"
)

// validateNonDevSecurityConfig 非 dev 环境强制安全配置（fail-closed）。等价于现网
// apps/auth 的启动校验，迁入 iam 使内嵌形态同样受保护：
//   - cookieSecure 必须 true（HTTPS SSO cookie）
//   - allowInsecure 必须 false
//   - 必须配置 OIDC 签名私钥（不允许 dev 自动生成被误部署到生产）
//   - 必须配置 encryptionKey
func validateNonDevSecurityConfig(conf *pkconfig.Config, env string) error {
	if conf == nil || env == "dev" || env == "" {
		return nil
	}
	o := conf.OIDC
	if !o.CookieSecure {
		return fmt.Errorf("iam: oidc.cookieSecure must be true in non-dev environment")
	}
	if o.AllowInsecure {
		return fmt.Errorf("iam: oidc.allowInsecure must be false in non-dev environment")
	}
	if o.SigningPrivateKeyPath == "" && o.SigningPrivateKeyPEM == "" && len(o.EffectiveSigningKeys()) == 0 {
		return fmt.Errorf("iam: oidc signing key must be configured in non-dev environment")
	}
	if o.EncryptionKey == "" {
		return fmt.Errorf("iam: oidc encryptionKey must be configured in non-dev environment")
	}
	return nil
}
