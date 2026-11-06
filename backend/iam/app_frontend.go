package iam

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/frontend"
)

// MountFrontend 托管内嵌登录/初始化页，并注册 /config.json 运行时配置端点。
//
// 登录页（login-web）前端在根路径访问 /v1、/oidc，因此同源托管在 API 同域根即可直接
// 工作（相对路径天然正确，无需构建期 base）。应在 Mount 之后调用。
func (a *App) MountFrontend(engine *gin.Engine, basePath string) error {
	engine.GET(joinPath(basePath, "/config.json"), func(c *gin.Context) {
		conf := a.cfg.Raw
		issuer := ""
		if conf != nil {
			issuer = conf.OIDC.Issuer
		}
		c.JSON(http.StatusOK, gin.H{
			"issuer":       issuer,
			"apiBase":      "/v1",
			"oidcBase":     "/oidc",
			"callbackPath": "/auth/callback",
			"clientId":     "",
		})
	})

	dist, err := fs.Sub(frontend.Login, "login")
	if err != nil {
		return err
	}

	// SPA fallback：GET 请求命中静态文件则直接返回；否则回退 index.html（交给前端路由）。
	engine.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet || isAPIPath(c.Request.URL.Path) {
			c.Status(http.StatusNotFound)
			return
		}
		rel := strings.TrimPrefix(c.Request.URL.Path, strings.TrimSuffix(basePath, "/"))
		rel = strings.TrimPrefix(rel, "/")
		if rel != "" {
			if data, ferr := fs.ReadFile(dist, rel); ferr == nil {
				c.Data(http.StatusOK, contentType(rel), data)
				return
			}
		}
		// 回退 SPA 首页。
		data, ferr := fs.ReadFile(dist, "index.html")
		if ferr != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
	return nil
}

func isAPIPath(p string) bool {
	// /install 是前端页面（初始化页），只有其下的 /install/status、/install/initialize 是 API。
	for _, prefix := range []string{"/v1/", "/oidc/", "/config.json"} {
		if strings.HasPrefix(p, prefix) || p == strings.TrimSuffix(prefix, "/") {
			return true
		}
	}
	if strings.HasPrefix(p, "/install/") {
		return true
	}
	return false
}

func joinPath(base, p string) string {
	base = strings.TrimSuffix(base, "/")
	if base == "" || base == "/" {
		return p
	}
	return base + p
}

func contentType(name string) string {
	switch {
	case strings.HasSuffix(name, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(name, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(name, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(name, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".woff2"):
		return "font/woff2"
	}
	return "application/octet-stream"
}
