package iam

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestNoGlobalDBInInjectedServices 静态守卫：iam 下各域 service 的业务代码（非 _test）里
// 不得出现「无参 dao.NewXxxDao()」调用——那会静默退回全局 IamDB，破坏可嵌入库的实例化语义。
// DAO 必须经 dao.WithDBGetter(<recv>.db) 注入。
//
// 允许的例外：service 构造函数里 `if db == nil { db = dbclient.IamDB }` 的兜底（供旧测试
// 直接构造服务时使用），以及注释。
func TestNoGlobalDBInInjectedServices(t *testing.T) {
	noArgDao := regexp.MustCompile(`dao\.New[A-Za-z]+Dao\(\)`)
	roots := []string{
		filepath.Join("platformadmin", "service"),
		filepath.Join("tenantadmin", "service"),
		filepath.Join("auth", "service"),
	}

	var offenders []string
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			for i, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") {
					continue // 注释豁免
				}
				if noArgDao.MatchString(line) {
					offenders = append(offenders, path+":"+strconv.Itoa(i+1))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("发现无参 dao.NewXxxDao() 调用（应改用 dao.WithDBGetter(recv.db)）:\n%s",
			strings.Join(offenders, "\n"))
	}
}
