package appassembly

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/oidcctx"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

// resolveResult 是一次 Host 解析的产物，放入请求上下文供 Gin/controller/OP 共享（DG-MW-1：一请求一解析）。
type resolveResult struct {
	Host      string
	TenantID  string
	Issuer    string
	Audience  string
	ClientID  string
	BindingID string
	Kind      model.ConsoleBindingKind
	// RedirectTo 非空表示 alias：应 302 到该 URL（path+query 已在中间件拼好）。
	RedirectTo string
}

const (
	resolveKey = "appassembly_resolve"
	// defaultHostCacheTTL host 路由缓存 TTL（P0 占位：暂不启用缓存，直查 DB；见 §6.1）。
	defaultHostCacheTTL = 60
)

// normalizeHost 归一化 Host：小写、去端口、去末尾点。
func normalizeHost(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	if i := strings.IndexByte(h, ':'); i >= 0 {
		h = h[:i]
	}
	h = strings.TrimSuffix(h, ".")
	return h
}

// resolveHost 两段式解析 Host（DG-ALIAS-1 / §6）：
//  1. 查 iam_console_binding（CrossTenant）命中 kind=platform/tenant/central 且 enable；
//  2. 未命中查 iam_domain：active alias → 302 target；active primary 无 binding → 拒绝；其他 → 404。
//
// 返回 (result, handled)：handled=true 表示中间件已写响应（302/404/拒绝），调用方应 return。
func (a *App) resolveHost(ctx context.Context, host, rawPathAndQuery string) (*resolveResult, bool, int) {
	db := a.dbGetter(ctx)
	if db == nil {
		return nil, false, http.StatusInternalServerError
	}
	host = normalizeHost(host)

	// 1) binding 命中（跨租户读）。
	bindings, err := dao.NewConsoleBindingDao(dao.WithDBGetter(a.dbGetter)).GetByHost(ctx, host)
	if err != nil {
		glog.Errorf(ctx, "[appassembly.resolveHost] binding query fail, host:%s, err:%v", host, err)
		return nil, true, http.StatusInternalServerError
	}
	for i := range bindings {
		b := bindings[i]
		if b.Status != model.ConsoleBindingStatusEnable {
			continue // draft/disable 不对外
		}
		return &resolveResult{
			Host: host, TenantID: b.TenantID, Issuer: b.Issuer, Audience: b.Audience,
			ClientID: b.ClientID, BindingID: b.ID, Kind: b.Kind,
		}, false, 0
	}

	// 2) 未命中 binding：查 iam_domain（CrossTenant），两段式。
	dom, derr := dao.NewDomainDao(dao.WithDBGetter(a.dbGetter)).
		GetByDomainCrossTenant(ctx, host)
	if derr != nil {
		glog.Errorf(ctx, "[appassembly.resolveHost] domain query fail, host:%s, err:%v", host, derr)
		return nil, true, http.StatusInternalServerError
	}
	if dom != nil {
		switch {
		case dom.DomainRole == model.DomainRoleAlias && dom.Status == model.DomainStatusActive:
			// active alias → 302 到 target primary host（保留 path+query）；alias 不查 binding。
			target, e := dao.NewDomainDao(dao.WithDBGetter(a.dbGetter)).GetByID(
				dbclient.CrossTenantContext(ctx), dom.TargetDomainID)
			if e != nil || target == nil {
				return nil, true, http.StatusNotFound
			}
			return &resolveResult{Host: host, RedirectTo: "https://" + target.Domain + rawPathAndQuery}, true, http.StatusFound
		case dom.DomainRole == model.DomainRolePrimary && dom.Status == model.DomainStatusActive:
			// active primary 但无 enable binding → 配置缺失，拒绝（不回退中央）。
			return nil, true, http.StatusNotFound
		default:
			// disabled/retired/draft/verified/tls_ready 等 → 404。
			return nil, true, http.StatusNotFound
		}
	}

	// 完全未知 Host → 404（不回退中央）。
	return nil, true, http.StatusNotFound
}

// HostResolveMiddleware 返回入口 Host 解析中间件：解析一次、写 pkg/oidcctx、按 Host 类型做白名单/alias 302/404。
//
// 仅当配置了 Raw 且开启绑定路由时由组合根挂载（见 MountHostResolver）。解析失败/未知一律 fail-closed。
// D Host（kind=tenant）访问 /v1/platform/*、/v1/rp/*、/install/*、平台 back-channel → 404。
func (a *App) HostResolveMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		path := c.Request.URL.Path
		rawPQ := c.Request.URL.RequestURI()

		res, handled, code := a.resolveHost(c.Request.Context(), host, rawPQ)
		if handled {
			if code == http.StatusFound && res != nil && res.RedirectTo != "" {
				// alias：仅 GET/HEAD 允许 302；其他方法 405。
				if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
					c.Redirect(http.StatusFound, res.RedirectTo)
					c.Abort()
					return
				}
				c.Status(http.StatusMethodNotAllowed)
				c.Abort()
				return
			}
			c.Status(code)
			c.Abort()
			return
		}

		// 平台专属路径：只有 kind=platform 可访问；D 租户 Host 一律 404（ProcessAll Host 白名单）。
		if isPlatformOnlyPath(path) && res.Kind != model.ConsoleBindingKindPlatform && res.Kind != model.ConsoleBindingKindCentral {
			c.Status(http.StatusNotFound)
			c.Abort()
			return
		}

		// 写入 oidcctx（唯一真源）+ gin keys 供下游读取。
		ctx := oidcctx.WithIssuer(c.Request.Context(), res.Issuer)
		c.Request = c.Request.WithContext(ctx)
		c.Set(resolveKey, res)
		c.Next()
	}
}

// isPlatformOnlyPath 判定仅平台 Host 可访问的路径前缀。
func isPlatformOnlyPath(p string) bool {
	for _, pre := range []string{"/v1/platform/", "/v1/rp/", "/install/"} {
		if strings.HasPrefix(p, pre) {
			return true
		}
	}
	return false
}

// resolveFromContext 取出本次请求的解析结果。
func resolveFromContext(c *gin.Context) (*resolveResult, bool) {
	v, ok := c.Get(resolveKey)
	if !ok {
		return nil, false
	}
	r, ok := v.(*resolveResult)
	return r, ok
}

var _ = url.Parse
var _ = gorm.ErrRecordNotFound

// MountHostResolver 在根 engine 上挂载 Host 解析中间件（须先于业务路由 Mount）。
// 启用后：每请求解析一次 Host→binding/domain，写 pkg/oidcctx；D Host 拒绝平台路径；
// /config.json、平台/租户组鉴权中间件据其取 issuer/audience。未启用时行为与旧版一致（静态配置）。
func (a *App) MountHostResolver(engine *gin.Engine) {
	engine.Use(a.HostResolveMiddleware())
}
