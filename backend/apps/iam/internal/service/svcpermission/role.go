package svcpermission

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objpermission"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type roleScopeRepository interface {
	GetByID(ctx context.Context, id uint) (*model.RoleEntity, error)
	GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.RoleEntityList, int64, error)
}

var newRoleScopeRepo = func() roleScopeRepository {
	return dao.NewRoleDao()
}

func roleVisibleToTenant(entity *model.RoleEntity, tenantID uint) bool {
	return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}

type RoleSvc interface {
	Create(ctx *gin.Context, req *dtopermission.RoleCreateReq) (*dtopermission.RoleCreateResp, error)
	Delete(ctx *gin.Context, req *dtopermission.RoleDeleteReq) error
	Update(ctx *gin.Context, req *dtopermission.RoleUpdateReq) error
	Detail(ctx *gin.Context, req *dtopermission.RoleDetailReq) (*dtopermission.RoleDetailResp, error)
	PageList(ctx *gin.Context, req *dtopermission.RolePageListReq) (*dtopermission.RolePageListResp, error)
	ListUsers(ctx *gin.Context, req *dtouser.RoleUserListReq) (*dtouser.RoleUserListResp, error)
	AssignUsers(ctx *gin.Context, req *dtouser.AssignRoleUsersReq) error
	RemoveUser(ctx *gin.Context, req *dtouser.RemoveRoleUserReq) error
}

type roleSvc struct{}

var _ RoleSvc = (*roleSvc)(nil)

func NewRoleSvc() RoleSvc {
	return &roleSvc{}
}

func (svc *roleSvc) Create(ctx *gin.Context, req *dtopermission.RoleCreateReq) (*dtopermission.RoleCreateResp, error) {
	insertEntity := &model.RoleEntity{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Type:        req.Type,
		IsDefault:   req.IsDefault,
		CreatedBy:   gincontext.GetUserID(ctx),
	}

	if err := dao.NewRoleDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateRole] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleCreateError)
	}
	return &dtopermission.RoleCreateResp{
		RoleID: insertEntity.ID,
	}, nil
}

func (svc *roleSvc) Delete(ctx *gin.Context, req *dtopermission.RoleDeleteReq) error {
	roleEntity, err := newRoleScopeRepo().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleDeleteError)
	}
	if !roleVisibleToTenant(roleEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.RoleNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewRoleDao().Delete(ctx, req.RoleID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRole] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleDeleteError)
	}
	return nil
}

func (svc *roleSvc) Update(ctx *gin.Context, req *dtopermission.RoleUpdateReq) error {
	roleEntity, err := newRoleScopeRepo().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleUpdateError)
	}
	if !roleVisibleToTenant(roleEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.RoleNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":   req.TenantID,
		"name":        req.Name,
		"code":        req.Code,
		"description": req.Description,
		"type":        req.Type,
		"is_default":  req.IsDefault,
		"updated_by":  userID,
	}
	if err := dao.NewRoleDao().UpdateMap(ctx, req.RoleID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateRole] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleUpdateError)
	}
	return nil
}

func (svc *roleSvc) Detail(ctx *gin.Context, req *dtopermission.RoleDetailReq) (*dtopermission.RoleDetailResp, error) {
	roleEntity, err := newRoleScopeRepo().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DetailRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if !roleVisibleToTenant(roleEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.RoleNotExistError)
	}

	resp := &dtopermission.RoleDetailResp{
		RoleID: roleEntity.ID,
		RoleBaseInfo: objpermission.RoleBaseInfo{
			TenantID:    roleEntity.TenantID,
			Name:        roleEntity.Name,
			Code:        roleEntity.Code,
			Description: roleEntity.Description,
			Type:        roleEntity.Type,
			IsDefault:   roleEntity.IsDefault,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: int64(roleEntity.CreatedAt.Unix()),
			UpdatedAt: int64(roleEntity.UpdatedAt.Unix()),
		},
	}
	return resp, nil
}

func (svc *roleSvc) PageList(ctx *gin.Context, req *dtopermission.RolePageListReq) (*dtopermission.RolePageListResp, error) {
	roleRepo := newRoleScopeRepo()
	cond := &dao.RoleCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: gincontext.GetTenantID(ctx),
		Name:     req.Name,
		Code:     req.Code,
		Type:     req.Type,
	}
	roleEntityList, total, err := roleRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.PageListRole] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetPageListError)
	}

	list := make([]dtopermission.RolePageListItem, 0, len(roleEntityList))
	for _, v := range roleEntityList {
		list = append(list, dtopermission.RolePageListItem{
			RoleID: v.ID,
			RoleBaseInfo: objpermission.RoleBaseInfo{
				TenantID:    v.TenantID,
				Name:        v.Name,
				Code:        v.Code,
				Description: v.Description,
				Type:        v.Type,
				IsDefault:   v.IsDefault,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: int64(v.UpdatedAt.Unix()),
			},
		})
	}
	return &dtopermission.RolePageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *roleSvc) ListUsers(ctx *gin.Context, req *dtouser.RoleUserListReq) (*dtouser.RoleUserListResp, error) {
	userRoleDao := dao.NewUserRoleDao()
	userDao := dao.NewUserDao()

	list, err := userRoleDao.GetListByCond(ctx, &dao.UserRoleCond{
		RoleID: uint(req.RoleID),
	})
	if err != nil {
		glog.Errorf(ctx, "[roleSvc.ListUsers] get users fail, err:%v", err)
		return nil, code.GetError(code.RoleUserGetListError)
	}

	userMap := make(map[uint]*model.UserEntity)
	for _, ur := range list {
		if user, err := userDao.GetByID(ctx, ur.UserID); err == nil && user != nil {
			userMap[user.ID] = user
		}
	}

	users := make([]dtouser.RoleUserResp, 0, len(list))
	for _, ur := range list {
		if user, ok := userMap[ur.UserID]; ok {
			users = append(users, dtouser.RoleUserResp{
				UserID:    uint64(ur.UserID),
				Username:  "",
				Name:      user.Name,
				Email:     "",
				RoleID:    uint64(ur.RoleID),
				CreatedAt: ur.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	return &dtouser.RoleUserListResp{
		Total: int64(len(users)),
		Users: users,
	}, nil
}

func (svc *roleSvc) AssignUsers(ctx *gin.Context, req *dtouser.AssignRoleUsersReq) error {
	userRoleDao := dao.NewUserRoleDao()
	userID := gincontext.GetUserID(ctx)

	for _, uid := range req.UserIDs {
		existing, _ := userRoleDao.GetListByCond(ctx, &dao.UserRoleCond{
			RoleID: uint(req.RoleID),
			UserID: uint(uid),
		})
		if len(existing) > 0 {
			continue
		}

		entity := &model.UserRoleEntity{
			TenantID:  gincontext.GetTenantID(ctx),
			UserID:    uint(uid),
			RoleID:    uint(req.RoleID),
			CreatedBy: userID,
		}
		if err := userRoleDao.Insert(ctx, entity); err != nil {
			glog.Errorf(ctx, "[roleSvc.AssignUsers] insert fail, err:%v", err)
			return code.GetError(code.RoleUserCreateError)
		}
	}

	return nil
}

func (svc *roleSvc) RemoveUser(ctx *gin.Context, req *dtouser.RemoveRoleUserReq) error {
	userRoleDao := dao.NewUserRoleDao()

	list, err := userRoleDao.GetListByCond(ctx, &dao.UserRoleCond{
		RoleID: uint(req.RoleID),
		UserID: uint(req.UserID),
	})
	if err != nil || len(list) == 0 {
		return code.GetError(code.RoleUserNotExistError)
	}

	if err := userRoleDao.Delete(ctx, list[0].ID, gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[roleSvc.RemoveUser] delete fail, err:%v", err)
		return code.GetError(code.RoleUserDeleteError)
	}

	return nil
}
