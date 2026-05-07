package svcapplication

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objapplication"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ApplicationSvc interface {
	Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error)
	Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error
	Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error
	Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error)
	PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error)
	ListRoles(ctx *gin.Context, req *dtoapplication.ApplicationRoleListReq) (*dtoapplication.ApplicationRoleListResp, error)
	AssignRoles(ctx *gin.Context, req *dtoapplication.AssignApplicationRolesReq) error
	RemoveRole(ctx *gin.Context, req *dtoapplication.RemoveApplicationRoleReq) error
}

type applicationSvc struct {
}

var _ ApplicationSvc = (*applicationSvc)(nil)

func NewApplicationSvc() ApplicationSvc {
	return &applicationSvc{}
}

func (svc *applicationSvc) Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error) {
	insertEntity := &model.ApplicationEntity{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Secret:      req.Secret,
		Description: req.Description,
		Type:        req.Type,
		IsThirdParty: req.IsThirdParty,
		CreatedBy:   gincontext.GetUserID(ctx),
	}

	if err := dao.NewApplicationDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCreateError)
	}
	return &dtoapplication.ApplicationCreateResp{
		ApplicationID: insertEntity.ID,
	}, nil
}

func (svc *applicationSvc) Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error {
	appEntity, err := dao.NewApplicationDao().GetByID(ctx, req.ApplicationID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	if appEntity == nil || appEntity.ID == 0 {
		return code.GetError(code.ApplicationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewApplicationDao().Delete(ctx, req.ApplicationID, userID); err != nil {
		glog.Errorf(ctx, "[svcapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	return nil
}

func (svc *applicationSvc) Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error {
	appEntity, err := dao.NewApplicationDao().GetByID(ctx, req.ApplicationID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	if appEntity == nil || appEntity.ID == 0 {
		return code.GetError(code.ApplicationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":    req.TenantID,
		"name":         req.Name,
		"description":  req.Description,
		"type":         req.Type,
		"is_third_party": req.IsThirdParty,
		"updated_by":   userID,
	}
	if err := dao.NewApplicationDao().UpdateMap(ctx, req.ApplicationID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	return nil
}

func (svc *applicationSvc) Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error) {
	appEntity, err := dao.NewApplicationDao().GetByID(ctx, req.ApplicationID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetDetailError)
	}
	if appEntity == nil || appEntity.ID == 0 {
		return nil, code.GetError(code.ApplicationNotExistError)
	}

	resp := &dtoapplication.ApplicationDetailResp{
		ApplicationID: appEntity.ID,
		ApplicationBaseInfo: objapplication.ApplicationBaseInfo{
			TenantID:    appEntity.TenantID,
			Name:        appEntity.Name,
			Secret:      appEntity.Secret,
			Description: appEntity.Description,
			Type:        appEntity.Type,
			IsThirdParty: appEntity.IsThirdParty,
		},
	}
	return resp, nil
}

func (svc *applicationSvc) PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error) {
	cond := &dao.ApplicationCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:    req.TenantID,
		Name:       req.Name,
		Type:       req.Type,
	}
	appEntityList, total, err := dao.NewApplicationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetPageListError)
	}

	list := make([]dtoapplication.ApplicationPageListItem, 0, len(appEntityList))
	for _, v := range appEntityList {
		list = append(list, dtoapplication.ApplicationPageListItem{
			ApplicationID: v.ID,
			ApplicationBaseInfo: objapplication.ApplicationBaseInfo{
				TenantID:    v.TenantID,
				Name:        v.Name,
				Secret:      v.Secret,
				Description: v.Description,
				Type:        v.Type,
				IsThirdParty: v.IsThirdParty,
			},
		})
	}
	return &dtoapplication.ApplicationPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *applicationSvc) ListRoles(ctx *gin.Context, req *dtoapplication.ApplicationRoleListReq) (*dtoapplication.ApplicationRoleListResp, error) {
	appRoleDao := dao.NewApplicationRoleDao()
	roleDao := dao.NewRoleDao()

	list, err := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
		ApplicationID: req.ApplicationID,
	})
	if err != nil {
		glog.Errorf(ctx, "[applicationSvc.ListRoles] get roles fail, err:%v", err)
		return nil, code.GetError(code.ApplicationRoleGetListError)
	}

	roleMap := make(map[uint]*model.RoleEntity)
	for _, ar := range list {
		if role, err := roleDao.GetByID(ctx, ar.RoleID); err == nil && role != nil {
			roleMap[role.ID] = role
		}
	}

	roles := make([]dtoapplication.ApplicationRoleResp, 0, len(list))
	for _, ar := range list {
		if role, ok := roleMap[ar.RoleID]; ok {
			roles = append(roles, dtoapplication.ApplicationRoleResp{
				RoleID:        ar.RoleID,
				RoleName:      role.Name,
				RoleCode:      role.Code,
				ApplicationID: ar.ApplicationID,
				CreatedAt:    ar.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	return &dtoapplication.ApplicationRoleListResp{
		Total: int64(len(roles)),
		Roles: roles,
	}, nil
}

func (svc *applicationSvc) AssignRoles(ctx *gin.Context, req *dtoapplication.AssignApplicationRolesReq) error {
	appRoleDao := dao.NewApplicationRoleDao()
	userID := gincontext.GetUserID(ctx)

	for _, roleID := range req.RoleIDs {
		existing, _ := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
			ApplicationID: req.ApplicationID,
			RoleID:       roleID,
		})
		if len(existing) > 0 {
			continue
		}

		entity := &model.ApplicationRoleEntity{
			TenantID:      gincontext.GetTenantID(ctx),
			ApplicationID: req.ApplicationID,
			RoleID:       roleID,
			CreatedBy:    userID,
		}
		if err := appRoleDao.Insert(ctx, entity); err != nil {
			glog.Errorf(ctx, "[applicationSvc.AssignRoles] insert fail, err:%v", err)
			return code.GetError(code.ApplicationRoleCreateError)
		}
	}

	return nil
}

func (svc *applicationSvc) RemoveRole(ctx *gin.Context, req *dtoapplication.RemoveApplicationRoleReq) error {
	appRoleDao := dao.NewApplicationRoleDao()

	list, err := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
		ApplicationID: req.ApplicationID,
		RoleID:       req.RoleID,
	})
	if err != nil || len(list) == 0 {
		return code.GetError(code.ApplicationRoleNotExistError)
	}

	if err := appRoleDao.Delete(ctx, list[0].ID, gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[applicationSvc.RemoveRole] delete fail, err:%v", err)
		return code.GetError(code.ApplicationRoleDeleteError)
	}

	return nil
}