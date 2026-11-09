package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/morehao/ark-iam/pkg/seed"
)

// columns 输出列定义：名称 + 取值函数。
//
// 显式列出全部 15 个字段（而不是用反射）的理由：字段一旦增删，这里会编译失败，
// 迫使人确认清单是否漏字段——而"漏字段"正是本工具要防的那个错。
var columns = []struct {
	title string
	value func(seed.BuiltinMenu) string
}{
	{"appCode", func(m seed.BuiltinMenu) string { return m.AppCode }},
	{"parentCode", func(m seed.BuiltinMenu) string { return m.ParentCode }},
	{"name", func(m seed.BuiltinMenu) string { return m.Name }},
	{"code", func(m seed.BuiltinMenu) string { return m.Code }},
	{"path", func(m seed.BuiltinMenu) string { return m.Path }},
	{"icon", func(m seed.BuiltinMenu) string { return m.Icon }},
	{"sort", func(m seed.BuiltinMenu) string { return fmt.Sprintf("%d", m.Sort) }},
	{"component", func(m seed.BuiltinMenu) string { return m.Component }},
	{"type", func(m seed.BuiltinMenu) string { return string(m.Type) }},
	{"visibility", func(m seed.BuiltinMenu) string { return string(m.Visibility) }},
	{"redirect", func(m seed.BuiltinMenu) string { return m.Redirect }},
	{"hidden", func(m seed.BuiltinMenu) string { return string(m.Hidden) }},
	{"externalLink", func(m seed.BuiltinMenu) string { return string(m.ExternalLink) }},
	{"keepAlive", func(m seed.BuiltinMenu) string { return string(m.KeepAlive) }},
	{"status", func(m seed.BuiltinMenu) string { return string(m.Status) }},
}

// Render 把内置菜单清单写成 Markdown。
//
// 输出必须**逐字节稳定**：同一份输入两次渲染结果相同（排序显式、无 map 迭代），
// 否则把它贴进变更单/工单后无法用 diff 核对"这次到底加了哪几个菜单"。
func Render(w io.Writer, menus []seed.BuiltinMenu) error {
	sorted := append([]seed.BuiltinMenu(nil), menus...)
	sortMenus(sorted)

	var b strings.Builder
	b.WriteString("# 内置菜单清单（`make print-builtin-menus`）\n\n")
	b.WriteString("L1 首次引导会写入下列菜单；**版本升级带来的新菜单不会自动下发**，\n")
	b.WriteString("请照本表在「菜单管理」页录入（先建 `type=directory` 的父级，再建子菜单）。\n\n")
	b.WriteString("`parentCode` 指向同表的 `code`；留空表示顶级菜单。\n\n")

	// 表头
	b.WriteString("| " + strings.Join(columnTitles(), " | ") + " |\n")
	b.WriteString("|" + strings.Repeat("---|", len(columns)) + "\n")
	for _, m := range sorted {
		cells := make([]string, 0, len(columns))
		for _, col := range columns {
			cells = append(cells, escapeCell(col.value(m)))
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}

	b.WriteString("\n## 字段说明\n\n")
	b.WriteString("- `type`：`directory` 为分组目录（无页面，`path`/`component` 留空）；`menu` 为页面。\n")
	b.WriteString("- `visibility`：`member` 任意登录成员可见；`admin` 仅管理员角色可见。\n")
	b.WriteString("- 后 5 列（`redirect` / `hidden` / `externalLink` / `keepAlive` / `status`）是**建表列默认值**，\n")
	b.WriteString("  种子不写这些字段；照本表填写即与种子产物一致。\n")
	b.WriteString("- 菜单 `code` 可改（种子按不可见的 `seed_key` 认行），但**改代码里的 `code` 不会改已有行的 `seed_key`**，\n")
	b.WriteString("  因此升级时新增菜单请用**新的 `code`**，不要复用已下线菜单的 `code`。\n")
	b.WriteString("\n## 授权\n\n")
	b.WriteString("- 平台 admin 角色：`dashboard`、`menu`、`tenant`、`application`、`tenant-application`、\n")
	b.WriteString("  `oauth-client`、`domain`、`department`、`tenant-user`、`tenant-role`。\n")
	b.WriteString("- 租户 admin 角色：`department`、`tenant-user`、`tenant-role`、`tenant-api-key`。\n")
	b.WriteString("- 目录菜单不参与授权（`role_menu` 不含 `directory`）。\n")

	_, err := io.WriteString(w, b.String())
	return err
}

// columnTitles 返回全部列名。
func columnTitles() []string {
	titles := make([]string, 0, len(columns))
	for _, col := range columns {
		titles = append(titles, col.title)
	}
	return titles
}

// sortMenus 显式排序：应用 → 层级（顶级在前，子级紧随其父）→ 定义顺序。
//
// 用"应用 + 顶级 sort + 是否子级 + 父级 sort + 自身 sort + code"全序排序，
// 保证不依赖 builtinMenuDefs 的书写顺序也不会抖动。
func sortMenus(menus []seed.BuiltinMenu) {
	rank := func(m seed.BuiltinMenu) int {
		if m.ParentCode == "" {
			return 0
		}
		return 1
	}
	parentSort := make(map[string]int, len(menus))
	for _, m := range menus {
		if m.ParentCode == "" {
			parentSort[m.AppCode+"\x00"+m.Code] = m.Sort
		}
	}
	sort.SliceStable(menus, func(i, j int) bool {
		a, b := menus[i], menus[j]
		if a.AppCode != b.AppCode {
			return a.AppCode < b.AppCode
		}
		// 顶级菜单之间按其 sort，子菜单排在其后
		if rank(a) != rank(b) {
			return rank(a) < rank(b)
		}
		if rank(a) == 0 {
			if a.Sort != b.Sort {
				return a.Sort < b.Sort
			}
			return a.Code < b.Code
		}
		pa, pb := parentSort[a.AppCode+"\x00"+a.ParentCode], parentSort[b.AppCode+"\x00"+b.ParentCode]
		if pa != pb {
			return pa < pb
		}
		if a.ParentCode != b.ParentCode {
			return a.ParentCode < b.ParentCode
		}
		if a.Sort != b.Sort {
			return a.Sort < b.Sort
		}
		return a.Code < b.Code
	})
}

// emptyCell 空值的占位渲染：留空的单元格在 Markdown 里看不出"是空还是漏了"。
const emptyCell = "`-`"

// escapeCell 转义 Markdown 表格里的竖线，避免路径把表格切断。
func escapeCell(v string) string {
	if v == "" {
		return emptyCell
	}
	return strings.ReplaceAll(v, "|", `\|`)
}
