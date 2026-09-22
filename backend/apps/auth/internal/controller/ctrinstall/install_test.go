package ctrinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoinstall"
	"github.com/morehao/ark-iam/auth/internal/service/svcinstall"
	"github.com/morehao/ark-iam/auth/testutil"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/biz/gcontext"
)

// 这些用例锁的是**对外 HTTP 契约**：状态码 + 业务码。
//
// 与项目其它接口不同，/install 必须用真实 HTTP 状态码（gincontext.FailWithStatus）——
// 调用方是浏览器页面与自动化部署脚本，它们按 409/401/503/400 决定"该做什么"。
// 因此这里断言的是 rec.Code，而不是"恒 200 + 业务码"。

// newInstallEngine 装配只含 /install 路由的引擎（不引 OIDC provider，避免依赖 Redis/密钥）。
func newInstallEngine(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	db := testutil.SetupSQLite(t, model.AllEntities()...)
	config.Conf = &pkgconfig.Config{}
	config.Conf.OIDC.Issuer = "http://localhost:8081/oidc"
	config.Conf.OIDC.FrontendLoginURL = "http://localhost:4000/login"
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	initInstallRoutesForTest(engine)
	return engine, db
}

func initInstallRoutesForTest(engine *gin.Engine) {
	g := engine.Group("/install")
	ctr := NewInstallCtr()
	g.GET("/status", ctr.Status)
	g.POST("/initialize", ctr.Initialize)
}

func postInitialize(t *testing.T, engine *gin.Engine, token string, req dtoinstall.InstallationInitializeReq) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal req: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/install/initialize", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	if token != "" {
		httpReq.Header.Set(svcinstall.HeaderBootstrapToken, token)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httpReq)
	return rec
}

func validBody() dtoinstall.InstallationInitializeReq {
	return dtoinstall.InstallationInitializeReq{
		AdminUsername: "acme-admin",
		AdminPassword: "Acme1234",
		AdminEmail:    "admin@acme.example.com",
		AdminName:     "ACME 管理员",
		TenantName:    "ACME 平台运营中心",
	}
}

// envelope 解析响应体（gincontext 的 DtoRender：code/msg/requestID/data）。
func envelope(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body %q: %v", rec.Body.String(), err)
	}
	return body
}

func codeOf(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()
	v, ok := envelope(t, rec)["code"].(float64)
	if !ok {
		t.Fatalf("响应体缺少 code: %s", rec.Body.String())
	}
	return int(v)
}

// TestStatusEndpointIsPublicAndReadOnly 状态查询必须公开可读：初始化页面在**没有任何凭据**时
// 也要能判断该渲染表单还是"已完成"页。
func TestStatusEndpointIsPublicAndReadOnly(t *testing.T) {
	engine, _ := newInstallEngine(t)
	t.Setenv(svcinstall.EnvBootstrapToken, "s3cret-token")

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/install/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := envelope(t, rec)
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("响应体缺少 data: %s", rec.Body.String())
	}
	if data["initialized"] != false {
		t.Errorf("initialized = %v, want false", data["initialized"])
	}
	if data["tokenRequired"] != true {
		t.Errorf("tokenRequired = %v, want true", data["tokenRequired"])
	}
	if data["schemaReady"] != true {
		t.Errorf("schemaReady = %v, want true", data["schemaReady"])
	}
	// 状态查询不得回显任何敏感值
	if strings.Contains(rec.Body.String(), "s3cret-token") {
		t.Error("/install/status 泄露了 BOOTSTRAP_TOKEN")
	}
}

// TestInitializeHTTPContract 逐个错误路径断言"状态码 + 业务码"。
func TestInitializeHTTPContract(t *testing.T) {
	cases := []struct {
		name       string
		envToken   string
		sendToken  string
		mutate     func(*dtoinstall.InstallationInitializeReq)
		wantStatus int
		wantCode   int
	}{
		{
			name: "未携带令牌 → 401", envToken: "s3cret", sendToken: "",
			wantStatus: http.StatusUnauthorized, wantCode: 107001,
		},
		{
			name: "令牌不匹配 → 401", envToken: "s3cret", sendToken: "wrong",
			wantStatus: http.StatusUnauthorized, wantCode: 107001,
		},
		{
			name: "服务端未配置令牌 → 503（fail-closed，不是跳过校验）", envToken: "", sendToken: "anything",
			wantStatus: http.StatusServiceUnavailable, wantCode: 107002,
		},
		{
			name: "口令强度不足 → 400", envToken: "s3cret", sendToken: "s3cret",
			mutate:     func(r *dtoinstall.InstallationInitializeReq) { r.AdminPassword = "weakpass" },
			wantStatus: http.StatusBadRequest, wantCode: 107006,
		},
		{
			name: "缺联系方式 → 400", envToken: "s3cret", sendToken: "s3cret",
			mutate:     func(r *dtoinstall.InstallationInitializeReq) { r.AdminEmail = ""; r.AdminPhone = "" },
			wantStatus: http.StatusBadRequest, wantCode: 107005,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine, _ := newInstallEngine(t)
			t.Setenv(svcinstall.EnvBootstrapToken, tc.envToken)
			req := validBody()
			if tc.mutate != nil {
				tc.mutate(&req)
			}
			rec := postInitialize(t, engine, tc.sendToken, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if got := codeOf(t, rec); got != tc.wantCode {
				t.Errorf("code = %d, want %d", got, tc.wantCode)
			}
		})
	}
}

// TestInitializeSucceedsThenLocks 完整走一遍：成功 → 200 + 报告；再试 → 409 且锁定。
//
// "锁定"必须是库内事实而非进程内状态：这里在同一个进程里第二次请求（同库）即被拒；
// 跨进程/重启后的行为由 pkg/seed 的自锁判定保证（依据平台租户行，见 seed.IsInitialized）。
func TestInitializeSucceedsThenLocks(t *testing.T) {
	engine, _ := newInstallEngine(t)
	t.Setenv(svcinstall.EnvBootstrapToken, "s3cret-token")

	rec := postInitialize(t, engine, "s3cret-token", validBody())
	if rec.Code != http.StatusOK {
		t.Fatalf("首次初始化 status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	body := envelope(t, rec)
	data := body["data"].(map[string]any)
	if data["adminUsername"] != "acme-admin" {
		t.Errorf("adminUsername = %v", data["adminUsername"])
	}
	if data["loginURL"] != "http://localhost:4000/login" {
		t.Errorf("loginURL = %v", data["loginURL"])
	}
	report := data["report"].(map[string]any)
	if report["tenantId"] == "" || report["tenantId"] == nil {
		t.Error("report.tenantId 不得为空")
	}
	changes, ok := report["changes"].([]any)
	if !ok || len(changes) == 0 {
		t.Fatalf("report.changes 必须非空: %v", report["changes"])
	}
	// 口令绝不出现在响应中
	if strings.Contains(rec.Body.String(), "Acme1234") {
		t.Error("初始化响应泄露了明文口令")
	}

	// 第二次：永久 409，且不得写入任何数据
	req := validBody()
	req.AdminUsername = "attacker"
	rec2 := postInitialize(t, engine, "s3cret-token", req)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("二次初始化 status = %d, want 409 (body=%s)", rec2.Code, rec2.Body.String())
	}
	if got := codeOf(t, rec2); got != 107000 {
		t.Errorf("二次初始化 code = %d, want 107000", got)
	}
}

// TestInitializeAuditsWithoutLeakingSecrets 审计必须记录成功与失败，且**绝不**包含口令/令牌。
//
// /install 是未认证写面，失败记录是发现扫描行为的唯一线索；同时它又是最可能把明文口令
// 写进日志与审计的接口（入参里就有口令）。这两件事必须同时成立。
func TestInitializeAuditsWithoutLeakingSecrets(t *testing.T) {
	engine, db := newInstallEngine(t)
	t.Setenv(svcinstall.EnvBootstrapToken, "s3cret-token")

	// 一次失败（弱口令：无大写字母。口令值刻意独特，便于搜索是否泄露）
	bad := validBody()
	bad.AdminPassword = "leakme123"
	if rec := postInitialize(t, engine, "s3cret-token", bad); rec.Code != http.StatusBadRequest {
		t.Fatalf("弱口令 status = %d, want 400", rec.Code)
	}
	// 一次成功
	if rec := postInitialize(t, engine, "s3cret-token", validBody()); rec.Code != http.StatusOK {
		t.Fatalf("初始化 status = %d, want 200", rec.Code)
	}

	// audit_log 是租户表：断言查询也必须显式声明作用域（fail-closed 下缺失会直接报错）。
	var logs []model.AuditLogEntity
	auditCtx := gcontext.WithTenantScope(context.Background(), gcontext.AllScope())
	if err := db.WithContext(auditCtx).Where("action = ?", model.AuditActionInstallationInitialize).Find(&logs).Error; err != nil {
		t.Fatalf("query audit logs: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("审计记录数 = %d, want ≥2（成功与失败都要记）", len(logs))
	}
	var success, failure int
	for _, l := range logs {
		switch l.Result {
		case model.AuditResultSuccess:
			success++
		case model.AuditResultFailure:
			failure++
		}
		if strings.Contains(l.Detail, "Acme1234") || strings.Contains(l.Detail, "leakme123") {
			t.Errorf("审计 detail 泄露了口令: %s", l.Detail)
		}
		if strings.Contains(l.Detail, "s3cret-token") {
			t.Errorf("审计 detail 泄露了引导令牌: %s", l.Detail)
		}
	}
	if success == 0 || failure == 0 {
		t.Errorf("成功 %d 条 / 失败 %d 条，两者都必须有", success, failure)
	}
	// 成功记录必须能定位到本次创建的租户（事后追责的关键字段）
	var tenantID string
	for _, l := range logs {
		if l.Result == model.AuditResultSuccess && l.TenantID != "" {
			tenantID = l.TenantID
		}
	}
	if tenantID == "" {
		t.Error("成功审计必须带 tenantId（事后唯一能回答'建在哪个租户'的信息）")
	}
}

// TestStatusReportsSchemaNotReady 表结构未就绪时必须如实报告 schemaReady=false：
// 这把"db.auto_migrate=false + 空库"这个部署失败模式变成可执行提示，而不是一个 500。
func TestStatusReportsSchemaNotReady(t *testing.T) {
	testutil.SetupSQLite(t) // 刻意不迁移任何实体
	config.Conf = &pkgconfig.Config{}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	initInstallRoutesForTest(engine)

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/install/status", nil))
	// 表不存在时 schemaReady 的判定本身会报错 → 503（"暂时给不出状态"），
	// 前端据此提示"检查 db.auto_migrate"，而不是渲染一个必然失败的初始化表单。
	if rec.Code == http.StatusOK {
		body := envelope(t, rec)
		if data, ok := body["data"].(map[string]any); ok {
			if data["schemaReady"] != false {
				t.Errorf("空库 schemaReady = %v, want false", data["schemaReady"])
			}
		}
		return
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("空库 status 查询 status = %d, want 200(schemaReady=false) 或 503", rec.Code)
	}
}
