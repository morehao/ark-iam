package internal

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
			// clientId：本控制台 SPA 的 OIDC client_id（平台管理台内置客户端编码）。
			"clientId": model.SeedBuiltinClientPlatformAdminWeb,
			// enabledDomains：已挂载域白名单（G4）；缺省/全量时前端视为全部域可用。
			"enabledDomains": enabledDomains,
		})
	})
	return nil
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
