package router

import (
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/auth/internal/controller/ctrinstall"
)

// TestInstallRoutesRegistered /install 是自举入口（R3 例外）：状态查询公开可读，
// 初始化是唯一写入口。两条路由都必须注册，且**不得**被加进业务前缀（否则会被鉴权中间件拦掉）。
func TestInstallRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	installRouter(engine, ctrinstall.NewInstallCtr())

	routes := map[string]map[string]bool{}
	for _, route := range engine.Routes() {
		if routes[route.Path] == nil {
			routes[route.Path] = map[string]bool{}
		}
		routes[route.Path][route.Method] = true
	}
	if !routes["/install/status"][http.MethodGet] {
		t.Error("GET /install/status 未注册（页面与部署脚本靠它判断状态）")
	}
	if !routes["/install/initialize"][http.MethodPost] {
		t.Error("POST /install/initialize 未注册（唯一写入口）")
	}
	// /install 是自举专用前缀，不得混入业务前缀：否则会被业务鉴权中间件拦掉
	// （此刻还没有鉴权主体），且守卫的放行清单会退化成逐条白名单。
	for path := range routes {
		if strings.HasPrefix(path, "/v1/") {
			t.Errorf("初始化路由不得挂在业务前缀下: %s", path)
		}
	}
}

// TestSeedPackageHasNoAuditDependency pkg/seed 不得依赖 pkg/audit。
//
// 为什么这条要单独守：审计是**应用层**的横切关注点，而 pkg/seed 是"能被任何入口调用"的
// 领域写通道。一旦 seed 直接写审计，就会出现两个问题——种子写入的审计语义被迫与
// install 的请求上下文耦合；以及将来任何新调用方都会"顺带"产生审计记录，
// "谁触发了这次写入"这个信息就不可信了。
// 正确的分工是：pkg/seed 只写业务数据并返回变更报告，审计由调用方（ctrinstall）按报告写。
func TestSeedPackageHasNoAuditDependency(t *testing.T) {
	seedDir := filepath.Join(repoRootForDeps(t), "backend", "pkg", "seed")
	forbidden := map[string]string{
		"github.com/morehao/ark-iam/pkg/audit":  "pkg/seed 不得 import pkg/audit（审计由调用方按 Report 写）",
		"github.com/morehao/ark-iam/pkg/config": "pkg/seed 不得 import pkg/config（配置由调用方翻译成 Definition）",
	}
	entries, err := os.ReadDir(seedDir)
	if err != nil {
		t.Fatalf("read %s: %v", seedDir, err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(seedDir, e.Name())
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		checked++
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if reason, bad := forbidden[p]; bad {
				t.Errorf("%s 违规 import %s：%s", e.Name(), p, reason)
			}
		}
	}
	if checked == 0 {
		t.Fatal("未解析到任何 pkg/seed 源文件——守卫本身失效了")
	}
}

// TestInstallServiceDoesNotWriteAudit 审计只能由控制器写。
//
// 理由同 pkg/seed：service 是"可以被任何入口复用"的一层，把审计写在 service 会让
// "谁触发了这次操作"退化成不可信（例如未来加一个 CLI 引导入口，它会带着 HTTP 的
// actor 信息写审计，或者干脆写不出来）。
func TestInstallServiceDoesNotWriteAudit(t *testing.T) {
	svcDir := filepath.Join(repoRootForDeps(t), "backend", "apps", "auth", "internal", "service", "svcinstall")
	entries, err := os.ReadDir(svcDir)
	if err != nil {
		t.Fatalf("read %s: %v", svcDir, err)
	}
	checked := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(svcDir, e.Name())
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		checked++
		for _, imp := range f.Imports {
			if strings.Trim(imp.Path.Value, `"`) == "github.com/morehao/ark-iam/pkg/audit" {
				t.Errorf("%s 违规 import pkg/audit：审计由 ctrinstall 按 Report 写", e.Name())
			}
		}
	}
	if checked == 0 {
		t.Fatal("未解析到任何 svcinstall 源文件")
	}
}

// repoRootForDeps 定位仓库根（以 AGENTS.md 为标记）。
func repoRootForDeps(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("未找到仓库根（AGENTS.md）")
	return ""
}
