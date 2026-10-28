package dbclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 本文件是三条设计红线的可执行守卫（见 docs/design/context-and-tenant-scope.md
// 的「2026-10-27 边界调整」，以及 golib v1.33.0 biz/gcontext/README.md「身份不经过 context」）：
//
//	① 内层零 gin（TestServiceLayerHasNoGinDependency）：service / dao / core / audit 等
//	   非 controller / middleware 的生产代码，不得出现 *gin.Context，也不得 import
//	   gincontext —— 它们只收 context.Context。
//	② 唯一转换点（TestRequestContextOnlyAtControllerBoundary）：`.Request.Context()`
//	   只允许出现在 controller / middleware 目录（入站适配体）。
//	③ 业务层无 gin 侧身份契约（TestBusinessLayerDoesNotImportGinIdentity）：service / dao /
//	   core / audit 不得 import pkg/identity —— 身份走 DTO（req.Operator），
//	   业务层不读 gin Keys。这一条防止改造后回流。
//
// 守卫用"扫描真实源码"而非"约定"来保证：约束被打破时测试当场失败，并在失败信息里
// 指出违规文件与行号。
//
// 豁免的"入站适配体/夹具"（它们本来就是 gin 边界的实现处）：
//   - pkg/identity：gin Keys 的读写契约本身；
//   - pkg/sso：Cookie 会话读写；
//   - pkg/oidckit：OIDC 回调适配；
//   - pkg/middleware：鉴权/策略中间件（gin 中间件就是它的产物）；
//   - pkg/testsetup、各 app 的 testutil：测试夹具。
const repoRootRelPath = "../../.."

// guardedGoDirs 是允许出现业务代码的目录（相对于仓库根）。
var guardedGoDirs = []string{"backend/pkg", "backend/apps"}

// skippedDirNames 不参与扫描（历史副本、依赖、工具产物）。
var skippedDirNames = map[string]bool{
	".git": true, ".history": true, "vendor": true, "node_modules": true, "dist": true,
}

// inboundBoundaryPathSegments 是允许直接接触 gin 的目录段（入站适配体）。
var inboundBoundaryPathSegments = []string{
	"/controller/", "/middleware/",
}

// ginAdapterPkgs 是本身即 gin 边界实现/夹具的包（相对 backend/ 的路径前缀）。
var ginAdapterPkgs = []string{
	"pkg/identity/",
	"pkg/sso/",
	"pkg/oidckit/",
	"pkg/middleware/",
	"pkg/testsetup/",
	"apps/auth/internal/middleware/",
	"apps/rpapi/internal/middleware/",
	"/testutil/",
}

// innerLayerDirs 是被要求"零 gin"的内层目录（相对 backend/）。
// 这些目录下的生产代码只收 context.Context。
var innerLayerDirs = []string{
	"apps/auth/internal/service",
	"apps/platformadmin/internal/service",
	"apps/tenantadmin/internal/service",
	"apps/rpapi/internal/service",
	"apps/auth/internal/core",
	"apps/platformadmin/internal/core",
	"apps/tenantadmin/internal/core",
	"apps/rpapi/internal/core",
	"apps/auth/internal/dao",
	"apps/platformadmin/internal/dao",
	"apps/tenantadmin/internal/dao",
	"apps/rpapi/internal/dao",
	"pkg/audit",
	"pkg/core",
	"pkg/dao",
	"pkg/model",
	"pkg/seed",
	"pkg/dbclient",
	"pkg/credential",
	"pkg/code",
	"pkg/config",
	"pkg/object",
}

// TestServiceLayerHasNoGinDependency 守护①：内层代码不出现 *gin.Context、不 import gincontext。
//
// 身份是业务输入（DTO 的 Operator 字段），租户作用域随 context.Context 传递；
// 内层若还持有 gin 上下文，说明改造漏点，且"传 gin ctx 还是纯 ctx 结果不同"的分支会重新长出。
func TestServiceLayerHasNoGinDependency(t *testing.T) {
	root := filepath.Join(repoRootRelPath)
	require.DirExists(t, root, "仓库根目录必须存在（守卫依赖 backend/ 布局）")

	var violations []string
	for _, dir := range innerLayerDirs {
		abs := filepath.Join(root, "backend", dir)
		if _, err := os.Stat(abs); err != nil {
			continue // 目录可能尚未创建（如空域），跳过
		}
		walkGoFiles(t, abs, func(path string, lines []string) {
			if strings.HasSuffix(path, "_test.go") {
				return
			}
			for i, line := range lines {
				if !isCodeLine(line) {
					continue
				}
				if strings.Contains(line, "*gin.Context") {
					violations = append(violations, formatViolation(root, path, i+1, line))
				}
				if strings.Contains(line, "gincontext") {
					violations = append(violations, formatViolation(root, path, i+1, line))
				}
			}
		})
	}
	require.Empty(t, violations,
		"内层（service/dao/core/audit 等）只收 context.Context：身份填进 DTO 的 Operator 字段，"+
			"租户用 dbclient.TenantIDFromContext(ctx) 读取；gin 上下文只允许出现在 controller/middleware")
}

// TestRequestContextOnlyAtControllerBoundary 守护②：`.Request.Context()` 只在入站适配体里出现。
//
// 这是 controller → service 的**唯一一次**载体转换：service/dao 拿到的是纯 ctx，
// 因此租户作用域/取消/trace 的传递与 gin 引擎开关无关。
func TestRequestContextOnlyAtControllerBoundary(t *testing.T) {
	root := filepath.Join(repoRootRelPath)
	require.DirExists(t, root, "仓库根目录必须存在（守卫依赖 backend/ 布局）")

	var violations []string
	walkGoFiles(t, filepath.Join(root, "backend"), func(path string, lines []string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		slashPath := filepath.ToSlash(path)
		if isInboundBoundary(slashPath) {
			return
		}
		for i, line := range lines {
			if !isCodeLine(line) {
				continue
			}
			if strings.Contains(line, ".Request.Context(") {
				violations = append(violations, formatViolation(root, path, i+1, line))
			}
		}
	})
	require.Empty(t, violations,
		"`.Request.Context()` 只允许出现在 controller / middleware 目录："+
			"那里是 gin 载体与标准 context.Context 的唯一转换点")
}

// TestBusinessLayerDoesNotImportGinIdentity 守护③：service/dao/core/audit 不得 import pkg/identity。
//
// pkg/identity 是 gin 侧身份契约（gin Keys 读写）。业务层若 import 它，就会绕过 DTO
// 重新从上下文"捞"身份，异步路径（AsyncContext 不带走 gin Keys 身份）也会静默丢身份。
func TestBusinessLayerDoesNotImportGinIdentity(t *testing.T) {
	root := filepath.Join(repoRootRelPath)
	require.DirExists(t, root, "仓库根目录必须存在（守卫依赖 backend/ 布局）")

	var violations []string
	for _, dir := range innerLayerDirs {
		abs := filepath.Join(root, "backend", dir)
		if _, err := os.Stat(abs); err != nil {
			continue
		}
		walkGoFiles(t, abs, func(path string, lines []string) {
			if strings.HasSuffix(path, "_test.go") {
				return
			}
			for i, line := range lines {
				if strings.Contains(line, `"github.com/morehao/ark-iam/pkg/identity"`) {
					violations = append(violations, formatViolation(root, path, i+1, line))
				}
			}
		})
	}
	require.Empty(t, violations,
		"业务层不得 import pkg/identity（gin 侧身份契约）：身份走 DTO 的 Operator 字段，"+
			"controller 用 identity.OperatorFromGin 填一次即可")
}

// TestTenantScopeLayerDoesNotDependOnGin 守护租户隔离层与 gin 解耦（无载体断言）。
func TestTenantScopeLayerDoesNotDependOnGin(t *testing.T) {
	root := filepath.Join(repoRootRelPath)
	require.DirExists(t, root, "仓库根目录必须存在（守卫依赖 backend/ 布局）")

	var violations []string
	walkGoFiles(t, filepath.Join(root, "backend/pkg/dbclient"), func(path string, lines []string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		for i, line := range lines {
			if strings.Contains(line, "gin-gonic/gin") {
				violations = append(violations, formatViolation(root, path, i+1, line))
			}
			if strings.Contains(line, ".(*gin.Context)") {
				violations = append(violations, formatViolation(root, path, i+1, line))
			}
		}
	})
	require.Empty(t, violations,
		"租户隔离层不得依赖 gin：作用域从 context 键值解析（gcontext.TenantScope），"+
			"载体断言会让隔离随调用方 ctx 类型变化")
}

// isInboundBoundary 判断该文件是否属于入站适配体（controller/middleware 目录）或 gin 夹具包。
func isInboundBoundary(slashPath string) bool {
	for _, seg := range inboundBoundaryPathSegments {
		if strings.Contains(slashPath, seg) {
			return true
		}
	}
	idx := strings.LastIndex(slashPath, "/backend/")
	if idx >= 0 {
		rel := slashPath[idx+len("/backend/"):]
		for _, adapter := range ginAdapterPkgs {
			if strings.HasPrefix(rel, adapter) || strings.Contains(rel, adapter) {
				return true
			}
		}
	}
	return false
}

// isCodeLine 判断该行是否为真实代码（非注释、非字符串外的说明文字）。
//
// 守卫扫描的是"代码是否违规"，注释里提到 *gin.Context / .Request.Context() /
// gincontext 属于文档说明（如"controller 用 ctx.Request.Context() 做唯一转换"），
// 不应算违规——否则守卫会逼着把解释性注释删掉。
func isCodeLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") && !strings.HasPrefix(trimmed, "*")
}

// walkGoFiles 递归遍历目录下的 .go 文件并回传其内容行。
func walkGoFiles(t *testing.T, dir string, visit func(path string, lines []string)) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDirNames[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		visit(path, strings.Split(string(content), "\n"))
		return nil
	})
	require.NoError(t, err)
}

// formatViolation 输出可直接定位的违规信息。
func formatViolation(root, path string, lineNo int, line string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	return rel + ":" + itoa(lineNo) + ": " + strings.TrimSpace(line)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
