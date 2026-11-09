package seed

import "github.com/morehao/ark-iam/pkg/model"

// builtinMenuDefs 内置菜单定义：**唯一事实源**。
//
// seedMenus（首次引导写入菜单行）与 BuiltinMenus（`make print-builtin-menus` 输出给运维的
// 升级清单）共用本清单，避免"代码里加了菜单、清单里没有"的漂移。
//
// 注意它只携带 10 个字段——redirect/hidden/externalLink/keepAlive/status 由建表列默认值兜底；
// 需要完整 15 字段时用 BuiltinMenus()。
var builtinMenuDefs = []seedMenu{
	// 平台管理后台：目录分组（type=directory，无页面）+ 页面叶子（type=menu，指向真实前端页面）。
	// 一级菜单按「对象域」划分（对象名词 + 中心/叶子），不使用「X 与 Y」并列命名：
	// 租户中心（租户及其资源）/ 应用中心（应用及其接入凭证）/
	// 平台管理（平台自身治理：菜单字典与审计日志）。
	// 用户与角色不再有平台端入口：两者按租户归属，读写与成员管理全部收敛到租户管理后台。
	{appCode: appCodeAdmin, name: "工作台", code: "dashboard", path: "/dashboard", icon: "dashboard", sort: 1, component: "/dashboard/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityMember},
	{appCode: appCodeAdmin, name: "租户中心", code: "grp-tenant", icon: "apartment", sort: 2, menuType: model.MenuTypeDirectory, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeAdmin, parentCode: "grp-tenant", name: "租户管理", code: "tenant", path: "/tenant", icon: "global", sort: 1, component: "/tenant/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeAdmin, parentCode: "grp-tenant", name: "租户应用", code: "tenant-application", path: "/tenant-application", icon: "shopping", sort: 2, component: "/tenantApplication/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeAdmin, parentCode: "grp-tenant", name: "自定义域名", code: "domain", path: "/domain", icon: "global", sort: 3, component: "/domain/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeAdmin, name: "应用中心", code: "grp-app", icon: "app", sort: 3, menuType: model.MenuTypeDirectory, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeAdmin, parentCode: "grp-app", name: "应用管理", code: "application", path: "/application", icon: "app", sort: 1, component: "/application/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeAdmin, parentCode: "grp-app", name: "OAuth客户端", code: "oauth-client", path: "/oauth-client", icon: "key", sort: 2, component: "/oauthClient/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeAdmin, name: "平台管理", code: "grp-platform", icon: "setting", sort: 4, menuType: model.MenuTypeDirectory, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeAdmin, parentCode: "grp-platform", name: "菜单管理", code: "menu", path: "/menu", icon: "menu", sort: 1, component: "/menu/index", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
	// 租户管理后台一级菜单：控制台定位为「租户管理层专用」（部门/用户/角色/密钥均属管理操作，
	// 全部 visibility=admin 硬隔离；普通成员不面向该控制台，仅内置管理员角色可见与授权）。
	// 用户/角色/密钥编码加 tenant- 前缀，避免与平台菜单 code 撞名。
	{appCode: appCodeTenantAdmin, name: "部门管理", code: "department", path: "/department", icon: "apartment", sort: 1, component: "pages/department", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeTenantAdmin, name: "用户管理", code: "tenant-user", path: "/user", icon: "user", sort: 2, component: "pages/user", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeTenantAdmin, name: "角色管理", code: "tenant-role", path: "/role", icon: "role", sort: 3, component: "pages/role", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
	{appCode: appCodeTenantAdmin, name: "API密钥", code: "tenant-api-key", path: "/api-key", icon: "key", sort: 4, component: "pages/apiKey", menuType: model.MenuTypeMenu, visibility: model.MenuVisibilityAdmin},
}

// BuiltinMenu 内置菜单的**完整字段清单**，与 objpermission.MenuBaseInfo 的 15 个字段一一对应。
//
// 为什么需要它：菜单行归运维后（L1 只播种一次，之后新增/调整由「菜单管理」页完成），
// 版本升级带来的新菜单必须由运维手工录入；而 seedMenu 只携带 10 个字段，
// 照抄 seedMenu 会漏掉 redirect/hidden/externalLink/keepAlive/status。
// 本类型把这 5 个字段以"列默认值"的形式补全，使清单可被逐字照抄。
type BuiltinMenu struct {
	// 以下 10 个字段来自 builtinMenuDefs（种子写入）
	AppCode    string
	ParentCode string
	Name       string
	Code       string
	Path       string
	Icon       string
	Sort       int
	Component  string
	Type       model.MenuType
	Visibility model.MenuVisibility

	// 以下 5 个字段不由种子携带，取建表列默认值。
	// 它们必须与 pkg/model/menu.go 的 gorm default 标签、以及菜单表实际落库值保持一致，
	// 由 TestBuiltinMenus_MatchSeededRows 直接比对真实库行来钉住。
	Redirect     string
	Hidden       model.MenuHiddenFlag
	ExternalLink model.MenuExternalLinkFlag
	KeepAlive    model.MenuKeepAliveFlag
	Status       model.MenuStatus
}

// BuiltinMenus 返回内置菜单的完整 15 字段清单（含列默认值），顺序与 builtinMenuDefs 一致。
//
// 只读：返回新切片，调用方修改不会影响种子定义（本包对外的写通道只有 Bootstrap）。
func BuiltinMenus() []BuiltinMenu {
	out := make([]BuiltinMenu, 0, len(builtinMenuDefs))
	for _, def := range builtinMenuDefs {
		out = append(out, BuiltinMenu{
			AppCode:      def.appCode,
			ParentCode:   def.parentCode,
			Name:         def.name,
			Code:         def.code,
			Path:         def.path,
			Icon:         def.icon,
			Sort:         def.sort,
			Component:    def.component,
			Type:         menuTypeOf(def),
			Visibility:   menuVisibilityOf(def),
			Redirect:     "",
			Hidden:       model.MenuHiddenFlagDisable,
			ExternalLink: model.MenuExternalLinkFlagDisable,
			KeepAlive:    model.MenuKeepAliveFlagDisable,
			Status:       model.MenuStatusEnable,
		})
	}
	return out
}

// menuVisibilityOf 返回菜单种子定义的 visibility；未显式指定时缺省为 public。
func menuVisibilityOf(def seedMenu) model.MenuVisibility {
	if def.visibility == "" {
		return model.MenuVisibilityPublic
	}
	return def.visibility
}
