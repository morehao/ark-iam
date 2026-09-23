package seed_test

import (
	"context"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
)

// TestBuiltinMenus_UniqueAndComplete 清单本身的结构约束。
func TestBuiltinMenus_UniqueAndComplete(t *testing.T) {
	menus := seed.BuiltinMenus()
	if len(menus) != 14 {
		t.Fatalf("BuiltinMenus() 长度 = %d, want 14", len(menus))
	}
	seen := map[string]bool{}
	for _, m := range menus {
		if m.Code == "" {
			t.Error("菜单 code 为空")
		}
		if seen[m.Code] {
			t.Errorf("菜单 code %q 重复", m.Code)
		}
		seen[m.Code] = true
		if m.ParentCode != "" && !seen[m.ParentCode] {
			// 父级必须能在清单内解析（且必须已出现过，保证清单可自上而下照抄）
			t.Errorf("菜单 %s 的父级 %s 不在其之前出现", m.Code, m.ParentCode)
		}
		if m.AppCode == "" {
			t.Errorf("菜单 %s 缺 appCode", m.Code)
		}
	}
}

// TestBuiltinMenus_MatchSeededRows 清单的 15 个字段必须与真实库行逐一相等。
//
// 这是 `make print-builtin-menus` 的**可信度来源**：清单里那 5 个"列默认值"字段
// （redirect/hidden/externalLink/keepAlive/status）如果与建表默认值不一致，
// 运维照抄出来的菜单就和 L1 首次初始化的产物不同——本用例把两者钉在一起。
func TestBuiltinMenus_MatchSeededRows(t *testing.T) {
	db := setupDB(t)
	if _, _, err := seed.Bootstrap(context.Background(), db, testDefinition(t)); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	bySeedKey := map[string]model.MenuEntity{}
	var rows []model.MenuEntity
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("query menus: %v", err)
	}
	for _, row := range rows {
		bySeedKey[row.SeedKey] = row
	}

	appCodeByID := map[string]string{}
	var apps []model.ApplicationEntity
	if err := db.Find(&apps).Error; err != nil {
		t.Fatalf("query applications: %v", err)
	}
	for _, app := range apps {
		appCodeByID[app.ID] = app.Code
	}

	for _, want := range seed.BuiltinMenus() {
		got, ok := bySeedKey[want.Code]
		if !ok {
			t.Errorf("清单里的菜单 %s 在库中不存在（seed_key 未写入？）", want.Code)
			continue
		}
		if appCodeByID[got.AppID] != want.AppCode {
			t.Errorf("菜单 %s appCode = %q, want %q", want.Code, appCodeByID[got.AppID], want.AppCode)
		}
		wantParentID := ""
		if want.ParentCode != "" {
			wantParentID = bySeedKey[want.ParentCode].ID
		}
		if got.ParentID != wantParentID {
			t.Errorf("菜单 %s parentID = %q, want %q", want.Code, got.ParentID, wantParentID)
		}
		if got.Name != want.Name {
			t.Errorf("菜单 %s name = %q, want %q", want.Code, got.Name, want.Name)
		}
		if got.Path != want.Path {
			t.Errorf("菜单 %s path = %q, want %q", want.Code, got.Path, want.Path)
		}
		if got.Icon != want.Icon {
			t.Errorf("菜单 %s icon = %q, want %q", want.Code, got.Icon, want.Icon)
		}
		if got.Sort != want.Sort {
			t.Errorf("菜单 %s sort = %d, want %d", want.Code, got.Sort, want.Sort)
		}
		if got.Component != want.Component {
			t.Errorf("菜单 %s component = %q, want %q", want.Code, got.Component, want.Component)
		}
		if got.Type != want.Type {
			t.Errorf("菜单 %s type = %q, want %q", want.Code, got.Type, want.Type)
		}
		if got.Visibility != want.Visibility {
			t.Errorf("菜单 %s visibility = %q, want %q", want.Code, got.Visibility, want.Visibility)
		}
		// 以下 5 个字段是清单里补全的"列默认值"——必须与库中实际值相等
		if got.Redirect != want.Redirect {
			t.Errorf("菜单 %s redirect = %q, want %q", want.Code, got.Redirect, want.Redirect)
		}
		if got.Hidden != want.Hidden {
			t.Errorf("菜单 %s hidden = %q, want %q", want.Code, got.Hidden, want.Hidden)
		}
		if got.ExternalLink != want.ExternalLink {
			t.Errorf("菜单 %s externalLink = %q, want %q", want.Code, got.ExternalLink, want.ExternalLink)
		}
		if got.KeepAlive != want.KeepAlive {
			t.Errorf("菜单 %s keepAlive = %q, want %q", want.Code, got.KeepAlive, want.KeepAlive)
		}
		if got.Status != want.Status {
			t.Errorf("菜单 %s status = %q, want %q", want.Code, got.Status, want.Status)
		}
	}
}

// TestBuiltinMenus_NoDirectoryGranted 目录菜单不参与授权：
// 清单里 type=directory 的菜单不得出现在任何 role_menu 行里。
//
// 这条同时解释了"升级时新增目录菜单不需要补授权"。
func TestBuiltinMenus_NoDirectoryGranted(t *testing.T) {
	db := setupDB(t)
	if _, _, err := seed.Bootstrap(context.Background(), db, testDefinition(t)); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	directorySeedKeys := map[string]bool{}
	for _, m := range seed.BuiltinMenus() {
		if m.Type == model.MenuTypeDirectory {
			directorySeedKeys[m.Code] = true
		}
	}
	if len(directorySeedKeys) != 3 {
		t.Fatalf("目录菜单数 = %d, want 3（%v）", len(directorySeedKeys), directorySeedKeys)
	}

	var rows []model.MenuEntity
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("query menus: %v", err)
	}
	directoryIDs := map[string]string{}
	for _, row := range rows {
		if directorySeedKeys[row.SeedKey] {
			directoryIDs[row.ID] = row.SeedKey
		}
	}

	var grants []model.RoleMenuEntity
	if err := db.Find(&grants).Error; err != nil {
		t.Fatalf("query role_menu: %v", err)
	}
	for _, grant := range grants {
		if seedKey, ok := directoryIDs[grant.MenuID]; ok {
			t.Errorf("目录菜单 %s 被授权（role_menu id=%s）：目录不参与授权", seedKey, grant.ID)
		}
	}
}
