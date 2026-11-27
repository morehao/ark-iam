package appassembly

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	pkconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/model"
)

// MountConfig 注册 /config.json 运行时配置端点（**仅出运行期配置，不含任何静态资源/embed**）。
//
// 登录/管理台前端由 npm 包 @morehao/ark-iam-react 提供，Go 侧不再托管任何前端静态资源。
// /config.json 是前端启动时拉取的运行期配置来源（issuer/apiBase/oidcBase/installBase/
// loginPath/clientId），可能被「业务应用（独立 origin）」跨源读取，故放行 CORS（该端点只含
// 公开配置，不含任何密钥，回显请求 Origin 即可）。
//
// 应在 Mount 之后调用。basePath 是挂载前缀（如内嵌 /iam、根部署 ""）。
func (a *App) MountConfig(engine *gin.Engine, basePath string) error {
	engine.GET(joinPath(basePath, "/config.json"), func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}

		// issuer：优先取 Raw 配置的绝对值（多源部署时即 OP 的公网地址）；未配置时按请求
		// Host 派生为绝对地址（{scheme}://{host}{basePath}/oidc），避免下发空串导致前端无法完成 OIDC。
		issuer := ""
		if a.cfg.Raw != nil {
			issuer = a.cfg.Raw.OIDC.Issuer
		}
		if issuer == "" && a.cfg.Issuer != "" {
			issuer = a.cfg.Issuer
		}
		if issuer == "" {
			issuer = requestBaseURL(c) + joinPath(basePath, "/oidc")
		}
		// 命中 Host binding 时以 binding 的 issuer/clientId 为准（多 issuer；DG-MW-1 一请求一解析）。
		clientID := resolveConsoleClientID(c, a.cfg.Raw)
		if rr, ok := resolveFromContext(c); ok && rr != nil {
			if rr.Issuer != "" {
				issuer = rr.Issuer
			}
			if rr.ClientID != "" {
				clientID = rr.ClientID
			}
		}

		// enabledDomains：本实例实际挂载的业务域集合（G4）。用于前端控制台按域裁剪菜单/路由入口
		// （未挂载的域不下发，前端据此隐藏对应入口）。空 Domains 归一为全量（与 Mount 的解析口径一致）。
		enabledDomains := make([]string, 0, len(allDomains))
		for _, d := range resolveDomains(a.cfg.Domains) {
			enabledDomains = append(enabledDomains, string(d))
		}

		c.JSON(http.StatusOK, gin.H{
			"issuer":       issuer,
			"apiBase":      joinPath(basePath, "/v1"),
			"oidcBase":     joinPath(basePath, "/oidc"),
			"installBase":  joinPath(basePath, "/install"),
			"loginPath":    joinPath(basePath, "/login"),
			"callbackPath": "/auth/callback",
			// clientId：本控制台 SPA 的 OIDC client_id。两个内置控制台（平台/租户）是不同应用、
			// 各有独立 client_id 与回调白名单；单进程聚合部署时 /config.json 只有一份端点，
			// 故按请求来源 Referer/Origin 的 host 命中哪个控制台的 console 地址，返回对应 client_id。
			// 无匹配（内嵌单控制台 / 外部业务应用读取）回退平台管理台 client_id（历史行为）。
			"clientId": clientID,
			// enabledDomains：已挂载域白名单（G4）；缺省/全量时前端视为全部域可用。
			"enabledDomains": enabledDomains,
		})
	})
	return nil
}

// resolveConsoleClientID 按请求来源判定该返回哪个内置控制台的 OIDC client_id。
//
// 单进程聚合（gateway）下 /config.json 是唯一端点，但平台控制台（:4001，client=platform_admin_web）
// 与租户控制台（:4002，client=tenant_admin_web）是两个独立 SPA、各有独立 client_id 与回调白名单。
// 若恒返回 platform_admin_web，租户控制台会拿着平台 client 发起授权、回调地址不在白名单 → 登录失败。
//
// 判定：取 Referer（页面同源拉取时浏览器默认携带）或 Origin 的 scheme://host，与配置里两个控制台
// 回调白名单的 host 比对，命中租户控制台 → tenant_admin_web，命中平台控制台 → platform_admin_web；
// 都不命中（内嵌单控制台同源、外部业务应用跨源读取）回退 platform_admin_web（与历史一致）。
func resolveConsoleClientID(c *gin.Context, conf *pkconfig.Config) string {
	if conf != nil {
		src := requestSourceOrigin(c)
		if src != "" {
			if consoleOriginMatches(src, conf.OIDC.Consoles.TenantAdminWeb.RedirectURIs) {
				return model.SeedBuiltinClientTenantAdminWeb
			}
			if consoleOriginMatches(src, conf.OIDC.Consoles.PlatformAdminWeb.RedirectURIs) {
				return model.SeedBuiltinClientPlatformAdminWeb
			}
		}
	}
	return model.SeedBuiltinClientPlatformAdminWeb
}

// requestSourceOrigin 从 Referer 或 Origin 头解析出 scheme://host（去掉路径）。
func requestSourceOrigin(c *gin.Context) string {
	candidate := c.GetHeader("Origin")
	if candidate == "" {
		candidate = c.GetHeader("Referer")
	}
	if candidate == "" {
		return ""
	}
	if u, err := url.Parse(candidate); err == nil && u.Scheme != "" && u.Host != "" {
		return u.Scheme + "://" + u.Host
	}
	return ""
}

// consoleOriginMatches 判断 src（scheme://host）是否命中任一 console 回调 URL 的 scheme://host。
func consoleOriginMatches(src string, redirectURIs []string) bool {
	for _, ru := range redirectURIs {
		if u, err := url.Parse(ru); err == nil && u.Scheme != "" && u.Host != "" {
			if u.Scheme+"://"+u.Host == src {
				return true
			}
		}
	}
	return false
}

// requestBaseURL 按请求派生 {scheme}://{host}，用于 issuer 缺省时生成绝对地址。
func requestBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	switch p := c.GetHeader("X-Forwarded-Proto"); p {
	case "https", "http":
		scheme = p
	}
	return scheme + "://" + c.Request.Host
}

// joinPath 拼接挂载前缀与路径（去重斜杠）。
func joinPath(base, p string) string {
	base = trimSuffixSlash(base)
	if base == "" || base == "/" {
		return p
	}
	return base + p
}

// trimSuffixSlash 去掉结尾的 '/'。
func trimSuffixSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
