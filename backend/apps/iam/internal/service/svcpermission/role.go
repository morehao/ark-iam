package svcpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objpermission"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type RoleSvc interface {
	Create(ctx *gin.Context, req *dtopermission.RoleCreateReq) (*dtopermission.RoleCreateResp, error)
	Delete(ctx *gin.Context, req *dtopermission.RoleDeleteReq) error
	Update(ctx *gin.Context, req *dtopermission.RoleUpdateReq) error
	Detail(ctx *gin.Context, req *dtopermission.RoleDetailReq) (*dtopermission.RoleDetailResp, error)
	PageList(ctx *gin.Context, req *dtopermission.RolePageListReq) (*dtopermission.RolePageListResp, error)
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
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleDeleteError)
	}
	if roleEntity == nil || roleEntity.ID == 0 {
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
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleUpdateError)
	}
	if roleEntity == nil || roleEntity.ID == 0 {
		return code.GetError(code.RoleNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":    req.TenantID,
		"name":         req.Name,
		"code":         req.Code,
		"description":  req.Description,
		"type":         req.Type,
		"is_default":   req.IsDefault,
		"updated_by":   userID,
	}
	if err := dao.NewRoleDao().UpdateMap(ctx, req.RoleID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateRole] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.RoleUpdateError)
	}
	return nil
}

func (svc *roleSvc) Detail(ctx *gin.Context, req *dtopermission.RoleDetailReq) (*dtopermission.RoleDetailResp, error) {
	roleEntity, err := dao.NewRoleDao().GetByID(ctx, req.RoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DetailRole] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.RoleGetDetailError)
	}
	if roleEntity == nil || roleEntity.ID == 0 {
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
	cond := &dao.RoleCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		Name:     req.Name,
		Code:     req.Code,
		Type:     req.Type,
	}
	roleEntityList, total, err := dao.NewRoleDao().GetPageListByCond(ctx, cond)
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