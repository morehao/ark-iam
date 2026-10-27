package svctenant

import (
	"context"
	"errors"

	"github.com/morehao/ark-iam/pkg/code"
	iamapplication "github.com/morehao/ark-iam/pkg/core/application"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

// ApplicationAccessSvc 用户应用准入（准入层的租户端唯一写入口）。
//
// 准入只回答"该成员能否使用某应用"：与角色（进入应用后能做什么）解耦，
// 因此**无角色的接入应用同样在这里授权**。判定式与写入不变式（I1/I2/I5）的唯一实现
// 在 pkg/core/application；本层只做鉴权、越权校验与 DTO 装配，不复制任何判定逻辑。
type ApplicationAccessSvc interface {
	ListTenantApplications(ctx context.Context) (*dtotenant.TenantApplicationsResp, error)
	ListUserApplications(ctx context.Context, req *dtotenant.UserApplicationsListReq) (*dtotenant.UserApplicationsListResp, error)
	UpdateUserApplications(ctx context.Context, req *dtotenant.UserApplicationsUpdateReq) error
	ListApplicationUsers(ctx context.Context, req *dtotenant.ApplicationUsersListReq) (*dtotenant.ApplicationUsersListResp, error)
}

type applicationAccessSvc struct{}

var _ ApplicationAccessSvc = (*applicationAccessSvc)(nil)

func NewApplicationAccessSvc() ApplicationAccessSvc {
	return &applicationAccessSvc{}
}

// ListTenantApplications 本租户订阅且启用的应用（应用准入编辑器与角色归属的应用选项全集）。
// 过滤口径复用菜单域的 loadSubscribedApps：订阅启用 ∧ 应用启用。
func (svc *applicationAccessSvc) ListTenantApplications(ctx context.Context) (*dtotenant.TenantApplicationsResp, error) {
	appList, err := loadSubscribedApps(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]dtotenant.TenantApplicationItem, 0, len(appList))
	for _, app := range appList {
		list = append(list, dtotenant.TenantApplicationItem{
			AppID: app.ID,
			Code:  app.Code,
			Name:  app.Name,
		})
	}
	return &dtotenant.TenantApplicationsResp{List: list}, nil
}

// ListUserApplications 该成员的准入应用清单（含未准入的订阅应用，供 UI 呈现全集与选中态）。
// 每行回传 roleCount，由前端拼"撤销后同时失去该应用下 N 个角色"的二次确认文案
// ——不做服务端 dry-run，避免出现第二份判定实现。
func (svc *applicationAccessSvc) ListUserApplications(ctx context.Context, req *dtotenant.UserApplicationsListReq) (*dtotenant.UserApplicationsListResp, error) {
	if err := requireSystemAdmin(ctx, req.Operator, code.UserApplicationGetListError); err != nil {
		return nil, err
	}
	tenantID := dbclient.TenantIDFromContext(ctx)
	if _, err := svc.loadMember(ctx, tenantID, req.UserID, code.UserApplicationGetListError); err != nil {
		return nil, err
	}

	rows, err := iamapplication.ListAccess(ctx, tenantID, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcappaccess.ListUserApplications] ListAccess fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserApplicationGetListError)
	}
	roleCountByApp, err := roleCountByAppOf(ctx, tenantID, req.UserID)
	if err != nil {
		return nil, err
	}

	list := make([]dtotenant.UserApplicationItem, 0, len(rows))
	for _, row := range rows {
		item := dtotenant.UserApplicationItem{
			AppID:      row.App.ID,
			Code:       row.App.Code,
			Name:       row.App.Name,
			AppStatus:  row.App.Status,
			Subscribed: row.Subscription != nil && row.Subscription.Status == model.TenantApplicationStatusEnable,
			Granted:    row.Grant != nil && row.Grant.Status == model.UserApplicationStatusEnable,
			RoleCount:  roleCountByApp[row.App.ID],
		}
		if row.Grant != nil {
			item.Status = row.Grant.Status
			item.Source = row.Grant.Source
			item.UpdatedAt = row.Grant.UpdatedAt.Unix()
		} else if row.Subscription != nil {
			item.UpdatedAt = row.Subscription.UpdatedAt.Unix()
		}
		list = append(list, item)
	}
	return &dtotenant.UserApplicationsListResp{List: list}, nil
}

// UpdateUserApplications 全量替换该成员的准入集合（PUT 集合语义）。
// 集合内 enable、集合外 disable，并对集合外应用级联撤销其角色绑定（I2）；
// 命中防自锁（I5，会移除最后一个内置管理员的控制台准入）时整体拒绝。
func (svc *applicationAccessSvc) UpdateUserApplications(ctx context.Context, req *dtotenant.UserApplicationsUpdateReq) error {
	if err := requireSystemAdmin(ctx, req.Operator, code.UserApplicationReplaceError); err != nil {
		return err
	}
	tenantID := dbclient.TenantIDFromContext(ctx)
	if _, err := svc.loadMember(ctx, tenantID, req.UserID, code.UserApplicationReplaceError); err != nil {
		return err
	}

	appIDs := []string{}
	if req.AppIDs != nil {
		appIDs = *req.AppIDs
	}
	operator := req.Operator.UserID
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		return iamapplication.Replace(ctx, tx, iamapplication.ReplaceParams{
			TenantID: tenantID,
			UserID:   req.UserID,
			AppIDs:   appIDs,
			Operator: operator,
			// 防自锁（I5）：复用租户端「最后一个内置管理员」保护——Replace 传入撤销后
			// 保留的角色集合，由本层判断该集合是否仍含内置管理员角色。
			CheckLastAdmin: func(_ context.Context, keptRoleIDs []string) (bool, error) {
				return (&userSvc{}).hasOtherSystemAdminHolder(ctx, tenantID, req.UserID, keptRoleIDs)
			},
		})
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcappaccess.UpdateUserApplications] replace fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		// 业务边界与系统错误分离：哨兵错误映射为功能级错误码，其余按系统错误兜底
		if errors.Is(txErr, iamapplication.ErrAppNotSubscribed) {
			return code.GetError(code.UserApplicationNotSubscribedError)
		}
		if errors.Is(txErr, iamapplication.ErrLastAdminForbidden) {
			return code.GetError(code.UserApplicationLastAdminForbiddenError)
		}
		return code.GetError(code.UserApplicationReplaceError)
	}
	return nil
}

// ListApplicationUsers 应用视角：被准入的成员列表（谁能进这个应用）。
func (svc *applicationAccessSvc) ListApplicationUsers(ctx context.Context, req *dtotenant.ApplicationUsersListReq) (*dtotenant.ApplicationUsersListResp, error) {
	if err := requireSystemAdmin(ctx, req.Operator, code.UserApplicationGetListError); err != nil {
		return nil, err
	}
	tenantID := dbclient.TenantIDFromContext(ctx)
	appEntity, err := dao.NewApplicationDao().GetByID(ctx, req.AppID)
	if err != nil {
		glog.Errorf(ctx, "[svcappaccess.ListApplicationUsers] dao application GetByID fail, err:%v, appID:%s", err, req.AppID)
		return nil, code.GetError(code.UserApplicationGetListError)
	}
	if appEntity == nil || appEntity.ID == "" {
		return nil, code.GetError(code.ApplicationNotExistError)
	}

	grants, err := dao.NewUserApplicationDao().GetListByCond(ctx, &dao.UserApplicationCond{
		TenantID: tenantID, AppID: req.AppID, Status: model.UserApplicationStatusEnable,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcappaccess.ListApplicationUsers] dao userApplication GetListByCond fail, err:%v, appID:%s", err, req.AppID)
		return nil, code.GetError(code.UserApplicationGetListError)
	}
	if len(grants) == 0 {
		return &dtotenant.ApplicationUsersListResp{List: []dtotenant.ApplicationUserItem{}}, nil
	}

	userIDs := make([]string, 0, len(grants))
	grantSource := make(map[string]model.UserApplicationSource, len(grants))
	grantUpdatedAt := make(map[string]int64, len(grants))
	for i := range grants {
		userIDs = append(userIDs, grants[i].UserID)
		grantSource[grants[i].UserID] = grants[i].Source
		grantUpdatedAt[grants[i].UserID] = grants[i].UpdatedAt.Unix()
	}
	users, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{TenantID: tenantID, IDs: userIDs})
	if err != nil {
		glog.Errorf(ctx, "[svcappaccess.ListApplicationUsers] dao user GetListByCond fail, err:%v, appID:%s", err, req.AppID)
		return nil, code.GetError(code.UserApplicationGetListError)
	}
	briefByUser, err := svc.personBriefOf(ctx, users)
	if err != nil {
		return nil, err
	}
	roleCountByUser, err := roleCountByUserOf(ctx, tenantID, req.AppID, userIDs)
	if err != nil {
		return nil, err
	}

	list := make([]dtotenant.ApplicationUserItem, 0, len(users))
	for i := range users {
		user := users[i]
		brief := briefByUser[user.ID]
		list = append(list, dtotenant.ApplicationUserItem{
			UserID:       user.ID,
			UserType:     user.UserType,
			Name:         user.Name,
			Username:     brief.username,
			PrimaryEmail: brief.primaryEmail,
			Status:       user.Status,
			Source:       grantSource[user.ID],
			RoleCount:    roleCountByUser[user.ID],
			UpdatedAt:    grantUpdatedAt[user.ID],
		})
	}
	return &dtotenant.ApplicationUsersListResp{List: list, Total: int64(len(list))}, nil
}

// loadMember 加载当前租户内的成员行（不存在/跨租户返回"用户不存在"）。
// 服务账号走 /machine-users 入口管理，本入口与角色接口同口径拒绝。
func (svc *applicationAccessSvc) loadMember(ctx context.Context, tenantID, userID string, opErr int) (*model.UserEntity, error) {
	userEntity, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil {
		glog.Errorf(ctx, "[svcappaccess.loadMember] dao user GetByID fail, err:%v, userID:%s", err, userID)
		return nil, code.GetError(opErr)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return nil, code.GetError(code.UserNotExistError)
	}
	if userEntity.IsMachine() {
		return nil, code.GetError(code.UserMemberOperationOnlyError)
	}
	return userEntity, nil
}

// personBrief 成员的自然人摘要（用户名/邮箱只在 person 上，服务账号恒为空）。
type personBrief struct {
	username     string
	primaryEmail string
}

// personBriefOf 批量回填成员的自然人摘要（一次查询，避免列表页 N+1）。
func (svc *applicationAccessSvc) personBriefOf(ctx context.Context, users []model.UserEntity) (map[string]personBrief, error) {
	result := make(map[string]personBrief, len(users))
	personIDs := make([]string, 0, len(users))
	for i := range users {
		if users[i].PersonID != "" {
			personIDs = append(personIDs, users[i].PersonID)
		}
	}
	personMap := map[string]personBrief{}
	if len(personIDs) > 0 {
		persons, err := dao.NewPersonDao().GetListByCond(ctx, &dao.PersonCond{IDs: personIDs})
		if err != nil {
			glog.Errorf(ctx, "[svcappaccess.personBriefOf] dao person GetListByCond fail, err:%v", err)
			return nil, code.GetError(code.UserApplicationGetListError)
		}
		for i := range persons {
			brief := personBrief{}
			if persons[i].Username != nil {
				brief.username = *persons[i].Username
			}
			if persons[i].PrimaryEmail != nil {
				brief.primaryEmail = *persons[i].PrimaryEmail
			}
			personMap[persons[i].ID] = brief
		}
	}
	for i := range users {
		result[users[i].ID] = personMap[users[i].PersonID]
	}
	return result, nil
}

// roleCountByAppOf 汇总某成员在各应用下的角色数（应用 → 计数）。两次查询，无 N+1。
func roleCountByAppOf(ctx context.Context, tenantID, userID string) (map[string]int64, error) {
	userRoles, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: userID})
	if err != nil {
		glog.Errorf(ctx, "[svcappaccess.roleCountByAppOf] dao userRole GetListByCond fail, err:%v, userID:%s", err, userID)
		return nil, code.GetError(code.UserApplicationGetListError)
	}
	if len(userRoles) == 0 {
		return map[string]int64{}, nil
	}
	roleIDs := make([]string, 0, len(userRoles))
	for i := range userRoles {
		roleIDs = append(roleIDs, userRoles[i].RoleID)
	}
	roles, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: roleIDs})
	if err != nil {
		glog.Errorf(ctx, "[svcappaccess.roleCountByAppOf] dao role GetListByCond fail, err:%v", err)
		return nil, code.GetError(code.UserApplicationGetListError)
	}
	appOfRole := make(map[string]string, len(roles))
	for i := range roles {
		appOfRole[roles[i].ID] = roles[i].AppID
	}
	counts := make(map[string]int64, len(roles))
	for _, userRole := range userRoles {
		appID := appOfRole[userRole.RoleID]
		if appID == "" {
			continue
		}
		counts[appID]++
	}
	return counts, nil
}

// roleCountByUserOf 汇总某应用下各成员的角色数（成员 → 计数）。
// 一次 GROUP BY user_id 聚合（同 svctenant.loadUserRoleCountMap 的既有口径），行数随成员数封顶，无 N+1。
func roleCountByUserOf(ctx context.Context, tenantID, appID string, userIDs []string) (map[string]int64, error) {
	counts := make(map[string]int64, len(userIDs))
	if len(userIDs) == 0 {
		return counts, nil
	}
	roles, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, AppID: appID})
	if err != nil {
		glog.Errorf(ctx, "[svcappaccess.roleCountByUserOf] dao role GetListByCond fail, err:%v, appID:%s", err, appID)
		return nil, code.GetError(code.UserApplicationGetListError)
	}
	if len(roles) == 0 {
		return counts, nil
	}
	roleIDs := make([]string, 0, len(roles))
	for i := range roles {
		roleIDs = append(roleIDs, roles[i].ID)
	}
	type countRow struct {
		UserID string
		Cnt    int64
	}
	var rows []countRow
	if err := dbclient.IamDB(ctx).Model(&model.UserRoleEntity{}).
		Where("tenant_id = ? AND user_id IN ? AND role_id IN ?", tenantID, userIDs, roleIDs).
		Select("user_id, count(*) as cnt").Group("user_id").Scan(&rows).Error; err != nil {
		glog.Errorf(ctx, "[svcappaccess.roleCountByUserOf] count userRole fail, err:%v, appID:%s", err, appID)
		return nil, code.GetError(code.UserApplicationGetListError)
	}
	for _, row := range rows {
		counts[row.UserID] = row.Cnt
	}
	return counts, nil
}
