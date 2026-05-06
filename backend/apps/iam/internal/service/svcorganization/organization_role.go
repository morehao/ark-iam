package svcorganization

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoorganization"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type OrganizationRoleSvc interface {
	Create(ctx *gin.Context, req *dtoorganization.OrganizationRoleCreateReq) (*dtoorganization.OrganizationRoleCreateResp, error)
	Delete(ctx *gin.Context, req *dtoorganization.OrganizationRoleDeleteReq) error
	Update(ctx *gin.Context, req *dtoorganization.OrganizationRoleUpdateReq) error
	Detail(ctx *gin.Context, req *dtoorganization.OrganizationRoleDetailReq) (*dtoorganization.OrganizationRoleDetailResp, error)
	PageList(ctx *gin.Context, req *dtoorganization.OrganizationRolePageListReq) (*dtoorganization.OrganizationRolePageListResp, error)
}

type organizationRoleSvc struct {
}

var _ OrganizationRoleSvc = (*organizationRoleSvc)(nil)

func NewOrganizationRoleSvc() OrganizationRoleSvc {
	return &organizationRoleSvc{}
}

func (svc *organizationRoleSvc) Create(ctx *gin.Context, req *dtoorganization.OrganizationRoleCreateReq) (*dtoorganization.OrganizationRoleCreateResp, error) {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetDetailError)
	}
	if orgEntity == nil || orgEntity.ID == 0 {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	insertEntity := &model.OrganizationRoleEntity{
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		CreatedBy:      gincontext.GetUserID(ctx),
	}

	if err := dao.NewOrganizationRoleDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleCreateError)
	}
	return &dtoorganization.OrganizationRoleCreateResp{
		OrganizationRoleID: insertEntity.ID,
	}, nil
}

func (svc *organizationRoleSvc) Delete(ctx *gin.Context, req *dtoorganization.OrganizationRoleDeleteReq) error {
	orgRoleEntity, err := dao.NewOrganizationRoleDao().GetByID(ctx, req.OrganizationRoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleDeleteError)
	}
	if orgRoleEntity == nil || orgRoleEntity.ID == 0 {
		return code.GetError(code.OrganizationRoleNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewOrganizationRoleDao().Delete(ctx, req.OrganizationRoleID, userID); err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleDeleteError)
	}
	return nil
}

func (svc *organizationRoleSvc) Update(ctx *gin.Context, req *dtoorganization.OrganizationRoleUpdateReq) error {
	orgRoleEntity, err := dao.NewOrganizationRoleDao().GetByID(ctx, req.OrganizationRoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleUpdateError)
	}
	if orgRoleEntity == nil || orgRoleEntity.ID == 0 {
		return code.GetError(code.OrganizationRoleNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":        req.TenantID,
		"organization_id":  req.OrganizationID,
		"name":             req.Name,
		"description":      req.Description,
		"type":             req.Type,
		"updated_by":       userID,
	}
	if err := dao.NewOrganizationRoleDao().UpdateMap(ctx, req.OrganizationRoleID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleUpdateError)
	}
	return nil
}

func (svc *organizationRoleSvc) Detail(ctx *gin.Context, req *dtoorganization.OrganizationRoleDetailReq) (*dtoorganization.OrganizationRoleDetailResp, error) {
	orgRoleEntity, err := dao.NewOrganizationRoleDao().GetByID(ctx, req.OrganizationRoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleGetDetailError)
	}
	if orgRoleEntity == nil || orgRoleEntity.ID == 0 {
		return nil, code.GetError(code.OrganizationRoleNotExistError)
	}

	resp := &dtoorganization.OrganizationRoleDetailResp{
		OrganizationRoleID: orgRoleEntity.ID,
		OrganizationRoleBaseInfo: dtoorganization.OrganizationRoleBaseInfo{
			TenantID:       orgRoleEntity.TenantID,
			OrganizationID: orgRoleEntity.OrganizationID,
			Name:           orgRoleEntity.Name,
			Description:    orgRoleEntity.Description,
			Type:           orgRoleEntity.Type,
		},
	}
	return resp, nil
}

func (svc *organizationRoleSvc) PageList(ctx *gin.Context, req *dtoorganization.OrganizationRolePageListReq) (*dtoorganization.OrganizationRolePageListResp, error) {
	cond := &dao.OrganizationRoleCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
	}
	orgRoleEntityList, total, err := dao.NewOrganizationRoleDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleGetPageListError)
	}

	list := make([]dtoorganization.OrganizationRolePageListItem, 0, len(orgRoleEntityList))
	for _, v := range orgRoleEntityList {
		list = append(list, dtoorganization.OrganizationRolePageListItem{
			OrganizationRoleID: v.ID,
			OrganizationRoleBaseInfo: dtoorganization.OrganizationRoleBaseInfo{
				TenantID:       v.TenantID,
				OrganizationID: v.OrganizationID,
				Name:           v.Name,
				Description:    v.Description,
				Type:           v.Type,
			},
		})
	}
	return &dtoorganization.OrganizationRolePageListResp{
		List:  list,
		Total: total,
	}, nil
}