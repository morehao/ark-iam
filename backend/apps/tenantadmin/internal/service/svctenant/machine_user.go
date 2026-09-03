package svctenant

import (
	"context"
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
// 服务账号不可登录、不入组织树、无自然人；仅可被授权角色并作为 API Key 归属主体。
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
// opErr 仅在系统错误(角色查询失败等)时兜底返回。
func (svc *machineUserSvc) checkSystemAdmin(ctx *gin.Context, opErr int) error {
	ok, err := HasSystemAdminCapability(ctx)
	if err != nil {
		glog.Errorf(ctx, "[svcmachine.checkSystemAdmin] resolve admin level fail, err:%v", err)
		return code.GetError(opErr)
	}
	if !ok {
		return code.GetError(code.UserSystemAdminRequiredError)
	}
	return nil
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
	return &dtotenant.MachineUserPageListResp{List: respList, Total: total}, nil
}

func (svc *machineUserSvc) Create(ctx *gin.Context, req *dtotenant.MachineUserCreateReq) (*dtotenant.MachineUserCreateResp, error) {
	if err := svc.checkSystemAdmin(ctx, code.MachineUserCreateError); err != nil {
		return nil, err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	entity := &model.UserEntity{
		TenantID:    tenantID,
		PersonID:    "",
		UserType:    model.UserTypeMachine,
		Name:        req.Name,
		Description: req.Description,
		Profile:     json.RawMessage(`{}`),
		CustomData:  json.RawMessage(`{}`),
		CreatedBy:   gincontext.GetUserIDString(ctx),
		UpdatedBy:   gincontext.GetUserIDString(ctx),
	}
	if err := dao.NewUserDao().Insert(context.Background(), entity); err != nil {
		glog.Errorf(ctx, "[svcmachine.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.MachineUserCreateError)
	}
	return &dtotenant.MachineUserCreateResp{MachineUserID: entity.ID}, nil
}

func (svc *machineUserSvc) Update(ctx *gin.Context, req *dtotenant.MachineUserUpdateReq) error {
	if err := svc.checkSystemAdmin(ctx, code.MachineUserUpdateError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	entity, err := svc.loadMachineUser(ctx, tenantID, req.MachineUserID)
	if err != nil {
		return err
	}
	entity.Name = req.Name
	entity.Description = req.Description
	entity.UpdatedBy = gincontext.GetUserIDString(ctx)
	if err := dao.NewUserDao().UpdateMap(ctx, entity.ID, map[string]any{
		"name":        entity.Name,
		"description": entity.Description,
		"updated_by":  entity.UpdatedBy,
	}); err != nil {
		glog.Errorf(ctx, "[svcmachine.Update] dao UpdateMap fail, err:%v, id:%s", err, entity.ID)
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
		oldRoles, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: req.MachineUserID})
		if err != nil {
			return err
		}
		for _, r := range oldRoles {
			if err := dao.NewUserRoleDao().WithTx(tx).Delete(ctx, r.ID, operator); err != nil {
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
		Roles: roles,
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

// UpdateRoles 全量替换服务账号的角色（PUT 集合语义）。
// 服务账号可被授予普通/自定义角色，禁止授予系统管理角色（admin_level=super），杜绝管理能力落入机器主体。
func (svc *machineUserSvc) UpdateRoles(ctx *gin.Context, req *dtotenant.MachineUserRolesUpdateReq) error {
	if err := svc.checkSystemAdmin(ctx, code.MachineUserRoleReplaceError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	if _, err := svc.loadMachineUser(ctx, tenantID, req.MachineUserID); err != nil {
		return err
	}

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
			if model.SysAdminLevel(r.AdminLevel).HasSystemAdmin() {
				return code.GetError(code.UserSuperRoleAssignForbidden)
			}
		}
	}

	operator := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		oldList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: req.MachineUserID})
		if err != nil {
			return err
		}
		for _, r := range oldList {
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
