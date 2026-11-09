// Command printbuiltinmenus 输出内置菜单的完整字段清单（Markdown），供版本升级时
// 在「菜单管理」页手工补齐新菜单。
//
// 为什么需要它：L1 引导只在首次初始化时写入菜单，之后菜单行归运维（控制台可增删改）。
// 因此版本升级带来的新菜单**不会**被自动下发，必须由运维照本清单录入。
//
// 用法：
//
//	cd backend && go run ./pkg/cmd/printbuiltinmenus
//	make print-builtin-menus
package main

import (
	"fmt"
	"os"

	"github.com/morehao/ark-iam/pkg/seed"
)

func main() {
	if err := Render(os.Stdout, seed.BuiltinMenus()); err != nil {
		fmt.Fprintf(os.Stderr, "print builtin menus: %v\n", err)
		os.Exit(1)
	}
}
