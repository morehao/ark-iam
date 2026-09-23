package svcinstall

import (
	"errors"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoinstall"
	"github.com/morehao/ark-iam/auth/testutil"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/model"
)

// setupSvc 打开内存 SQLite（全部 IAM 表）+ 全局 iam 库注册，并注入一份可预测的配置。
//
// 返回的 *gorm.DB 用于**直接**断言落库结果（绕过租户作用域插件）——
// 被测服务走的是全局 iam 库，断言走原始句柄，两者是同一个内存库。
func setupSvc(t *testing.T) (*gin.Context, *gorm.DB) {
	t.Helper()
	db := testutil.SetupSQLite(t, model.AllEntities()...)
	config.Conf = &pkgconfig.Config{}
	config.Conf.OIDC.Issuer = "http://localhost:8081/oidc"
	config.Conf.OIDC.FrontendLoginURL = "http://localhost:4000/login"
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)
	// 刻意**不**给 ctx 声明租户作用域：真实请求路径上也不会有人替 /install 声明。
	// 引导自身的跨租户声明必须由 service 自己做（否则 fail-closed 会直接拒绝），
	// 这条用例因此同时是"service 不依赖调用方设置作用域"的回归。
	return ctx, db
}

func validReq() *dtoinstall.InstallationInitializeReq {
	return &dtoinstall.InstallationInitializeReq{
		AdminUsername: "acme-admin",
		AdminPassword: "Acme1234",
		AdminEmail:    "admin@acme.example.com",
		AdminName:     "ACME 管理员",
		TenantName:    "ACME 平台运营中心",
	}
}

// TestStatus_BeforeAndAfterInitialize 状态查询是页面与部署脚本的唯一判断依据，
// 三个状态位必须如实反映库内事实。
func TestStatus_BeforeAndAfterInitialize(t *testing.T) {
	ctx, _ := setupSvc(t)
	svc := NewInstallSvc()

	t.Setenv(EnvBootstrapToken, "s3cret-token")
	before, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("status fail: %v", err)
	}
	if before.Initialized {
		t.Error("空库必须报告 initialized=false")
	}
	if !before.SchemaReady {
		t.Error("表已建出，schemaReady 必须为 true")
	}
	if !before.TokenRequired {
		t.Error("已配置 BOOTSTRAP_TOKEN，tokenRequired 必须为 true")
	}

	if _, err := svc.Initialize(ctx, validReq()); err != nil {
		t.Fatalf("initialize fail: %v", err)
	}

	after, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("status (after) fail: %v", err)
	}
	if !after.Initialized {
		t.Error("初始化后必须报告 initialized=true")
	}
}

// TestStatus_TokenNotConfigured fail-closed 的前置告知：未配置 token 时
// tokenRequired=false，页面可提前禁用提交，而不是让运维填完三步表单才收到 503。
func TestStatus_TokenNotConfigured(t *testing.T) {
	ctx, _ := setupSvc(t)
	t.Setenv(EnvBootstrapToken, "")
	st, err := NewInstallSvc().Status(ctx)
	if err != nil {
		t.Fatalf("status fail: %v", err)
	}
	if st.TokenRequired {
		t.Error("未配置 BOOTSTRAP_TOKEN 时 tokenRequired 必须为 false")
	}
}

// TestInitialize_UsesRequestInputs 页面填写的身份信息必须真正落库
// （这是"管理员由运维指定"的全部意义），且口令以摘要形式存储。
func TestInitialize_UsesRequestInputs(t *testing.T) {
	ctx, db := setupSvc(t)
	t.Setenv(EnvBootstrapToken, "s3cret-token")

	res, err := NewInstallSvc().Initialize(ctx, validReq())
	if err != nil {
		t.Fatalf("initialize fail: %v", err)
	}
	if res.AdminUsername != "acme-admin" {
		t.Errorf("回显 adminUsername = %q", res.AdminUsername)
	}
	if res.Report.TenantID == "" {
		t.Error("报告必须带 tenantId")
	}
	if len(res.Report.Changes) == 0 {
		t.Error("报告必须列出本次写入")
	}
	if res.LoginURL != "http://localhost:4000/login" {
		t.Errorf("loginURL = %q, want 取 oidc.frontendLoginURL", res.LoginURL)
	}

	var person model.PersonEntity
	if err := db.Where("username = ?", "acme-admin").First(&person).Error; err != nil {
		t.Fatalf("person 未按页面输入创建: %v", err)
	}
	if person.PrimaryEmail == nil || *person.PrimaryEmail != "admin@acme.example.com" {
		t.Errorf("primary_email = %v, want 页面输入", person.PrimaryEmail)
	}
	if person.Name != "ACME 管理员" {
		t.Errorf("name = %q, want 页面输入", person.Name)
	}
	// 口令必须以摘要入库，且**不得**等于明文（这是最容易写错、后果最严重的一处）
	if person.PasswordEncrypted == "" || person.PasswordEncrypted == "Acme1234" {
		t.Errorf("口令未以摘要形式落库: %q", person.PasswordEncrypted)
	}

	var tenant model.TenantEntity
	if err := db.Where("code = ?", model.SeedPlatformTenantCode).First(&tenant).Error; err != nil {
		t.Fatalf("平台租户未创建: %v", err)
	}
	if tenant.Name != "ACME 平台运营中心" {
		t.Errorf("租户名 = %q, want 页面输入", tenant.Name)
	}
}

// TestInitialize_RejectsInvalidInput 入参校验必须在**服务端**做：
// /install 是未认证写面，前端校验只是体验优化，绕过它只需一个 curl。
func TestInitialize_RejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*dtoinstall.InstallationInitializeReq)
		wantErr error
	}{
		{"缺失联系方式", func(r *dtoinstall.InstallationInitializeReq) { r.AdminEmail = ""; r.AdminPhone = "" }, errContactRequired},
		{"弱口令（无大写）", func(r *dtoinstall.InstallationInitializeReq) { r.AdminPassword = "acme1234" }, errPasswordWeak},
		{"弱口令（无数字）", func(r *dtoinstall.InstallationInitializeReq) { r.AdminPassword = "AcmeSecret" }, errPasswordWeak},
		{"弱口令（过短）", func(r *dtoinstall.InstallationInitializeReq) { r.AdminPassword = "Ac1" }, errPasswordWeak},
		{"空用户名", func(r *dtoinstall.InstallationInitializeReq) { r.AdminUsername = "   " }, errBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := setupSvc(t)
			t.Setenv(EnvBootstrapToken, "s3cret-token")
			req := validReq()
			tc.mutate(req)
			if _, err := NewInstallSvc().Initialize(ctx, req); err != tc.wantErr {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// TestInitialize_RejectsWeakPasswordBeforeWriting 校验失败必须**零写入**：
// 不能出现"租户建了、管理员没建"的中间态（本项目单事务 + 入口校验共同保证）。
func TestInitialize_RejectsWeakPasswordBeforeWriting(t *testing.T) {
	ctx, db := setupSvc(t)
	t.Setenv(EnvBootstrapToken, "s3cret-token")
	req := validReq()
	req.AdminPassword = "weak"
	if _, err := NewInstallSvc().Initialize(ctx, req); err == nil {
		t.Fatal("弱口令必须被拒")
	}
	var tenants int64
	if err := db.Model(&model.TenantEntity{}).Count(&tenants).Error; err != nil {
		t.Fatalf("count tenants: %v", err)
	}
	if tenants != 0 {
		t.Errorf("校验失败后租户行数 = %d, want 0（不得留下半初始化状态）", tenants)
	}
}

// TestInitialize_SecondCallIsAlreadyInitialized 自锁：第二次调用返回"已初始化"且零写入。
// 这是安全要求（未认证写面不允许改已有系统），不是幂等性优化。
func TestInitialize_SecondCallIsAlreadyInitialized(t *testing.T) {
	ctx, db := setupSvc(t)
	t.Setenv(EnvBootstrapToken, "s3cret-token")
	svc := NewInstallSvc()
	if _, err := svc.Initialize(ctx, validReq()); err != nil {
		t.Fatalf("initialize fail: %v", err)
	}
	var before int64
	if err := db.Model(&model.PersonEntity{}).Count(&before).Error; err != nil {
		t.Fatalf("count persons: %v", err)
	}

	req := validReq()
	req.AdminUsername = "attacker"
	if _, err := svc.Initialize(ctx, req); err != errAlreadyInitialized {
		t.Fatalf("二次初始化 err = %v, want %v", err, errAlreadyInitialized)
	}
	var after int64
	if err := db.Model(&model.PersonEntity{}).Count(&after).Error; err != nil {
		t.Fatalf("count persons: %v", err)
	}
	if after != before {
		t.Errorf("二次初始化写入了数据: person %d -> %d", before, after)
	}
	var attacker model.PersonEntity
	if err := db.Where("username = ?", "attacker").First(&attacker).Error; err == nil {
		t.Error("二次初始化不得创建任何账号")
	}
}

// TestCheckBootstrapToken fail-closed 的三条路径。
func TestCheckBootstrapToken(t *testing.T) {
	cases := []struct {
		name     string
		env      string
		provided string
		wantErr  error
	}{
		{"未配置 token → 端点整体不可用（不是跳过校验）", "", "anything", errTokenNotConfigured},
		{"纯空白 token 视同未配置", "   ", "anything", errTokenNotConfigured},
		{"未携带 token", "s3cret", "", errTokenInvalid},
		{"token 不匹配", "s3cret", "wrong", errTokenInvalid},
		{"token 匹配（含尾随换行归一）", "s3cret\n", "s3cret", nil},
		{"调用方携带尾随空白也归一", "s3cret", "  s3cret  ", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvBootstrapToken, tc.env)
			if err := CheckBootstrapToken(tc.provided); err != tc.wantErr {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// TestHTTPStatusOf 状态码是**对外契约**（页面与部署脚本按它决定做什么）。
func TestHTTPStatusOf(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
		wantCode   int
	}{
		{errAlreadyInitialized, 409, 107000},
		{errTokenInvalid, 401, 107001},
		{errTokenNotConfigured, 503, 107002},
		{errBadRequest, 400, 107005},
		{errContactRequired, 400, 107005},
		{errPasswordWeak, 400, 107006},
	}
	for _, tc := range cases {
		status, bizCode := HTTPStatusOf(tc.err)
		if status != tc.wantStatus || bizCode != tc.wantCode {
			t.Errorf("HTTPStatusOf(%v) = (%d, %d), want (%d, %d)", tc.err, status, bizCode, tc.wantStatus, tc.wantCode)
		}
		if got := BusinessCodeOf(tc.err); got != tc.wantCode {
			t.Errorf("BusinessCodeOf(%v) = %d, want %d", tc.err, got, tc.wantCode)
		}
	}
	// 未知错误 → 500 且业务码识别为 0（不得伪装成 107003 或 400）
	unknown := errUnknownForTest()
	if status, _ := HTTPStatusOf(unknown); status != 500 {
		t.Errorf("未知错误 status = %d, want 500", status)
	}
	if got := BusinessCodeOf(unknown); got != 0 {
		t.Errorf("未知错误业务码 = %d, want 0", got)
	}
}

// TestLogSafeReqNeverContainsPassword 日志安全：/install 的入参里有明文口令，
// 一旦被整体序列化进日志就是口令泄露（且日志留存最久、可见人最多）。
func TestLogSafeReqNeverContainsPassword(t *testing.T) {
	req := validReq()
	got := LogSafeReq(req)
	if contains(got, req.AdminPassword) {
		t.Fatalf("LogSafeReq 泄露了口令: %s", got)
	}
	// 其余字段仍应保留（日志要有排查价值）
	if !contains(got, req.AdminUsername) {
		t.Errorf("LogSafeReq 丢了用户名: %s", got)
	}
	// 原请求对象不得被就地修改（调用方还要用它写库）
	if req.AdminPassword != "Acme1234" {
		t.Error("LogSafeReq 修改了入参（调用方仍需要明文口令去生成摘要）")
	}
}

// TestConsolesFromConfig 控制台地址来自 oidc.consoles，且与 L1 写库的地址同源。
func TestConsolesFromConfig(t *testing.T) {
	_, _ = setupSvc(t)
	config.Conf.OIDC.Consoles.PlatformAdminWeb.RedirectURIs = []string{"https://admin.acme.com/auth/callback"}
	config.Conf.OIDC.Consoles.PlatformAdminWeb.PostLogoutRedirectURIs = []string{"https://admin.acme.com/login"}
	config.Conf.OIDC.Consoles.TenantAdminWeb.RedirectURIs = []string{"https://tenant.acme.com/auth/callback"}

	consoles := consolesFromConfig()
	if got := consoles.PlatformAdminWeb.RedirectURIs[0]; got != "https://admin.acme.com/auth/callback" {
		t.Errorf("platformAdminWeb.redirectURIs = %q", got)
	}
	entry := consolesFromConfigForStatus()
	if entry.PlatformAdminWeb != "https://admin.acme.com/login" {
		t.Errorf("平台控制台入口 = %q, want 登出回跳地址（即登录页）", entry.PlatformAdminWeb)
	}
	// 未配置 postLogout 时用回调地址的 origin + /login 兜底（不能把 OAuth 回调地址当入口给运维看）
	if entry.TenantAdminWeb != "https://tenant.acme.com/login" {
		t.Errorf("租户控制台入口 = %q, want origin + /login", entry.TenantAdminWeb)
	}
}

// deref 取可空字符串指针的值（person.username/primary_email 为 *string）。
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// errUnknownForTest 一个不属于 install 领域的普通错误。
func errUnknownForTest() error { return errors.New("boom") }

// contains 字符串包含（避免为测试引入额外依赖）。
func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

// 引导令牌现在有两个来源：环境变量 BOOTSTRAP_TOKEN 与配置 install.bootstrapToken。
// 这里把**优先级**钉死，因为它决定了生产能不能用环境变量覆盖掉入库的配置值。
func TestBootstrapTokenSources(t *testing.T) {
	cases := []struct {
		name     string
		env      string
		cfg      string
		provided string
		wantErr  error
	}{
		{"仅配置：可用", "", "cfg-secret", "cfg-secret", nil},
		{"仅配置：不匹配仍 401", "", "cfg-secret", "wrong", errTokenInvalid},
		{"env 优先于配置", "env-secret", "cfg-secret", "env-secret", nil},
		{"env 优先时配置值不再被接受", "env-secret", "cfg-secret", "cfg-secret", errTokenInvalid},
		{"env 为纯空白 → 回落到配置，而不是被判成未配置", "   ", "cfg-secret", "cfg-secret", nil},
		{"配置值带尾随换行 → 归一", "", "cfg-secret\n", "cfg-secret", nil},
		{"两个来源都为空 → 未配置（fail-closed 503）", "", "", "anything", errTokenNotConfigured},
		{"两个来源都是空白 → 未配置", "  ", "\t", "anything", errTokenNotConfigured},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvBootstrapToken, tc.env)
			prev := config.Conf
			config.Conf = &pkgconfig.Config{}
			config.Conf.Install.BootstrapToken = tc.cfg
			t.Cleanup(func() { config.Conf = prev })

			if err := CheckBootstrapToken(tc.provided); err != tc.wantErr {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if want := tc.wantErr != errTokenNotConfigured; tokenConfigured() != want {
				t.Fatalf("tokenConfigured() = %v, want %v", tokenConfigured(), want)
			}
		})
	}
}

// config.Conf 为 nil 时不得 panic（独立运行/未加载配置的单元测试路径）。
func TestBootstrapTokenNilConfigIsSafe(t *testing.T) {
	prev := config.Conf
	config.Conf = nil
	t.Cleanup(func() { config.Conf = prev })

	t.Setenv(EnvBootstrapToken, "")
	if err := CheckBootstrapToken("x"); err != errTokenNotConfigured {
		t.Fatalf("Conf 为 nil 且 env 为空时应视为未配置, got %v", err)
	}

	t.Setenv(EnvBootstrapToken, "env-secret")
	if err := CheckBootstrapToken("env-secret"); err != nil {
		t.Fatalf("Conf 为 nil 不应影响环境变量来源, got %v", err)
	}
}
