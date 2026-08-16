package config

import (
	"net/http"
	"strings"

	"github.com/morehao/golib/dbaccess/dbes"
	"github.com/morehao/golib/dbaccess/dbgorm"
	"github.com/morehao/golib/dbaccess/dbredis"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gtrace"
	"github.com/morehao/golib/protocol/ghttp"
)

type Config struct {
	Server      Server                    `yaml:"server"`
	DB          DBConfig                  `yaml:"db"`
	Log         map[string]glog.LogConfig `yaml:"log"`
	Trace       gtrace.TraceConfig        `yaml:"trace"`
	DBConfigs   []dbgorm.Config           `yaml:"db_configs"`
	RedisConfig dbredis.RedisConfig       `yaml:"redis_config"`
	ESConfigs   []dbes.ESConfig           `yaml:"es_configs"`
	Client      Client                    `yaml:"client"`
	JWT         JWT                       `yaml:"jwt"`
	OIDC        OIDC                      `yaml:"oidc"`
	Password    PasswordConfig            `yaml:"password"`
	Security    SecurityConfig            `yaml:"security"`
	MasterKey   string                    `yaml:"masterKey"`
}

type SecurityConfig struct {
	Login LoginGuardConfig `yaml:"login"`
}

type LoginGuardConfig struct {
	MaxFailures int `yaml:"maxFailures"`
	WindowSec   int `yaml:"windowSec"`
	LockSec     int `yaml:"lockSec"`
	// RateLimit 登录接口频率限流（按 IP），每分钟允许的请求数与突发容量。
	// 未配置时使用默认值（如 30/min、burst 10），见 middleware.LoginRateLimit。
	RatePerMinute int `yaml:"ratePerMinute"`
	Burst         int `yaml:"burst"`
}

type OIDC struct {
	Issuer           string `yaml:"issuer"`
	FrontendLoginURL string `yaml:"frontendLoginURL"`
	CookieDomain     string `yaml:"cookieDomain"`
	// CookieSecure 控制 SSO 会话 cookie 的 Secure 标志。生产环境（HTTPS）必须为 true。
	CookieSecure bool `yaml:"cookieSecure"`
	// CookieSameSite 控制 SSO 会话 cookie 的 SameSite 属性，取值 lax/strict/none。
	// 默认 lax；跨站（不同站点间 SSO）场景需 none（且 CookieSecure 必须为 true）。
	CookieSameSite        string `yaml:"cookieSameSite"`
	SigningKeyID          string `yaml:"signingKeyID"`
	SigningPrivateKeyPath string `yaml:"signingPrivateKeyPath"`
	SigningPrivateKeyPEM  string `yaml:"signingPrivateKeyPEM"`
	EncryptionKey         string `yaml:"encryptionKey"`
	EncryptionKeyID       string `yaml:"encryptionKeyID"`
	AllowInsecure         bool   `yaml:"allowInsecure"`
	AuthRequestTTL        int    `yaml:"authRequestTTL"`
	AuthCodeTTL           int    `yaml:"authCodeTTL"`
	SpentCodeTTL          int    `yaml:"spentCodeTTL"`
	SessionTTL            int    `yaml:"sessionTTL"`
	// EnableSSOSessionValidation 控制业务应用（RP）是否在每次请求时校验
	// 用户的 SSO 中心会话活性（HasActiveSession）。开启后"一处登出、处处登出"
	// 在请求粒度即时生效；要求业务应用与 auth 共享同一认证 Redis。
	EnableSSOSessionValidation bool `yaml:"enableSSOSessionValidation"`
	// BackChannelLogoutPath 是本应用挂载 back-channel logout 接收端的基础路径
	// （默认 /oidc/bc-logout）。
	BackChannelLogoutPath string `yaml:"backChannelLogoutPath"`
}

func (o OIDC) SSOCookieDomain() string {
	return o.CookieDomain
}

// CookieSameSiteMode 把配置字符串转换为 http.SameSite；未配置或非法值默认 Lax。
func (o OIDC) CookieSameSiteMode() http.SameSite {
	switch strings.ToLower(o.CookieSameSite) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

type PasswordConfig struct {
	Prefix string `yaml:"prefix"`
}

type JWT struct {
	SignKey string `yaml:"signKey"`
}

type Server struct {
	Name string `yaml:"name"` // 服务名称
	Port string `yaml:"port"` // 服务端口
	Env  string `yaml:"env"`  // 环境变量
}

// DBConfig 数据库启动行为配置。
type DBConfig struct {
	// AutoMigrate 是否在启动时基于 GORM AutoMigrate 自动创建/同步数据表（幂等）。
	AutoMigrate bool `yaml:"auto_migrate"`
	// Seed 是否在启动时幂等写入基础种子数据（租户/角色/权限/菜单/管理员/应用客户端）。
	Seed bool `yaml:"seed"`
}

type Client struct {
	HTTPBingo *ghttp.Client `yaml:"httpbingo"`
}
