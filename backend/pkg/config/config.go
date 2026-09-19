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

// SigningKeyConfig 是 OP 的一把签名密钥配置（最小多 key：列表 + active 标记）。
//
// 语义（见设计文档关键决策五）：
//   - BackChannelLogoutURI 之外的全部公钥都会经 /oidc/keys 发布，SDK 侧按 kid 取键，
//     因此"追加 → 切 active → 等 ≥2×TTL → 摘除"可实现例行零中断轮换；
//   - 紧急轮换（疑似泄露）= 追加新 key + 切 active + **立即删除旧 key 条目**。
type SigningKeyConfig struct {
	// Kid 是密钥标识（JWKS 的 kid）。留空时按公钥派生（RFC 7638 风格 thumbprint），
	// 避免"换 key 忘改 kid"导致验签方拿到错误公钥。
	Kid string `yaml:"kid"`
	// PrivateKeyPath 是 RSA 私钥 PEM 文件路径（PKCS#1 或 PKCS#8）。
	PrivateKeyPath string `yaml:"privateKeyPath"`
	// PrivateKeyPEM 是内联的 RSA 私钥 PEM（与 PrivateKeyPath 二选一，PEM 优先）。
	PrivateKeyPEM string `yaml:"privateKeyPEM"`
	// Active 标记该 key 是否为当前签发用 key；同一时刻必须恰好一个 active。
	Active bool `yaml:"active"`
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
	// Keys 是签名密钥列表（最小多 key）。配置后优先于上面三个单 key 字段；
	// 单 key 字段保留是为了零破坏迁移（等价于 Keys 只有一项且 Active=true）。
	Keys            []SigningKeyConfig `yaml:"keys"`
	EncryptionKey   string             `yaml:"encryptionKey"`
	EncryptionKeyID string             `yaml:"encryptionKeyID"`
	AllowInsecure   bool               `yaml:"allowInsecure"`
	AuthRequestTTL  int                `yaml:"authRequestTTL"`
	AuthCodeTTL     int                `yaml:"authCodeTTL"`
	SpentCodeTTL    int                `yaml:"spentCodeTTL"`
	SessionTTL      int                `yaml:"sessionTTL"`
	// EnableSSOSessionValidation 控制业务应用（RP）是否在每次请求时校验
	// 用户的 SSO 中心会话活性（HasActiveSession）。开启后"一处登出、处处登出"
	// 在请求粒度即时生效；要求业务应用与 auth 共享同一认证 Redis。
	EnableSSOSessionValidation bool `yaml:"enableSSOSessionValidation"`
	// BackChannelLogoutPath 是本应用挂载 back-channel logout 接收端的基础路径
	// （默认 /oidc/bc-logout）。
	BackChannelLogoutPath string `yaml:"backChannelLogoutPath"`
	// JWKSURL 是 RP 侧（含内置应用）获取 OP 公钥的**显式端点**（末段为 keys/jwks
	// 或以 .json 结尾）。留空时按 issuer 解析：标准 discovery 的 jwks_uri →
	// `{issuer}/keys` → `{issuer}/.well-known/jwks.json`（本仓 OP 发布在 `{issuer}/keys`）；
	// gateway 单体部署下由 auth 注入进程内 key set，完全不产生网络调用。
	JWKSURL string `yaml:"jwksURL"`
	// Audiences 是本应用接受的 aud 白名单（按应用声明，不硬编码 client_id）。
	// 留空表示不校验 aud（如 auth 的 /v1/auth/*：它服务的是多个内置控制台客户端）。
	// 注意是 per-app 的：gateway 聚合某应用时，gateway 配置也须为该应用声明同样的白名单。
	Audiences []string `yaml:"audiences"`
}

// EffectiveSigningKeys 返回生效的签名密钥列表：
// 配置了 keys 时用它；否则把单 key 三元组归一为"只有一项且 active"的列表。
func (o OIDC) EffectiveSigningKeys() []SigningKeyConfig {
	if len(o.Keys) > 0 {
		out := make([]SigningKeyConfig, len(o.Keys))
		copy(out, o.Keys)
		return out
	}
	if o.SigningPrivateKeyPath == "" && o.SigningPrivateKeyPEM == "" {
		return nil
	}
	return []SigningKeyConfig{{
		Kid:            o.SigningKeyID,
		PrivateKeyPath: o.SigningPrivateKeyPath,
		PrivateKeyPEM:  o.SigningPrivateKeyPEM,
		Active:         true,
	}}
}

// ActiveSigningKey 返回 active 密钥；未配置或没有 active 时返回 nil。
func (o OIDC) ActiveSigningKey() *SigningKeyConfig {
	keys := o.EffectiveSigningKeys()
	for i := range keys {
		if keys[i].Active {
			return &keys[i]
		}
	}
	return nil
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
	// TrustedProxies 是可信反向代理的 CIDR 列表（如 ["10.0.0.0/8"]）。
	// 配置后 gin 仅从这些代理透传的 X-Forwarded-For 取客户端 IP；
	// 未配置时默认不信任任何代理（直接使用 RemoteAddr），
	// 防止客户端伪造 X-Forwarded-For 绕过按 IP 维度的限流/登录锁定。
	TrustedProxies []string `yaml:"trustedProxies"`
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
