package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 本文件覆盖 T1.2（pkg/middleware.RequireAppAccess）的 8 个用例：
// 准入/应用/订阅三者各自的可放行与拒绝、客户端不存在、缺 client_id、以及 DAO 报错必须上抛而非伪装成 110016。
//
// 测试库沿用同目录 oidc_auth_test.go 的口径：自建内存 SQLite 并注册为全局 iam 库
// （不依赖各 app 的 testutil），因此被测中间件内部的 dao.NewXxxDao() 直接落本测试库。

// appAccessProbe 是 T1.2 的夹具：一个「租户 + 成员 + 应用 + 客户端 + 订阅 + 准入行」可逐项改档的基线。
type appAccessProbe struct {
	db       *gorm.DB
	tenantID string
	userID   string
	appID    string
	clientID string
	called   bool
}

// newAppAccessProbe 构造夹具；withSchema=false 时不建表（用于「DAO 报错」用例：
// 查询不存在的表会返回系统错误，中间件必须如实上抛）。
func newAppAccessProbe(t *testing.T, withSchema bool) *appAccessProbe {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:require_app_access_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	if withSchema {
		// 三元组 + 客户端解析所需的四张表；不建 user（CheckUsable 不认识它）。
		require.NoError(t, db.AutoMigrate(
			&model.ApplicationEntity{},
			&model.TenantApplicationEntity{},
			&model.UserApplicationEntity{},
			&model.ApplicationClientEntity{},
		))
	}
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return &appAccessProbe{db: db, tenantID: "tenant-1", userID: "user-1", appID: "app-1", clientID: "probe_client"}
}

func (p *appAccessProbe) seedApp(t *testing.T, status model.AppStatus) {
	t.Helper()
	require.NoError(t, p.db.Create(&model.ApplicationEntity{
		BaseEntity:   gormdao.BaseEntity{StringID: gormdao.StringID{ID: p.appID}},
		Code:         "probe_app",
		Name:         "探针应用",
		Status:       status,
		RoleTemplate: model.RoleTemplateItemList{},
	}).Error)
}

func (p *appAccessProbe) seedSubscription(t *testing.T, status model.TenantApplicationStatus) {
	t.Helper()
	require.NoError(t, p.db.Create(&model.TenantApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "sub-1"}},
		TenantID:   p.tenantID,
		AppID:      p.appID,
		Status:     status,
	}).Error)
}

func (p *appAccessProbe) seedGrant(t *testing.T, status model.UserApplicationStatus) {
	t.Helper()
	require.NoError(t, p.db.Create(&model.UserApplicationEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "grant-1"}},
		TenantID:   p.tenantID,
		UserID:     p.userID,
		AppID:      p.appID,
		Status:     status,
		Source:     model.UserApplicationSourceManual,
	}).Error)
}

// seedClient 写入客户端行；code 传别的值即造出「令牌里的 client_id 在库里不存在」。
func (p *appAccessProbe) seedClient(t *testing.T, code string) {
	t.Helper()
	require.NoError(t, p.db.Create(&model.ApplicationClientEntity{
		BaseEntity:             gormdao.BaseEntity{StringID: gormdao.StringID{ID: "client-1"}},
		AppID:                  p.appID,
		Code:                   code,
		Name:                   "探针客户端",
		RedirectURIs:           model.RedirectURIList{},
		PostLogoutRedirectURIs: model.PostLogoutRedirectURIList{},
		GrantTypes:             model.GrantTypeList{model.GrantTypeAuthorizationCode},
		ResponseTypes:          model.ResponseTypeList{"code"},
		AllowedOrigins:         model.AllowedOriginList{},
		DefaultScopes:          model.DefaultScopeList{"openid"},
		Status:                 model.ApplicationClientStatusEnable,
	}).Error)
}

// xdb 返回显式声明「全部租户」作用域的句柄：改档用的是 UPDATE/DELETE，
// 而租户隔离插件对 UPDATE/DELETE 会注入条件，测试里必须显式声明作用域（fail-closed）。
func (p *appAccessProbe) xdb() *gorm.DB {
	return p.db.WithContext(dbclient.CrossTenantContext(context.Background()))
}

// setAppStatus / setSubscriptionStatus / setGrantStatus 在已播种的基线上改档（切状态），
// 不做第二次 INSERT（唯一索引会拒绝，且那样测的就不是"同一行被停用"）。
func (p *appAccessProbe) setAppStatus(t *testing.T, status model.AppStatus) {
	t.Helper()
	require.NoError(t, p.xdb().Model(&model.ApplicationEntity{}).Where("id = ?", p.appID).Update("status", status).Error)
}

func (p *appAccessProbe) setSubscriptionStatus(t *testing.T, status model.TenantApplicationStatus) {
	t.Helper()
	require.NoError(t, p.xdb().Model(&model.TenantApplicationEntity{}).Where("id = ?", "sub-1").Update("status", status).Error)
}

func (p *appAccessProbe) setGrantStatus(t *testing.T, status model.UserApplicationStatus) {
	t.Helper()
	require.NoError(t, p.xdb().Model(&model.UserApplicationEntity{}).Where("id = ?", "grant-1").Update("status", status).Error)
}

// removeGrant 软删准入行（模拟"从未授权"）：查询侧带 deleted_at IS NULL，效果等价于没有准入行。
func (p *appAccessProbe) removeGrant(t *testing.T) {
	t.Helper()
	require.NoError(t, p.xdb().Where("id = ?", "grant-1").Delete(&model.UserApplicationEntity{}).Error)
}

// renameClient 把客户端编码改成别的值，造出「令牌里的 client_id 在库里不存在」。
func (p *appAccessProbe) renameClient(t *testing.T, code string) {
	t.Helper()
	require.NoError(t, p.xdb().Model(&model.ApplicationClientEntity{}).Where("id = ?", "client-1").Update("code", code).Error)
}

// seedBaseline 播种「全部 enable」的可用基线。
func (p *appAccessProbe) seedBaseline(t *testing.T) {
	t.Helper()
	p.seedApp(t, model.AppStatusEnable)
	p.seedSubscription(t, model.TenantApplicationStatusEnable)
	p.seedGrant(t, model.UserApplicationStatusEnable)
	p.seedClient(t, p.clientID)
}

// appAccessResult 一次探测的结果。
type appAccessResult struct {
	httpStatus int
	code       int
	called     bool
}

// probe 以给定身份上下文跑一次请求：withTenant/withUser 控制是否写入租户与成员身份，
// clientID 为空表示令牌不带 client_id（如非 OIDC 通道）。
func (p *appAccessProbe) probe(t *testing.T, clientID string, withTenant, withUser bool) appAccessResult {
	t.Helper()
	engine := gin.New()
	engine.ContextWithFallback = true // 与生产一致：gin ctx 可当 context.Context 传给下游 dao
	engine.Use(func(ctx *gin.Context) {
		if withTenant {
			ctx.Set(gcontext.KeyTenantID, p.tenantID)
			gincontext.SetTenantScope(ctx, gcontext.CurrentScope(p.tenantID))
		}
		if withUser {
			ctx.Set(gcontext.KeyUserID, p.userID)
		}
		if clientID != "" {
			identity.SetClientID(ctx, clientID)
		}
		ctx.Next()
	})
	engine.Use(RequireAppAccess())
	engine.GET("/probe", func(ctx *gin.Context) {
		p.called = true
		gincontext.Success(ctx, gin.H{"ok": true})
	})

	p.called = false
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", nil))

	var body struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "响应体必须是统一 envelope：%s", rec.Body.String())
	return appAccessResult{httpStatus: rec.Code, code: body.Code, called: p.called}
}

// TestRequireAppAccess 覆盖「准入 ∧ 订阅 ∧ 应用」三层准入在请求期的执行点。
func TestRequireAppAccess(t *testing.T) {
	t.Run("三层全 enable 放行", func(t *testing.T) {
		p := newAppAccessProbe(t, true)
		p.seedBaseline(t)

		got := p.probe(t, p.clientID, true, true)

		assert.Equal(t, 0, got.code, "准入通过应返回业务成功码")
		assert.True(t, got.called, "准入通过时 handler 必须被调用")
	})

	t.Run("无准入行拒绝", func(t *testing.T) {
		p := newAppAccessProbe(t, true)
		p.seedBaseline(t)
		p.removeGrant(t) // 准入行缺失即无准入

		got := p.probe(t, p.clientID, true, true)

		assert.Equal(t, code.AuthAppAccessDeniedError, got.code)
		assert.False(t, got.called, "拒绝时 handler 不得被调用")
	})

	t.Run("准入行 disable 拒绝（证走准入行而非角色）", func(t *testing.T) {
		p := newAppAccessProbe(t, true)
		p.seedBaseline(t)
		p.setGrantStatus(t, model.UserApplicationStatusDisable)

		got := p.probe(t, p.clientID, true, true)

		assert.Equal(t, code.AuthAppAccessDeniedError, got.code)
		assert.False(t, got.called)
	})

	t.Run("应用 disable 拒绝", func(t *testing.T) {
		p := newAppAccessProbe(t, true)
		p.seedBaseline(t)
		p.setAppStatus(t, model.AppStatusDisable)

		got := p.probe(t, p.clientID, true, true)

		assert.Equal(t, code.AuthAppAccessDeniedError, got.code)
		assert.False(t, got.called)
	})

	t.Run("订阅 disable 拒绝", func(t *testing.T) {
		p := newAppAccessProbe(t, true)
		p.seedBaseline(t)
		p.setSubscriptionStatus(t, model.TenantApplicationStatusDisable)

		got := p.probe(t, p.clientID, true, true)

		assert.Equal(t, code.AuthAppAccessDeniedError, got.code)
		assert.False(t, got.called)
	})

	t.Run("客户端不存在拒绝（fail-closed）", func(t *testing.T) {
		p := newAppAccessProbe(t, true)
		p.seedBaseline(t)
		p.renameClient(t, "another_client") // 库里只有别的客户端

		got := p.probe(t, p.clientID, true, true)

		// 口径与 T1.1 上提的 AppIDByClientID 一致：客户端行不存在属状态异常，返回 error。
		// 中间件按「系统错误如实上抛」处置（不伪装成无权限），但**必须拒绝且不进入 handler**。
		assert.NotEqual(t, 0, got.code, "禁止放行")
		assert.NotEqual(t, code.AuthAppAccessDeniedError, got.code, "状态异常不得伪装成 110016")
		assert.False(t, got.called)
	})

	t.Run("令牌无 client_id 拒绝", func(t *testing.T) {
		p := newAppAccessProbe(t, true)
		p.seedBaseline(t)

		got := p.probe(t, "", true, true) // 不注入 client_id

		assert.Equal(t, code.AuthAppAccessDeniedError, got.code)
		assert.False(t, got.called)
	})

	t.Run("缺成员身份拒绝", func(t *testing.T) {
		p := newAppAccessProbe(t, true)
		p.seedBaseline(t)

		got := p.probe(t, p.clientID, true, false) // 无 user_id

		assert.Equal(t, code.AuthAppAccessDeniedError, got.code)
		assert.False(t, got.called)
	})

	t.Run("DAO 报错如实上抛而非伪装成 110016", func(t *testing.T) {
		p := newAppAccessProbe(t, false) // 不建表：所有查询都是系统错误
		got := p.probe(t, p.clientID, true, true)

		assert.NotEqual(t, code.AuthAppAccessDeniedError, got.code, "系统错误不得伪装成无权限")
		assert.NotEqual(t, 0, got.code, "系统错误不得放行")
		assert.False(t, got.called)
	})
}
