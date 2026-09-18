package seed

import (
	"context"
	"errors"
	"fmt"

	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/gorm"
)

// retiredMenu 已下线菜单登记项。
//
// seedMenus 只做幂等 upsert（存在则回填字段、不存在则创建），不会移除历史版本写入的菜单行，
// 因此每下线一个菜单都必须在此登记：启动时清理其菜单行与 role_menu 授权绑定，
// 保证存量库与当前种子定义一致（否则菜单管理页会残留幽灵菜单）。
//
// menuSeedKey 是该项在种子里的稳定身份（= 该菜单定义时的 code）：运营在控制台改过 code
// 之后，仍能按 seed_key 认出并清理这一行。
type retiredMenu struct {
	appCode     string // 菜单所属应用编码（仅用于兜底按 (app_id, code) 认领未回填 seed_key 的历史行）
	menuSeedKey string // 菜单种子身份键
}

// retiredMenus 已下线菜单清单（只增不减，每项仅在存量库首次启动时命中一次）。
//
// 登记判据：把 base 版本的 seedMenus 与当前定义做差集，凡「曾在历史版本种子里出现过、
// 现在不再定义」的菜单都必须在此登记——seedMenus 不会下线菜单，漏登记即存量库死链。
var retiredMenus = []retiredMenu{
	// 平台端「API密钥监督」页已下线：跨租户只读监督价值有限，
	// 密钥的创建/吊销/删除与排查统一收敛到租户管理后台「API密钥」模块。
	{appCode: appCodeAdmin, menuSeedKey: "api-key"},
	// 平台端「身份中心」目录及其「用户管理」「角色管理」子菜单已下线：
	// 用户与角色按租户归属，读写入口全部收敛到租户管理后台，
	// 平台端不再提供跨租户用户目录与角色只读视图（避免绕过租户授权边界）。
	// 注意父目录与子菜单必须一并登记：只登记父目录会留下指向已删除页面的孤立子菜单，
	// 只登记子菜单会留下空目录。
	{appCode: appCodeAdmin, menuSeedKey: "grp-identity"},
	{appCode: appCodeAdmin, menuSeedKey: "user"},
	{appCode: appCodeAdmin, menuSeedKey: "role"},
	// 租户端「组织管理」页改名为「部门管理」（organization → department）：
	// 菜单 code/path/component 全变更，登记旧 code 以清理存量库残留
	// （seedMenus 只 upsert 不下线，漏登记即存量库死链菜单 + role_menu 脏授权）。
	{appCode: appCodeTenantAdmin, menuSeedKey: "organization"},
	// 平台端「审计日志」页已随 log 表下线：日志无写入方、无字段权威矩阵与查询契约，
	// 保留只会是空页（log 表、payload 列与 LogDao 已在同批删除）。
	{appCode: appCodeAdmin, menuSeedKey: "log"},
}

// pruneRetiredMenus 幂等清理已下线菜单：先解除 role_menu 授权绑定，再**物理删除**菜单行。
// 认行顺序与 seedMenus 一致：先按 seed_key（改过 code 的行也能认出），
// 再兜底按 (app_id, code) 认领尚未回填 seed_key 的历史行——兜底必须限定应用，
// 否则运营自建的同名菜单会被误删。已清理过的项在后续启动中查询不到，直接跳过。
//
// 这里刻意物理删除：退役是**版本级**下线（定义已从 seedMenus 移除），与运维在控制台的删除
// （软删，作为"种子不得复活"的墓碑，见 menuSeedKeyRemoved）是两种语义。若退役也留软删行，
// 未来版本重新上线同名菜单时会被墓碑挡住，永远创建不出来。
func pruneRetiredMenus(ctx context.Context, db *gorm.DB, apps map[string]*model.ApplicationEntity) error {
	for _, item := range retiredMenus {
		app, ok := apps[item.appCode]
		if !ok || app == nil {
			continue
		}
		menu := &model.MenuEntity{}
		err := db.Where("seed_key = ?", item.menuSeedKey).First(menu).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = db.Where("seed_key = '' AND app_id = ? AND code = ?", app.ID, item.menuSeedKey).First(menu).Error
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("prune retired menu %s query fail: %w", item.menuSeedKey, err)
		}
		if err := db.WithContext(ctx).Where("menu_id = ?", menu.ID).Delete(&model.RoleMenuEntity{}).Error; err != nil {
			return fmt.Errorf("prune retired menu %s role_menu delete fail: %w", item.menuSeedKey, err)
		}
		if err := db.WithContext(ctx).Unscoped().Where("id = ?", menu.ID).Delete(&model.MenuEntity{}).Error; err != nil {
			return fmt.Errorf("prune retired menu %s delete fail: %w", item.menuSeedKey, err)
		}
	}
	return nil
}
