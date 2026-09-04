package svctenant

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

// MachineUserSvc 服务账号（租户内机器主体，user_type=machine）领域服务。
// 服务账号不可登录、无自然人；从属于部门（primary 主部门必填 + secondary 参与可多条），
// 可被授权角色并作为 API Key 归属主体；系统管理角色（admin_level=super）禁止授予服务账号。
type MachineUserSvc interface {
	PageList(ctx *gin.Context, req *dtotenant.MachineUserPageListReq) (*dtotenant.MachineUserPageListResp, error)
	Create(ctx *gin.Context, req *dtotenant.MachineUserCreateReq) (*dtotenant.MachineUserCreateResp, error)
	Update(ctx *gin.Context, req *dtotenant.MachineUserUpdateReq) error
	UpdateStatus(ctx *gin.Context, req *dtotenant.MachineUserStatusReq) error
	Delete(ctx *gin.Context, req *dtotenant.MachineUserDeleteReq) error
	Detail(ctx *gin.Context, req *dtotenant.MachineUserDetailReq) (*dtotenant.MachineUserDetailResp, error)
	ListRoles(ctx *gin.Context, req *dtotenant.MachineUserRolesListReq) (*dtotenant.UserRolesListResp, error)
	UpdateRoles(ctx *gin.Context, req *dtotenant.MachineUserRolesUpdateReq) error
}

type machineUserSvc struct{}

var _ MachineUserSvc = (*machineUserSvc)(nil)

func NewMachineUserSvc() MachineUserSvc {
	return &machineUserSvc{}
}

// checkSystemAdmin 校验当前操作者具备系统管理能力（admin_level=super），否则返回能力不足错误。
// opErr 仅在系统错误(角色查询失败等)时兜底返回。统一走共享门槛 svctenant.requireSystemAdmin。
func (svc *machineUserSvc) checkSystemAdmin(ctx *gin.Context, opErr int) error {
	return requireSystemAdmin(ctx, opErr)
}

// loadMachineUser 加载指定租户内的服务账号（不存在/跨租户/非服务账号均按不存在处理）。
func (svc *machineUserSvc) loadMachineUser(ctx *gin.Context, tenantID, machineUserID string) (*model.UserEntity, error) {
	entity, err := dao.NewUserDao().GetByID(ctx, machineUserID)
	if err != nil {
		glog.Errorf(ctx, "[svcmachine.loadMachineUser] dao GetByID fail, err:%v, id:%s", err, machineUserID)
		return nil, code.GetError(code.MachineUserGetDetailError)
	}
	if entity == nil || entity.TenantID != tenantID || !entity.IsMachine() {
		return nil, code.GetError(code.UserNotExistError)
	}
	return entity, nil
}

// tenantOrgSet 加载本租户全部组织节点 ID 集合，用于归属校验（组织必须属于当前租户）。
func tenantOrgSet(ctx *gin.Context, tenantID string) (map[string]bool, error) {
	orgList, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{TenantID: tenantID})
	if err != nil {
		glog.Errorf(ctx, "[svcmachine.tenantOrgSet] dao query org fail, err:%v", err)
		return nil, err
	}
	set := make(map[string]bool, len(orgList))
	for _, o := range orgList {
		set[o.ID] = true
	}
	return set, nil
}

// checkOrgIDs 校验组织 ID 集合全部属于本租户（set 为租户组织集合）。
func checkOrgIDs(set map[string]bool, orgIDs []string) bool {
	for _, orgID := range orgIDs {
		if !set[orgID] {
			return false
		}
	}
	return true
}

// fillPrimaryOrg 批量回填主部门信息（列表页 N+1 优化：一次查 primary 关系 + 一次组织名映射）。
func fillPrimaryOrg(ctx *gin.Context, tenantID string, respList []dtotenant.MachineUserPageListItem) error {
	userIDs := make([]string, 0, len(respList))
	for _, item := range respList {
		userIDs = append(userIDs, item.MachineUserID)
	}
	if len(userIDs) == 0 {
		return nil
	}
	relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:     tenantID,
		UserIDs:      userIDs,
		RelationType: model.OrgUserRelationPrimary,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcmachine.fillPrimaryOrg] dao query primary relations fail, err:%v", err)
		return err
	}
	primaryOrgOfUser := make(map[string]string, len(relationList))
	for _, r := range relationList {
		primaryOrgOfUser[r.UserID] = r.OrganizationID
	}
	orgNameMap := make(map[string]string)
	if len(primaryOrgOfUser) > 0 {
		orgList, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{TenantID: tenantID})
		if err != nil {
			glog.Errorf(ctx, "[svcmachine.fillPrimaryOrg] dao query org fail, err:%v", err)
			return err
		}
		for _, o := range orgList {
			orgNameMap[o.ID] = o.Name
		}
	}
	for i := range respList {
		orgID := primaryOrgOfUser[respList[i].MachineUserID]
		respList[i].PrimaryOrgID = orgID
		respList[i].PrimaryOrgName = orgNameMap[orgID]
	}
	return nil
}

func (svc *machineUserSvc) PageList(ctx *gin.Context, req *dtotenant.MachineUserPageListReq) (*dtotenant.MachineUserPageListResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	cond := &dao.UserCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:    tenantID,
		UserType:    model.UserTypeMachine,
		Keyword:     req.Name,
		IsSuspended: req.IsSuspended,
	}
	list, total, err := dao.NewUserDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcmachine.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MachineUserGetPageListError)
	}
	respList := make([]dtotenant.MachineUserPageListItem, 0, len(list))
	for _, v := range list {
		respList = append(respList, dtotenant.MachineUserPageListItem{
			MachineUserID: v.ID,
			TenantID:      v.TenantID,
			Name:          v.Name,
			Description:   v.Description,
			IsSuspended:   v.IsSuspended,
			CreatedAt:     v.CreatedAt.Unix(),
		})
	}
	if err := fillPrimaryOrg(ctx, tenantID, respList); err != nil {
		return nil, code.GetError(code.MachineUserGetPageListError)
	}
	return &dtotenant.MachineUserPageListResp{List: respList, Total: total}, nil
}

// Create 创建服务账号：与真实用户一致必须从属一个主部门（primary），可选参与部门（secondary）。
// 组织关系与账号同事务建立。
func (svc *machineUserSvc) Create(ctx *gin.Context, req *dtotenant.MachineUserCreateReq) (*dtotenant.MachineUserCreateResp, error) {
	if err := svc.checkSystemAdmin(ctx, code.MachineUserCreateError); err != nil {
		return nil, err
	}
	tenantID := gincontext.GetTenantIDString(ctx)

	// 主部门必填且至多一个（业务约束，防绕过 DTO 校验）
	if len(req.OrganizationIDs) != 1 {
		return nil, code.GetError(code.MachineUserOrgRequiredError)
	}
	orgSet, err := tenantOrgSet(ctx, tenantID)
	if err != nil {
		return nil, code.GetError(code.MachineUserCreateError)
	}
	if !checkOrgIDs(orgSet, req.OrganizationIDs) || !checkOrgIDs(orgSet, req.SecondaryOrgIDs) {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	operatorID := gincontext.GetUserIDString(ctx)
	entity := &model.UserEntity{
		TenantID:    tenantID,
		PersonID:    "",
		UserType:    model.UserTypeMachine,
		Name:        req.Name,
		Description: req.Description,
		Profile:     json.RawMessage(`{}`),
		CustomData:  json.RawMessage(`{}`),
		CreatedBy:   operatorID,
	}
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dao.NewUserDao().WithTx(tx).Insert(ctx, entity); err != nil {
			return err
		}
		// 建立主部门关系（primary，至多 1 条）
		for _, orgID := range req.OrganizationIDs {
			relation := &model.OrganizationUserEntity{
				TenantID:       tenantID,
				OrganizationID: orgID,
				UserID:         entity.ID,
				RelationType:   model.OrgUserRelationPrimary,
				CreatedBy:      operatorID,
			}
			if err := dao.NewOrganizationUserDao().WithTx(tx).Insert(ctx, relation); err != nil {
				return err
			}
		}
		// 建立参与部门关系（secondary，可多条）
		for _, orgID := range req.SecondaryOrgIDs {
			relation := &model.OrganizationUserEntity{
				TenantID:       tenantID,
				OrganizationID: orgID,
				UserID:         entity.ID,
				RelationType:   model.OrgUserRelationSecondary,
				CreatedBy:      operatorID,
			}
			if err := dao.NewOrganizationUserDao().WithTx(tx).Insert(ctx, relation); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcmachine.Create] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.MachineUserCreateError)
	}
	return &dtotenant.MachineUserCreateResp{MachineUserID: entity.ID}, nil
}

// Update 更新服务账号基础信息与部门归属（primary 不可清空、secondary 全量替换，均 nil=不变）。
func (svc *machineUserSvc) Update(ctx *gin.Context, req *dtotenant.MachineUserUpdateReq) error {
	if err := svc.checkSystemAdmin(ctx, code.MachineUserUpdateError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	entity, err := svc.loadMachineUser(ctx, tenantID, req.MachineUserID)
	if err != nil {
		return err
	}

	// 主部门不可清空：显式传 "" 拒绝。
	if req.PrimaryOrgID != nil && *req.PrimaryOrgID == "" {
		return code.GetError(code.MachineUserOrgRequiredError)
	}
	// 校验传入的组织均属本租户。
	if req.PrimaryOrgID != nil || req.SecondaryOrgIDs != nil {
		orgSet, err := tenantOrgSet(ctx, tenantID)
		if err != nil {
			return code.GetError(code.MachineUserUpdateError)
		}
		if req.PrimaryOrgID != nil && !orgSet[*req.PrimaryOrgID] {
			return code.GetError(code.OrganizationNotExistError)
		}
		if !checkOrgIDs(orgSet, derefSlice(req.SecondaryOrgIDs)) {
			return code.GetError(code.OrganizationNotExistError)
		}
	}

	operatorID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		updateMap := map[string]any{
			"name":        req.Name,
			"description": req.Description,
			"updated_by":  operatorID,
		}
		if err := dao.NewUserDao().UpdateMap(ctx, entity.ID, updateMap); err != nil {
			return err
		}
		if req.PrimaryOrgID != nil {
			// 替换主部门：删旧 primary（至多 1 行）后建新。
			if err := replaceOrgRelationList(ctx, tx, tenantID, entity.ID, []string{*req.PrimaryOrgID}, model.OrgUserRelationPrimary, operatorID); err != nil {
				return err
			}
		}
		if req.SecondaryOrgIDs != nil {
			// 参与部门全量替换（[]=清空）。
			if err := replaceOrgRelationList(ctx, tx, tenantID, entity.ID, *req.SecondaryOrgIDs, model.OrgUserRelationSecondary, operatorID); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcmachine.Update] transaction fail, err:%v, id:%s", txErr, entity.ID)
		return code.GetError(code.MachineUserUpdateError)
	}
	return nil
}

func (svc *machineUserSvc) UpdateStatus(ctx *gin.Context, req *dtotenant.MachineUserStatusReq) error {
	if err := svc.checkSystemAdmin(ctx, code.MachineUserStatusUpdateError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	if _, err := svc.loadMachineUser(ctx, tenantID, req.MachineUserID); err != nil {
		return err
	}
	if err := dao.NewUserDao().UpdateMap(ctx, req.MachineUserID, map[string]any{
		"is_suspended": req.IsSuspended,
		"updated_by":   gincontext.GetUserIDString(ctx),
	}); err != nil {
		glog.Errorf(ctx, "[svcmachine.UpdateStatus] dao UpdateMap fail, err:%v, id:%s", err, req.MachineUserID)
		return code.GetError(code.MachineUserStatusUpdateError)
	}
	return nil
}

func (svc *machineUserSvc) Delete(ctx *gin.Context, req *dtotenant.MachineUserDeleteReq) error {
	if err := svc.checkSystemAdmin(ctx, code.MachineUserDeleteError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	if _, err := svc.loadMachineUser(ctx, tenantID, req.MachineUserID); err != nil {
		return err
	}
	// 孤儿凭证防护：服务账号下仍有 API Key 时禁止删除
	keyList, _, err := dao.NewApiKeyDao().GetPageListByCond(ctx, &dao.ApiKeyCond{
		TenantID:    tenantID,
		OwnerUserID: req.MachineUserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcmachine.Delete] dao query api keys fail, err:%v, id:%s", err, req.MachineUserID)
		return code.GetError(code.MachineUserDeleteError)
	}
	if len(keyList) > 0 {
		return code.GetError(code.MachineUserDeleteHasKeysError)
	}

	operator := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 级联清理角色关联
		oldRoles, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: req.MachineUserID})
		if err != nil {
			return err
		}
		for _, r := range oldRoles {
			if err := dao.NewUserRoleDao().WithTx(tx).Delete(ctx, r.ID, operator); err != nil {
				return err
			}
		}
		// 级联清理部门归属关系
		oldOrgs, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{TenantID: tenantID, UserID: req.MachineUserID})
		if err != nil {
			return err
		}
		for _, r := range oldOrgs {
			if err := dao.NewOrganizationUserDao().WithTx(tx).Delete(ctx, r.ID, operator); err != nil {
				return err
			}
		}
		if err := dao.NewUserDao().WithTx(tx).Delete(ctx, req.MachineUserID, operator); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcmachine.Delete] transaction fail, err:%v, id:%s", txErr, req.MachineUserID)
		return code.GetError(code.MachineUserDeleteError)
	}
	return nil
}

func (svc *machineUserSvc) Detail(ctx *gin.Context, req *dtotenant.MachineUserDetailReq) (*dtotenant.MachineUserDetailResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	entity, err := svc.loadMachineUser(ctx, tenantID, req.MachineUserID)
	if err != nil {
		return nil, err
	}
	organizations, err := loadUserOrganizations(ctx, tenantID, req.MachineUserID)
	if err != nil {
		glog.Errorf(ctx, "[svcmachine.Detail] load organizations fail, err:%v, id:%s", err, req.MachineUserID)
		return nil, code.GetError(code.MachineUserGetDetailError)
	}
	roles, err := (&userSvc{}).listRoles(ctx, tenantID, req.MachineUserID)
	if err != nil {
		return nil, err
	}
	return &dtotenant.MachineUserDetailResp{
		MachineUserPageListItem: dtotenant.MachineUserPageListItem{
			MachineUserID: entity.ID,
			TenantID:      entity.TenantID,
			Name:          entity.Name,
			Description:   entity.Description,
			IsSuspended:   entity.IsSuspended,
			CreatedAt:     entity.CreatedAt.Unix(),
		},
		Organizations: organizations,
		Roles:         roles,
	}, nil
}

func (svc *machineUserSvc) ListRoles(ctx *gin.Context, req *dtotenant.MachineUserRolesListReq) (*dtotenant.UserRolesListResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	if _, err := svc.loadMachineUser(ctx, tenantID, req.MachineUserID); err != nil {
		return nil, err
	}
	roles, err := (&userSvc{}).listRoles(ctx, tenantID, req.MachineUserID)
	if err != nil {
		return nil, err
	}
	return &dtotenant.UserRolesListResp{List: roles}, nil
}

// UpdateRoles 按应用全量替换服务账号的角色（PUT 集合语义）。
// 与真实用户一致：仅替换目标应用(role.app_id == req.AppID)下的角色关联，其它应用不受影响；
// req.AppID 空串=系统/未归属应用组。服务账号可被授予普通/自定义角色，禁止授予系统管理角色
// （admin_level=super），杜绝管理能力落入机器主体。
func (svc *machineUserSvc) UpdateRoles(ctx *gin.Context, req *dtotenant.MachineUserRolesUpdateReq) error {
	if err := svc.checkSystemAdmin(ctx, code.MachineUserRoleReplaceError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	if _, err := svc.loadMachineUser(ctx, tenantID, req.MachineUserID); err != nil {
		return err
	}

	// 旧关联（事务外读取一次）
	oldList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: req.MachineUserID})
	if err != nil {
		glog.Errorf(ctx, "[svcmachine.UpdateRoles] dao GetListByCond old user_role fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.MachineUserRoleReplaceError)
	}
	oldRoleAppMap, err := (&userSvc{}).roleAppMapOf(ctx, tenantID, oldList)
	if err != nil {
		return err
	}

	// 校验新角色：均属于本租户、归属目标应用，且非系统管理角色
	if len(req.RoleIDs) > 0 {
		roleList, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: req.RoleIDs})
		if err != nil {
			glog.Errorf(ctx, "[svcmachine.UpdateRoles] dao role GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.MachineUserRoleReplaceError)
		}
		if len(roleList) != len(req.RoleIDs) {
			return code.GetError(code.RoleNotExistError)
		}
		for _, r := range roleList {
			if r.AppID != req.AppID {
				return code.GetError(code.RoleNotExistError)
			}
			if model.SysAdminLevel(r.AdminLevel).HasSystemAdmin() {
				return code.GetError(code.UserSuperRoleAssignForbidden)
			}
		}
	}

	operator := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 仅删除目标应用下的旧关联
		for _, r := range oldList {
			if oldRoleAppMap[r.RoleID] != req.AppID {
				continue
			}
			if err := dao.NewUserRoleDao().WithTx(tx).Delete(ctx, r.ID, operator); err != nil {
				return err
			}
		}
		for _, roleID := range req.RoleIDs {
			entity := &model.UserRoleEntity{
				TenantID:  tenantID,
				UserID:    req.MachineUserID,
				RoleID:    roleID,
				CreatedBy: operator,
			}
			if err := dao.NewUserRoleDao().WithTx(tx).Insert(ctx, entity); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcmachine.UpdateRoles] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.MachineUserRoleReplaceError)
	}
	return nil
}
