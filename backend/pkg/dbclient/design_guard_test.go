package dbclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 本文件是两条设计红线的可执行守卫（评审文档 §1 的两条硬约束）：
//
//	① 业务/基础设施代码一律直传 gin 上下文，不得出现 `ctx.Request.Context()`
//	   （异步续跑统一走 gincontext.AsyncContext，跨 http.Handler 的搬运统一走
//	   gincontext.WithRequestValue）；
//	② 租户隔离层不得依赖 *gin.Context 载体——不引入 gin 包、不做载体类型断言，
//	   作用域只从 context 的键值解析。
//
// 守卫用"扫描真实源码"而非"约定"来保证：约束被打破时测试当场失败，并在失败信息里
// 指出违规文件与行号。

const repoRootRelPath = "../../.."

// guardedGoDirs 是允许出现业务代码的目录（相对于仓库根）。
var guardedGoDirs = []string{"backend/pkg", "backend/apps"}

// skippedDirNames 不参与扫描（历史副本、依赖、工具产物）。
var skippedDirNames = map[string]bool{
	".git": true, ".history": true, "vendor": true, "node_modules": true, "dist": true,
}

// TestNoRequestContextInProductionCode 守护约束①：非测试 Go 代码里零 `ctx.Request.Context()`。
func TestNoRequestContextInProductionCode(t *testing.T) {
	root := filepath.Join(repoRootRelPath)
	require.DirExists(t, root, "仓库根目录必须存在（守卫依赖 backend/ 布局）")

	var violations []string
	for _, dir := range guardedGoDirs {
		walkGoFiles(t, filepath.Join(root, dir), func(path string, lines []string) {
			if strings.HasSuffix(path, "_test.go") {
				return
			}
			for i, line := range lines {
				if strings.Contains(line, ".Request.Context()") {
					violations = append(violations, formatViolation(root, path, i+1, line))
				}
			}
		})
	}
	require.Empty(t, violations,
		"禁止 ctx.Request.Context()：业务侧一路直传 *gin.Context；异步用 gincontext.AsyncContext，"+
			"跨 http.Handler 的取值搬运用 gincontext.WithRequestValue")
}

// TestTenantScopeLayerDoesNotDependOnGin 守护约束②：隔离层与 gin 解耦（无载体断言）。
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
