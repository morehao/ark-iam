package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
)

// renderString 渲染清单，便于断言。
func renderString(t *testing.T, menus []seed.BuiltinMenu) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, menus); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// TestRender_CoversAllFifteenFields 清单必须覆盖全部 15 个字段——漏字段正是本工具要防的错。
func TestRender_CoversAllFifteenFields(t *testing.T) {
	got := renderString(t, seed.BuiltinMenus())
	for _, title := range []string{
		"appCode", "parentCode", "name", "code", "path", "icon", "sort", "component",
		"type", "visibility", "redirect", "hidden", "externalLink", "keepAlive", "status",
	} {
		if !strings.Contains(got, title) {
			t.Errorf("输出缺少字段列 %q：清单漏字段会让运维照抄出错误的菜单", title)
		}
	}
	if len(columns) != 15 {
		t.Errorf("columns 数 = %d, want 15", len(columns))
	}
}

// TestRender_IncludesEveryBuiltinMenuCode 每个内置菜单的 code 都必须出现在清单里。
func TestRender_IncludesEveryBuiltinMenuCode(t *testing.T) {
	menus := seed.BuiltinMenus()
	if len(menus) != 14 {
		t.Fatalf("内置菜单数 = %d, want 14", len(menus))
	}
	got := renderString(t, menus)
	for _, m := range menus {
		if !strings.Contains(got, m.Code) {
			t.Errorf("输出缺少菜单 code %q", m.Code)
		}
	}
}

// TestRender_IsByteStable 同一份输入两次渲染必须逐字节相同。
//
// 否则把清单贴进工单后无法用 diff 核对"这次到底加了哪几个菜单"。
func TestRender_IsByteStable(t *testing.T) {
	menus := seed.BuiltinMenus()
	first := renderString(t, menus)
	second := renderString(t, menus)
	if first != second {
		t.Error("两次渲染结果不一致：输出不稳定（排序或 map 迭代引入了随机性）")
	}

	// 打乱输入顺序后仍应得到同一份输出（排序必须是全序）
	shuffled := append([]seed.BuiltinMenu(nil), menus...)
	for i, j := 0, len(shuffled)-1; i < j; i, j = i+1, j-1 {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	if again := renderString(t, shuffled); again != first {
		t.Error("打乱输入顺序后输出改变：排序不是全序")
	}
}

// tableRows 解析输出里的数据行，返回 (code, parentCode) 序列。
//
// 不能用 strings.Index("| menu |") 这类子串判断顺序：type 列的取值恰好也是 menu，
// 会命中别的行。按列解析才可靠。
func tableRows(t *testing.T, out string) [][2]string {
	t.Helper()
	var rows [][2]string
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) != len(columns) {
			continue // 表头/分隔行
		}
		appCode := strings.TrimSpace(cells[0])
		if appCode != "platform_admin" && appCode != "tenant_admin" {
			continue
		}
		rows = append(rows, [2]string{strings.TrimSpace(cells[3]), strings.TrimSpace(cells[1])})
	}
	return rows
}

// TestRender_OrderIsParentBeforeChild 顶级菜单全部排在子菜单之前（照抄时先建父级），
// 且整体顺序必须是确定的一份清单。
func TestRender_OrderIsParentBeforeChild(t *testing.T) {
	rows := tableRows(t, renderString(t, seed.BuiltinMenus()))
	if len(rows) != 14 {
		t.Fatalf("解析到 %d 个数据行, want 14", len(rows))
	}
	// 顶级按 sort（工作台 → 租户中心 → 应用中心 → 平台管理），随后是子菜单（按其父级 sort）
	wantOrder := []string{
		"dashboard", "grp-tenant", "grp-app", "grp-platform",
		"tenant", "tenant-application", "domain",
		"application", "oauth-client", "menu",
		"department", "tenant-user", "tenant-role", "tenant-api-key",
	}
	for i, want := range wantOrder {
		if rows[i][0] != want {
			t.Errorf("第 %d 行 code = %q, want %q（完整顺序 %v）", i+1, rows[i][0], want, rows)
		}
	}

	// 父级必须出现在其子级之前
	indexOf := func(code string) int {
		for i, row := range rows {
			if row[0] == code {
				return i
			}
		}
		return -1
	}
	for _, row := range rows {
		if row[1] == emptyCell || row[1] == "" {
			continue
		}
		parent, child := indexOf(row[1]), indexOf(row[0])
		if parent < 0 {
			t.Errorf("子菜单 %s 的父级 %s 不在清单里", row[0], row[1])
			continue
		}
		if parent > child {
			t.Errorf("子菜单 %s 排在其父级 %s 之前", row[0], row[1])
		}
	}
}

// TestRender_ContainsSeededDefaults 后 5 列必须印出真实的列默认值（而不是留空）。
func TestRender_ContainsSeededDefaults(t *testing.T) {
	got := renderString(t, seed.BuiltinMenus())
	for _, want := range []string{
		string(model.MenuHiddenFlagDisable),
		string(model.MenuExternalLinkFlagDisable),
		string(model.MenuKeepAliveFlagDisable),
		string(model.MenuStatusEnable),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("输出缺少列默认值 %q", want)
		}
	}
}

// TestRender_EmptyPathRenderedAsDash 空值渲染成 `-` 而不是空白单元格，避免表格错位。
func TestRender_EmptyPathRenderedAsDash(t *testing.T) {
	got := renderString(t, []seed.BuiltinMenu{{
		AppCode: "platform_admin", Name: "目录", Code: "grp-x",
		Type: model.MenuTypeDirectory, Visibility: model.MenuVisibilityAdmin,
		Hidden: model.MenuHiddenFlagDisable, ExternalLink: model.MenuExternalLinkFlagDisable,
		KeepAlive: model.MenuKeepAliveFlagDisable, Status: model.MenuStatusEnable,
	}})
	if !strings.Contains(got, "| `-` |") {
		t.Error("空 path 未渲染成 `-`")
	}
}
