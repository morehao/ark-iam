package application

import (
	"context"
	"errors"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

// 领域哨兵错误：本包不依赖 pkg/code（领域层不绑错误码），由应用层映射为功能级错误码。
var (
	// ErrAppNotSubscribed 目标应用不存在、已停用，或本租户未订阅（含订阅已停用）——
	// 即"订阅是准入的上界"被违反，调用方应映射为"不可准入"的功能级错误。
	ErrAppNotSubscribed = errors.New("application: 目标应用未订阅、已停用或不存在")
	// ErrLastAdminForbidden 本次撤销会使租户失去最后一个内置管理员（防自锁）。
	// 是否成立由应用层回调判定（本包不认识"内置管理员"这一控制台概念）。
	ErrLastAdminForbidden = errors.New("application: 撤销后将使租户失去最后一个内置管理员")
)

// ---------- 判定（唯一表达式） ----------

// appEnabled 应用自身启用。
func appEnabled(app *model.ApplicationEntity) bool {
	return app != nil && app.Status == model.AppStatusEnable
}

// subscriptionEnabled 租户订阅启用。
func subscriptionEnabled(sub *model.TenantApplicationEntity) bool {
	return sub != nil && sub.Status == model.TenantApplicationStatusEnable
}

// grantEnabled 用户准入启用。
func grantEnabled(grant *model.UserApplicationEntity) bool {
	return grant != nil && grant.Status == model.UserApplicationStatusEnable
}

// Usable 准入判定表达式（**唯一实现**）：
//
//	可用 ⟺ 应用启用 ∧ 租户订阅启用 ∧ 用户准入启用
//
// 三者取交集是刻意的：订阅是准入的上界（取消订阅立即失效但不删准入，重新订阅即恢复），
// 用户准入是必要条件（不因持有角色而放行——无角色应用的准入正是靠本行表达）。
// 租户 active 与成员 active 由既有链路各自承担，不在此重复。
func Usable(app *model.ApplicationEntity, subscription *model.TenantApplicationEntity, grant *model.UserApplicationEntity) bool {
	return appEnabled(app) && subscriptionEnabled(subscription) && grantEnabled(grant)
}

// Access 单个应用对某成员的完整准入视图（管理端列表用，判定统一走 Usable）。
type Access struct {
	App          *model.ApplicationEntity
	Subscription *model.TenantApplicationEntity
	Grant        *model.UserApplicationEntity
}

// Usable 该行是否可用于登录该应用。
func (a Access) Usable() bool { return Usable(a.App, a.Subscription, a.Grant) }

// CheckUsable 点查版判定（令牌签发与登录门禁用）。
//
// 缺参（租户/成员/应用任一为空）一律返回 (false, nil)：不可判定即不放行（fail-closed）。
// 短路顺序按"最可能拒绝且最省一次查询"排列：准入 → 应用 → 订阅。
func CheckUsable(ctx context.Context, tenantID, userID, appID string) (bool, error) {
	if tenantID == "" || userID == "" || appID == "" {
		return false, nil
	}
	scoped := dbclient.ExplicitTenantContext(ctx, tenantID)

	grant, err := dao.NewUserApplicationDao().GetByCond(scoped, &dao.UserApplicationCond{
		TenantID: tenantID, UserID: userID, AppID: appID,
	})
	if err != nil {
		glog.Errorf(ctx, "[application.CheckUsable] dao userApplication GetByCond fail, err:%v, tenantID:%s, userID:%s, appID:%s", err, tenantID, userID, appID)
		return false, err
	}
	if !grantEnabled(grant) {
		return false, nil
	}

	app, err := dao.NewApplicationDao().GetByID(scoped, appID)
	if err != nil {
		glog.Errorf(ctx, "[application.CheckUsable] dao application GetByID fail, err:%v, appID:%s", err, appID)
		return false, err
	}
	if !appEnabled(app) {
		return false, nil
	}

	sub, err := dao.NewTenantApplicationDao().GetByCond(scoped, &dao.TenantApplicationCond{
		TenantID: tenantID, AppID: appID,
	})
	if err != nil {
		glog.Errorf(ctx, "[application.CheckUsable] dao tenantApplication GetByCond fail, err:%v, tenantID:%s, appID:%s", err, tenantID, appID)
		return false, err
	}
	return Usable(app, sub, grant), nil
}

// ListAccess 返回该成员的全部准入视图（管理端一次成表，避免前端逐应用查询）。
//
// 取「本租户全部订阅行」与「该成员全部准入行」的并集：已撤销订阅但仍有准入行的应用
// 也要如实呈现（否则管理端看不到"这条准入已被订阅失效覆盖"的真相）。
func ListAccess(ctx context.Context, tenantID, userID string) ([]Access, error) {
	if tenantID == "" || userID == "" {
		return nil, nil
	}
	scoped := dbclient.ExplicitTenantContext(ctx, tenantID)

	subs, err := dao.NewTenantApplicationDao().GetListByCond(scoped, &dao.TenantApplicationCond{TenantID: tenantID})
	if err != nil {
		glog.Errorf(ctx, "[application.ListAccess] dao tenantApplication GetListByCond fail, err:%v, tenantID:%s", err, tenantID)
		return nil, err
	}
	grants, err := dao.NewUserApplicationDao().GetListByCond(scoped, &dao.UserApplicationCond{TenantID: tenantID, UserID: userID})
	if err != nil {
		glog.Errorf(ctx, "[application.ListAccess] dao userApplication GetListByCond fail, err:%v, tenantID:%s, userID:%s", err, tenantID, userID)
		return nil, err
	}

	appIDSet := make(map[string]struct{}, len(subs)+len(grants))
	for i := range subs {
		appIDSet[subs[i].AppID] = struct{}{}
	}
	for i := range grants {
		appIDSet[grants[i].AppID] = struct{}{}
	}
	if len(appIDSet) == 0 {
		return nil, nil
	}
	appIDs := make([]string, 0, len(appIDSet))
	for appID := range appIDSet {
		if appID != "" {
			appIDs = append(appIDs, appID)
		}
	}

	apps, err := dao.NewApplicationDao().GetListByCond(scoped, &dao.ApplicationCond{IDs: appIDs})
	if err != nil {
		glog.Errorf(ctx, "[application.ListAccess] dao application GetListByCond fail, err:%v", err)
		return nil, err
	}
	appMap := make(map[string]*model.ApplicationEntity, len(apps))
	for i := range apps {
		appMap[apps[i].ID] = &apps[i]
	}
	subMap := make(map[string]*model.TenantApplicationEntity, len(subs))
	for i := range subs {
		subMap[subs[i].AppID] = &subs[i]
	}
	grantMap := make(map[string]*model.UserApplicationEntity, len(grants))
	for i := range grants {
		grantMap[grants[i].AppID] = &grants[i]
	}

	result := make([]Access, 0, len(appIDs))
	for _, appID := range appIDs {
		app := appMap[appID]
		if app == nil {
			continue // 应用行缺失（历史脏数据）：不呈现，避免管理端出现无法操作的空行
		}
		result = append(result, Access{App: app, Subscription: subMap[appID], Grant: grantMap[appID]})
	}
	return result, nil
}

// ---------- 写入（不变式 I1 / I2） ----------

// GrantParams EnsureGrant 入参。
type GrantParams struct {
	TenantID string
	UserID   string
	AppID    string
	Source   model.UserApplicationSource
	Operator string
}

// EnsureGrant 在**调用方事务内**确保 (租户, 成员, 应用) 存在 enable 准入（不变式 I1：授权⇒准入）。
//
// 幂等：已 enable 且来源相同时不写库；已存在但为 disable（曾被撤销）或来源变化时
// 翻回 enable 并刷新 source/updatedBy——保留原行便于审计"曾被撤销又恢复"。
// app_id 为空串（未归属应用组）不产生准入，直接返回 nil。
func EnsureGrant(ctx context.Context, tx *gorm.DB, p GrantParams) error {
	if p.TenantID == "" || p.UserID == "" || p.AppID == "" {
		return nil
	}
	scoped := dbclient.ExplicitTenantContext(ctx, p.TenantID)
	userApplicationDao := dao.NewUserApplicationDao().WithTx(tx)

	existing, err := userApplicationDao.GetByCond(scoped, &dao.UserApplicationCond{
		TenantID: p.TenantID, UserID: p.UserID, AppID: p.AppID,
	})
	if err != nil {
		glog.Errorf(ctx, "[application.EnsureGrant] dao userApplication GetByCond fail, err:%v, tenantID:%s, userID:%s, appID:%s", err, p.TenantID, p.UserID, p.AppID)
		return err
	}

	if existing == nil || existing.ID == "" {
		return userApplicationDao.Insert(scoped, &model.UserApplicationEntity{
			TenantID:  p.TenantID,
			UserID:    p.UserID,
			AppID:     p.AppID,
			Status:    model.UserApplicationStatusEnable,
			Source:    p.Source,
			CreatedBy: p.Operator,
			UpdatedBy: p.Operator,
		})
	}
	if existing.Status == model.UserApplicationStatusEnable && existing.Source == p.Source {
		return nil
	}
	return dao.UpdateFields(scoped, userApplicationDao, existing.ID, &model.UserApplicationEntity{
		Status:    model.UserApplicationStatusEnable,
		Source:    p.Source,
		UpdatedBy: p.Operator,
	}, "status", "source", "updated_by")
}

// ReplaceParams Replace 入参。
type ReplaceParams struct {
	TenantID string
	UserID   string
	// AppIDs 是替换后的准入**全集**（PUT 集合语义）；集合外的既有准入一律撤销。
	AppIDs   []string
	Operator string
	// CheckLastAdmin 由应用层注入的防自锁判定（不变式 I5）：
	// Replace 先算出撤销后**保留**的角色集合 keptRoleIDs，再交由应用层判断
	// "这组角色是否仍含内置管理员"。返回 true 表示整体拒绝（ErrLastAdminForbidden）。
	// 传 nil 表示不做该保护（仅测试或无控制台概念的场景）。
	CheckLastAdmin func(ctx context.Context, keptRoleIDs []string) (bool, error)
}

// Replace 全量替换某成员的准入集合（管理端唯一写入口），并落实不变式 I2：
// **被撤销准入的应用，同步级联撤销该成员在该应用下的角色绑定**（单向级联，不反向撤准入）。
//
// 必须在调用方事务内执行（tx 由调用方传入）；任一步失败由调用方回滚，不留部分写入。
// 执行顺序刻意如此：先做全部只读预检（应用可准入校验 + 防自锁），再写库——预检失败时零写入。
func Replace(ctx context.Context, tx *gorm.DB, p ReplaceParams) error {
	if p.TenantID == "" || p.UserID == "" {
		return nil
	}
	scoped := dbclient.ExplicitTenantContext(ctx, p.TenantID)
	userApplicationDao := dao.NewUserApplicationDao().WithTx(tx)

	// 1) 目标集合去重
	nextSet := make(map[string]struct{}, len(p.AppIDs))
	nextAppIDs := make([]string, 0, len(p.AppIDs))
	for _, appID := range p.AppIDs {
		if appID == "" {
			continue
		}
		if _, ok := nextSet[appID]; ok {
			continue
		}
		nextSet[appID] = struct{}{}
		nextAppIDs = append(nextAppIDs, appID)
	}

	// 2) 预检 A：目标集合中的每个应用都必须「存在 + 启用 + 本租户订阅启用」
	if len(nextAppIDs) > 0 {
		apps, err := dao.NewApplicationDao().WithTx(tx).GetListByCond(scoped, &dao.ApplicationCond{IDs: nextAppIDs})
		if err != nil {
			glog.Errorf(ctx, "[application.Replace] dao application GetListByCond fail, err:%v", err)
			return err
		}
		appMap := make(map[string]*model.ApplicationEntity, len(apps))
		for i := range apps {
			appMap[apps[i].ID] = &apps[i]
		}
		subs, err := dao.NewTenantApplicationDao().WithTx(tx).GetListByCond(scoped, &dao.TenantApplicationCond{TenantID: p.TenantID, AppIDs: nextAppIDs})
		if err != nil {
			glog.Errorf(ctx, "[application.Replace] dao tenantApplication GetListByCond fail, err:%v", err)
			return err
		}
		subMap := make(map[string]*model.TenantApplicationEntity, len(subs))
		for i := range subs {
			subMap[subs[i].AppID] = &subs[i]
		}
		for _, appID := range nextAppIDs {
			if !appEnabled(appMap[appID]) || !subscriptionEnabled(subMap[appID]) {
				return ErrAppNotSubscribed
			}
		}
	}

	// 3) 既有准入 → 需撤销的应用集合（仅 enable 行参与撤销；disable 行无需再处理）
	existingGrants, err := userApplicationDao.GetListByCond(scoped, &dao.UserApplicationCond{TenantID: p.TenantID, UserID: p.UserID})
	if err != nil {
		glog.Errorf(ctx, "[application.Replace] dao userApplication GetListByCond fail, err:%v", err)
		return err
	}
	revokeAppSet := make(map[string]struct{})
	for i := range existingGrants {
		grant := existingGrants[i]
		if grant.Status != model.UserApplicationStatusEnable {
			continue
		}
		if _, keep := nextSet[grant.AppID]; keep {
			continue
		}
		revokeAppSet[grant.AppID] = struct{}{}
	}

	// 4) 角色绑定现状：算出「撤销后保留的角色」与「随撤销被删的角色」
	userRoles, err := dao.NewUserRoleDao().WithTx(tx).GetListByCond(scoped, &dao.UserRoleCond{TenantID: p.TenantID, UserID: p.UserID})
	if err != nil {
		glog.Errorf(ctx, "[application.Replace] dao userRole GetListByCond fail, err:%v", err)
		return err
	}
	roleAppMap, err := roleAppMapOf(ctx, scoped, tx, p.TenantID, userRoles)
	if err != nil {
		return err
	}
	keptRoleIDs := make([]string, 0, len(userRoles))
	revokedUserRoleIDs := make([]string, 0, len(userRoles))
	for i := range userRoles {
		if _, revoke := revokeAppSet[roleAppMap[userRoles[i].RoleID]]; revoke {
			revokedUserRoleIDs = append(revokedUserRoleIDs, userRoles[i].ID)
			continue
		}
		keptRoleIDs = append(keptRoleIDs, userRoles[i].RoleID)
	}

	// 5) 预检 B：防自锁（I5）。必须在任何写入之前判定，命中即整体拒绝。
	if len(revokeAppSet) > 0 && p.CheckLastAdmin != nil {
		forbidden, err := p.CheckLastAdmin(ctx, keptRoleIDs)
		if err != nil {
			glog.Errorf(ctx, "[application.Replace] CheckLastAdmin fail, err:%v, tenantID:%s, userID:%s", err, p.TenantID, p.UserID)
			return err
		}
		if forbidden {
			return ErrLastAdminForbidden
		}
	}

	// 6) 写准入：集合内 upsert enable（source=manual），集合外置 disable
	for _, appID := range nextAppIDs {
		if err := EnsureGrant(ctx, tx, GrantParams{
			TenantID: p.TenantID, UserID: p.UserID, AppID: appID,
			Source: model.UserApplicationSourceManual, Operator: p.Operator,
		}); err != nil {
			return err
		}
	}
	for i := range existingGrants {
		grant := existingGrants[i]
		if _, revoke := revokeAppSet[grant.AppID]; !revoke {
			continue
		}
		if err := dao.UpdateFields(scoped, userApplicationDao, grant.ID, &model.UserApplicationEntity{
			Status:    model.UserApplicationStatusDisable,
			UpdatedBy: p.Operator,
		}, "status", "updated_by"); err != nil {
			glog.Errorf(ctx, "[application.Replace] update userApplication status fail, err:%v, id:%s", err, grant.ID)
			return err
		}
	}

	// 7) 级联撤角色（I2）：只删被撤销应用下的绑定，其他应用与角色行本身不动
	userRoleDao := dao.NewUserRoleDao().WithTx(tx)
	for _, userRoleID := range revokedUserRoleIDs {
		if err := userRoleDao.Delete(scoped, userRoleID, p.Operator); err != nil {
			glog.Errorf(ctx, "[application.Replace] cascade delete userRole fail, err:%v, id:%s", err, userRoleID)
			return err
		}
	}
	return nil
}

// roleAppMapOf 收集一批 user_role 关联所指向角色的归属应用（角色行缺失时忽略，不入映射）。
// scoped 是已声明租户作用域的 ctx，logCtx 仅用于记日志（避免日志里丢掉原始上下文）。
func roleAppMapOf(logCtx context.Context, scoped context.Context, tx *gorm.DB, tenantID string, userRoles []model.UserRoleEntity) (map[string]string, error) {
	result := make(map[string]string, len(userRoles))
	if len(userRoles) == 0 {
		return result, nil
	}
	roleIDs := make([]string, 0, len(userRoles))
	for i := range userRoles {
		roleIDs = append(roleIDs, userRoles[i].RoleID)
	}
	roles, err := dao.NewRoleDao().WithTx(tx).GetListByCond(scoped, &dao.RoleCond{TenantID: tenantID, IDs: roleIDs})
	if err != nil {
		glog.Errorf(logCtx, "[application.roleAppMapOf] dao role GetListByCond fail, err:%v", err)
		return nil, err
	}
	for i := range roles {
		result[roles[i].ID] = roles[i].AppID
	}
	return result, nil
}
