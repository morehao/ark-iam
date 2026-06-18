package svcpermission

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objresource"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type scopeScopeRepository interface {
	GetByID(ctx context.Context, id uint) (*model.ScopeEntity, error)
	GetResourceByID(ctx context.Context, id uint) (*model.ResourceEntity, error)
	GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.ScopeEntityList, int64, error)
	Insert(ctx context.Context, entity *model.ScopeEntity) error
}

var newScopeScopeRepo = func() scopeScopeRepository {
	return &scopeScopeDAO{}
}

type scopeScopeDAO struct{}

func (d *scopeScopeDAO) GetByID(ctx context.Context, id uint) (*model.ScopeEntity, error) {
	return dao.NewScopeDao().GetByID(ctx, id)
}

func (d *scopeScopeDAO) GetResourceByID(ctx context.Context, id uint) (*model.ResourceEntity, error) {
	return dao.NewResourceDao().GetByID(ctx, id)
}

func (d *scopeScopeDAO) GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.ScopeEntityList, int64, error) {
	return dao.NewScopeDao().GetPageListByCond(ctx, cond)
}

func (d *scopeScopeDAO) Insert(ctx context.Context, entity *model.ScopeEntity) error {
	return dao.NewScopeDao().Insert(ctx, entity)
}

func scopeVisibleToTenant(entity *model.ScopeEntity, tenantID uint) bool {
	return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}

type ScopeSvc interface {
	Create(ctx *gin.Context, req *dtopermission.ScopeCreateReq) (*dtopermission.ScopeCreateResp, error)
	Delete(ctx *gin.Context, req *dtopermission.ScopeDeleteReq) error
	Update(ctx *gin.Context, req *dtopermission.ScopeUpdateReq) error
	Detail(ctx *gin.Context, req *dtopermission.ScopeDetailReq) (*dtopermission.ScopeDetailResp, error)
	PageList(ctx *gin.Context, req *dtopermission.ScopePageListReq) (*dtopermission.ScopePageListResp, error)
}

type scopeSvc struct{}

var _ ScopeSvc = (*scopeSvc)(nil)

func NewScopeSvc() ScopeSvc {
	return &scopeSvc{}
}

func (svc *scopeSvc) Create(ctx *gin.Context, req *dtopermission.ScopeCreateReq) (*dtopermission.ScopeCreateResp, error) {
	repo := newScopeScopeRepo()
	resourceEntity, err := repo.GetResourceByID(ctx, req.ResourceID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ResourceGetDetailError)
	}
	if !resourceVisibleToTenant(resourceEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.ResourceNotExistError)
	}

	insertEntity := &model.ScopeEntity{
		TenantID:    req.TenantID,
		ResourceID:  req.ResourceID,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   gincontext.GetUserID(ctx),
	}

	if err := repo.Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcpermission.CreateScope] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ScopeCreateError)
	}
	return &dtopermission.ScopeCreateResp{
		ScopeID: insertEntity.ID,
	}, nil
}

func (svc *scopeSvc) Delete(ctx *gin.Context, req *dtopermission.ScopeDeleteReq) error {
	scopeEntity, err := newScopeScopeRepo().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeDeleteError)
	}
	if !scopeVisibleToTenant(scopeEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.ScopeNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewScopeDao().Delete(ctx, req.ScopeID, userID); err != nil {
		glog.Errorf(ctx, "[svcpermission.DeleteScope] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeDeleteError)
	}
	return nil
}

func (svc *scopeSvc) Update(ctx *gin.Context, req *dtopermission.ScopeUpdateReq) error {
	scopeEntity, err := newScopeScopeRepo().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeUpdateError)
	}
	if !scopeVisibleToTenant(scopeEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.ScopeNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":    req.TenantID,
		"resource_id":  req.ResourceID,
		"name":         req.Name,
		"description":  req.Description,
		"updated_by":   userID,
	}
	if err := dao.NewScopeDao().UpdateMap(ctx, req.ScopeID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcpermission.UpdateScope] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ScopeUpdateError)
	}
	return nil
}

func (svc *scopeSvc) Detail(ctx *gin.Context, req *dtopermission.ScopeDetailReq) (*dtopermission.ScopeDetailResp, error) {
	scopeEntity, err := newScopeScopeRepo().GetByID(ctx, req.ScopeID)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.DetailScope] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ScopeGetDetailError)
	}
	if !scopeVisibleToTenant(scopeEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.ScopeNotExistError)
	}

	resp := &dtopermission.ScopeDetailResp{
		ScopeID: scopeEntity.ID,
		ScopeBaseInfo: objresource.ScopeBaseInfo{
			TenantID:    scopeEntity.TenantID,
			ResourceID:  scopeEntity.ResourceID,
			Name:        scopeEntity.Name,
			Description: scopeEntity.Description,
		},
	}
	return resp, nil
}

func (svc *scopeSvc) PageList(ctx *gin.Context, req *dtopermission.ScopePageListReq) (*dtopermission.ScopePageListResp, error) {
	scopeRepo := newScopeScopeRepo()
	cond := &dao.ScopeCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:   gincontext.GetTenantID(ctx),
		ResourceID: req.ResourceID,
		Name:       req.Name,
	}
	scopeEntityList, total, err := scopeRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcpermission.PageListScope] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ScopeGetPageListError)
	}

	list := make([]dtopermission.ScopePageListItem, 0, len(scopeEntityList))
	for _, v := range scopeEntityList {
		list = append(list, dtopermission.ScopePageListItem{
			ScopeID: v.ID,
			ScopeBaseInfo: objresource.ScopeBaseInfo{
				TenantID:    v.TenantID,
				ResourceID:  v.ResourceID,
				Name:        v.Name,
				Description: v.Description,
			},
		})
	}
	return &dtopermission.ScopePageListResp{
		List:  list,
		Total: total,
	}, nil
}
