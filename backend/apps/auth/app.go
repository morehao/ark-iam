package auth

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/router"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/golib/biz/gserver/gindocs"
)

const AppName = "auth"

func Init(engine *gin.Engine, Conf *pkgconfig.Config) {
	config.Conf = Conf

	// H15：非 dev 环境强制安全配置，否则拒绝启动（fail-closed，与签名密钥策略对齐）
	if err := validateNonDevSecurityConfig(); err != nil {
		panic(fmt.Sprintf("[auth.Init] security config validation fail: %v", err))
	}

	if config.Conf.Server.Env == "dev" {
		gindocs.Register(engine.Group("/"+AppName), AppName)
	}

	// 全部业务对象（OIDC provider、控制器、服务）由 router 层自装配，
	// app 层只做基础设施装配（配置、docs、引擎）。
	router.RegisterRouter(engine)
}

// validateNonDevSecurityConfig 非 dev 环境强制：HTTPS 场景 cookieSecure=true、
// allowInsecure=false、签名/加密密钥已配置，防止 dev 宽松配置被误部署到生产。
func validateNonDevSecurityConfig() error {
	if config.Conf == nil || config.Conf.Server.Env == "dev" || config.Conf.Server.Env == "" {
		return nil
	}
	if !config.Conf.OIDC.CookieSecure {
		return fmt.Errorf("oidc.cookieSecure must be true in non-dev environment (HTTPS SSO cookie)")
	}
	if config.Conf.OIDC.AllowInsecure {
		return fmt.Errorf("oidc.allowInsecure must be false in non-dev environment")
	}
	if config.Conf.OIDC.SigningPrivateKeyPath == "" && config.Conf.OIDC.SigningPrivateKeyPEM == "" {
		return fmt.Errorf("oidc signing key must be configured in non-dev environment")
	}
	if config.Conf.OIDC.EncryptionKey == "" {
		return fmt.Errorf("oidc encryptionKey must be configured in non-dev environment")
	}
	return nil
}
