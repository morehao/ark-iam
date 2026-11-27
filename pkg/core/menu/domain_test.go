package menu

import (
	"context"
	"errors"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objpermission"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

// getter 把一个固定 *gorm.DB 包装为 dao 层接受的 DBGetter（领域函数按注入的 getter 取库）。
func getter(db *gorm.DB) gormdao.DBGetter {
	return func(ctx context.Context) *gorm.DB { return db }
}

// TestValidateMenuEnums 覆盖枚举校验：合法值通过；Type/Status/Visibility 非法值拒绝；
// 三个开关允许空串（未提供）与合法枚举，非法值拒绝。
func TestValidateMenuEnums(t *testing.T) {
	base := func() objpermission.MenuBaseInfo {
		return objpermission.MenuBaseInfo{
			Type:       model.MenuTypeMenu,
			Status:     model.MenuStatusEnable,
			Visibility: model.MenuVisibilityAdmin,
		}
	}

	// 全合法：每个枚举的合法取值都应通过
	validCases := []objpermission.MenuBaseInfo{
		base(),
		{Type: model.MenuTypeDirectory, Status: model.MenuStatusDisable, Visibility: model.MenuVisibilityPublic},
		{Type: model.MenuTypeButton, Status: model.MenuStatusEnable, Visibility: model.MenuVisibilityMember},
		{Type: model.MenuTypeMenu, Status: model.MenuStatusEnable, Visibility: model.MenuVisibilityAdmin,
			Hidden: model.MenuHiddenFlagEnable, ExternalLink: model.MenuExternalLinkFlagDisable, KeepAlive: model.MenuKeepAliveFlagEnable},
		// 开关空串（未提供）合法
		{Type: model.MenuTypeMenu, Status: model.MenuStatusEnable, Visibility: model.MenuVisibilityAdmin,
			Hidden: "", ExternalLink: "", KeepAlive: ""},
	}
	for i, c := range validCases {
		req := c
		if !ValidateMenuEnums(&req) {
			t.Fatalf("合法枚举 case %d 应通过: %+v", i, req)
		}
	}

	// 非法：各字段逐一破坏
	invalid := []struct {
		name string
		mut  func(*objpermission.MenuBaseInfo)
	}{
		{"bad type", func(r *objpermission.MenuBaseInfo) { r.Type = model.MenuType("page") }},
		{"bad status", func(r *objpermission.MenuBaseInfo) { r.Status = model.MenuStatus("online") }},
		{"bad visibility", func(r *objpermission.MenuBaseInfo) { r.Visibility = model.MenuVisibility("all") }},
		{"bad hidden", func(r *objpermission.MenuBaseInfo) { r.Hidden = model.MenuHiddenFlag("yes") }},
		{"bad externalLink", func(r *objpermission.MenuBaseInfo) { r.ExternalLink = model.MenuExternalLinkFlag("true") }},
		{"bad keepAlive", func(r *objpermission.MenuBaseInfo) { r.KeepAlive = model.MenuKeepAliveFlag("1") }},
	}
	for _, tc := range invalid {
		req := base()
		tc.mut(&req)
		if ValidateMenuEnums(&req) {
			t.Fatalf("非法枚举[%s]应被拒: %+v", tc.name, req)
		}
	}
}

// TestNormalizeMenuSwitches 空串归一为 disable，已提供的值保持不变。
func TestNormalizeMenuSwitches(t *testing.T) {
	// 全空串 → 全部归一 disable
	empty := &objpermission.MenuBaseInfo{}
	NormalizeMenuSwitches(empty)
	if empty.Hidden != model.MenuHiddenFlagDisable ||
		empty.ExternalLink != model.MenuExternalLinkFlagDisable ||
		empty.KeepAlive != model.MenuKeepAliveFlagDisable {
		t.Fatalf("空串应全部归一 disable, got %+v", empty)
	}

	// 已提供 enable → 保持 enable，不被改写
	set := &objpermission.MenuBaseInfo{
		Hidden:       model.MenuHiddenFlagEnable,
		ExternalLink: model.MenuExternalLinkFlagEnable,
		KeepAlive:    model.MenuKeepAliveFlagEnable,
	}
	NormalizeMenuSwitches(set)
	if set.Hidden != model.MenuHiddenFlagEnable ||
		set.ExternalLink != model.MenuExternalLinkFlagEnable ||
		set.KeepAlive != model.MenuKeepAliveFlagEnable {
		t.Fatalf("已提供的 enable 不应被改写, got %+v", set)
	}

	// 混合：部分空串、部分 enable
	mixed := &objpermission.MenuBaseInfo{Hidden: model.MenuHiddenFlagEnable}
	NormalizeMenuSwitches(mixed)
	if mixed.Hidden != model.MenuHiddenFlagEnable {
		t.Fatalf("hidden 应保持 enable, got %q", mixed.Hidden)
	}
	if mixed.ExternalLink != model.MenuExternalLinkFlagDisable || mixed.KeepAlive != model.MenuKeepAliveFlagDisable {
		t.Fatalf("空串字段应归一 disable, got %+v", mixed)
	}
}

// TestMenuParentUsable 覆盖父菜单校验的三种结果：
//   - 根菜单（parentID 空）→ (nil, nil)，可用；
//   - 父存在且同应用 → 返回父实体，可用；
//   - 父不存在 / 空 ID / 不属于该应用 → ErrParentUnusable（业务性不可用，与系统错误分离）。
func TestMenuParentUsable(t *testing.T) {
	db := newMenuTestDB(t)
	app := seedOrderTestApp(t, db)
	otherApp := &model.ApplicationEntity{
		Code:   "other_app",
		Name:   "另一个应用",
		Source: model.AppSourceBuiltin,
		Status: model.AppStatusEnable,
	}
	if err := db.Create(otherApp).Error; err != nil {
		t.Fatalf("seed other app: %v", err)
	}
	seedMenu(t, db, "m-parent", app.ID, "", "parent", "父菜单", 1)
	seedMenu(t, db, "m-other", otherApp.ID, "", "other", "他应用菜单", 1)

	// 根菜单：parentID 空 → 合法
	parent, err := MenuParentUsable(getter(db), context.Background(), app.ID, "")
	if err != nil {
		t.Fatalf("根菜单不应报错: %v", err)
	}
	if parent != nil {
		t.Fatalf("根菜单应返回 nil 实体, got %+v", parent)
	}

	// 父存在且同应用 → 可可用，返回父实体
	parent, err = MenuParentUsable(getter(db), context.Background(), app.ID, "m-parent")
	if err != nil {
		t.Fatalf("同应用父菜单应可用: %v", err)
	}
	if parent == nil || parent.ID != "m-parent" {
		t.Fatalf("应返回父实体 m-parent, got %+v", parent)
	}

	// 父不存在 → ErrParentUnusable（非系统错误）
	parent, err = MenuParentUsable(getter(db), context.Background(), app.ID, "no-such-menu")
	if !errors.Is(err, ErrParentUnusable) {
		t.Fatalf("父不存在应返回 ErrParentUnusable, got err=%v parent=%+v", err, parent)
	}
	if parent != nil {
		t.Fatalf("父不存在不应返回实体, got %+v", parent)
	}

	// 父存在但属于别的应用 → ErrParentUnusable
	parent, err = MenuParentUsable(getter(db), context.Background(), app.ID, "m-other")
	if !errors.Is(err, ErrParentUnusable) {
		t.Fatalf("跨应用父菜单应返回 ErrParentUnusable, got err=%v parent=%+v", err, parent)
	}
	if parent != nil {
		t.Fatalf("跨应用父菜单不应返回实体, got %+v", parent)
	}
}

// TestCollectSubtreeIDs 覆盖子树收集：返回根及其全部子孙（广度优先，限定同一应用），
// 不包含其它应用的菜单，也不包含兄弟分支。
func TestCollectSubtreeIDs(t *testing.T) {
	db := newMenuTestDB(t)
	app := seedOrderTestApp(t, db)
	otherApp := &model.ApplicationEntity{
		Code:   "other_app2",
		Name:   "另一个应用2",
		Source: model.AppSourceBuiltin,
		Status: model.AppStatusEnable,
	}
	if err := db.Create(otherApp).Error; err != nil {
		t.Fatalf("seed other app: %v", err)
	}

	// 树：root → (c1 → (g1), c2); 另有兄弟 sibling 与其它应用 menu
	seedMenu(t, db, "root", app.ID, "", "root", "根", 1)
	seedMenu(t, db, "c1", app.ID, "root", "c1", "子1", 1)
	seedMenu(t, db, "c2", app.ID, "root", "c2", "子2", 2)
	seedMenu(t, db, "g1", app.ID, "c1", "g1", "孙1", 1)
	seedMenu(t, db, "sibling", app.ID, "", "sibling", "兄弟", 2)
	seedMenu(t, db, "foreign", otherApp.ID, "", "foreign", "他应用", 1)

	ids, err := CollectSubtreeIDs(getter(db), context.Background(), app.ID, "root")
	if err != nil {
		t.Fatalf("CollectSubtreeIDs: %v", err)
	}

	got := make(map[string]bool, len(ids))
	for _, id := range ids {
		got[id] = true
	}
	want := []string{"root", "c1", "c2", "g1"}
	if len(ids) != len(want) {
		t.Fatalf("子树应含 %d 个节点, got %d (%v)", len(want), len(ids), ids)
	}
	for _, id := range want {
		if !got[id] {
			t.Fatalf("子树应包含 %q, got %v", id, ids)
		}
	}
	if got["sibling"] {
		t.Fatalf("子树不应包含兄弟分支 sibling, got %v", ids)
	}
	if got["foreign"] {
		t.Fatalf("子树不应包含其它应用的菜单 foreign, got %v", ids)
	}

	// 叶子节点：只返回自身
	leafIDs, err := CollectSubtreeIDs(getter(db), context.Background(), app.ID, "g1")
	if err != nil {
		t.Fatalf("CollectSubtreeIDs leaf: %v", err)
	}
	if len(leafIDs) != 1 || leafIDs[0] != "g1" {
		t.Fatalf("叶子子树应只含自身, got %v", leafIDs)
	}
}
