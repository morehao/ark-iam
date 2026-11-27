package dao

// 本文件是 P0 多 issuer 的可执行 design-guard（DG-CLIENT-4）：
// application_client.code 的全局唯一已删除，改为 (issuer, code) 唯一。所有「按 code 查客户端」
// 的生产代码必须同时带 Issuer，否则会跨 issuer 命中错误客户端（AC-6 / p0-issuer-context §2.1）。
//
// 扫描规则：遍历 pkg/ 与 apps/ 下全部非 _test.go 的 .go 文件，找到 `ApplicationClientCond{`
// 的字面量块（到配对的 `}` 为止），若块内出现 `Code:` 但**没有** `Issuer:`，即视为违规。
// 例外：PageList 等纯列表筛选的 Cond 不含 Code（不会被命中）；本文件本身与 _test.go 不计。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuard_ApplicationClientCondMustCarryIssuer(t *testing.T) {
	repoRoot := findRepoRoot(t)
	var violations []string
	roots := []string{filepath.Join(repoRoot, "pkg"), filepath.Join(repoRoot, "apps")}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				// 跳过 vendor/前端目录
				if info.Name() == "node_modules" || info.Name() == "vendor" || info.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			src := string(data)
			if lines := scanCondBlocksForCodeWithoutIssuer(src); len(lines) > 0 {
				rel, _ := filepath.Rel(repoRoot, path)
				for _, ln := range lines {
					violations = append(violations, rel+":"+ln)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("application_client 按 code 查询必须带 Issuer（(issuer, code) 唯一），以下位置缺失:\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// scanCondBlocksForCodeWithoutIssuer 返回违规块起始行号（1-based，字符串）。
func scanCondBlocksForCodeWithoutIssuer(src string) []string {
	var out []string
	idx := 0
	for {
		pos := strings.Index(src[idx:], "ApplicationClientCond{")
		if pos < 0 {
			break
		}
		start := idx + pos
		// 找到配对的花括号块
		braceStart := start + strings.Index(src[start:], "{")
		depth := 0
		end := -1
		for i := braceStart; i < len(src); i++ {
			switch src[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = i
				}
			}
			if end >= 0 {
				break
			}
		}
		if end < 0 {
			break
		}
		block := src[braceStart : end+1]
		if strings.Contains(block, "Code:") && !strings.Contains(block, "Issuer:") {
			line := 1 + strings.Count(src[:start], "\n")
			out = append(out, itoa(line))
		}
		idx = end + 1
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.work not found upward from cwd")
		}
		dir = parent
	}
}
