package seed

import (
	"context"
	"errors"
	"fmt"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"gorm.io/gorm"
)

// retiredMenu 已下线菜单登记项。
//
// seedMenus 只做幂等 upsert（存在则回填字段、不存在则创建），不会移除历史版本写入的菜单行，
// 因此每下线一个菜单都必须在此登记：启动时清理其菜单行与 role_menu 授权绑定，
// 保证存量库与当前种子定义一致（否则菜单管理页会残留幽灵菜单）。
type retiredMenu struct {
	appCode  string // 菜单所属应用编码
	menuCode string // 菜单编码（app 内唯一）
}

// retiredMenus 已下线菜单清单（只增不减，每项仅在存量库首次启动时命中一次）。
//
// 登记判据：把 base 版本的 seedMenus 与当前定义做差集，凡「曾在历史版本种子里出现过、
// 现在不再定义」的菜单都必须在此登记——seedMenus 不会下线菜单，漏登记即存量库死链。
var retiredMenus = []retiredMenu{
	// 平台端「API密钥监督」页已下线：跨租户只读监督价值有限，
	// 密钥的创建/吊销/删除与排查统一收敛到租户自服务控制台「API密钥」模块。
	{appCode: appCodeAdmin, menuCode: "api-key"},
	// 平台端「身份中心」目录及其「用户管理」「角色管理」子菜单已下线：
	// 用户与角色按租户归属，读写入口全部收敛到租户自服务控制台，
	// 平台端不再提供跨租户用户目录与角色只读视图（避免绕过租户授权边界）。
	// 注意父目录与子菜单必须一并登记：只登记父目录会留下指向已删除页面的孤立子菜单，
	// 只登记子菜单会留下空目录。
	{appCode: appCodeAdmin, menuCode: "grp-identity"},
	{appCode: appCodeAdmin, menuCode: "user"},
	{appCode: appCodeAdmin, menuCode: "role"},
}

// pruneRetiredMenus 幂等清理已下线菜单：先解除 role_menu 授权绑定，再删除菜单行（软删除）。
// 已清理过的项在后续启动中查询不到，直接跳过。
func pruneRetiredMenus(ctx context.Context, db *gorm.DB, apps map[string]*model.ApplicationEntity) error {
	for _, item := range retiredMenus {
		app, ok := apps[item.appCode]
		if !ok || app == nil {
			continue
		}
		menu := &model.MenuEntity{}
		err := db.Where("app_id = ? AND code = ?", app.ID, item.menuCode).First(menu).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("prune retired menu %s query fail: %w", item.menuCode, err)
		}
		if err := db.WithContext(ctx).Where("menu_id = ?", menu.ID).Delete(&model.RoleMenuEntity{}).Error; err != nil {
			return fmt.Errorf("prune retired menu %s role_menu delete fail: %w", item.menuCode, err)
		}
		if err := db.WithContext(ctx).Where("id = ?", menu.ID).Delete(&model.MenuEntity{}).Error; err != nil {
			return fmt.Errorf("prune retired menu %s delete fail: %w", item.menuCode, err)
		}
	}
	return nil
}
