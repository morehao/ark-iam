package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/gctx"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

func organizationRoleVisibleToTenant(entity *model.OrganizationRoleEntity, tenantID string) bool {
	return entity != nil && entity.ID != "" && entity.TenantID == tenantID
}

type OrganizationRoleSvc interface {
	Create(ctx *gin.Context, req *dtotenant.OrganizationRoleCreateReq) (*dtotenant.OrganizationRoleCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.OrganizationRoleDeleteReq) error
	Update(ctx *gin.Context, req *dtotenant.OrganizationRoleUpdateReq) error
	Detail(ctx *gin.Context, req *dtotenant.OrganizationRoleDetailReq) (*dtotenant.OrganizationRoleDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenant.OrganizationRolePageListReq) (*dtotenant.OrganizationRolePageListResp, error)
}

type organizationRoleSvc struct {
}

var _ OrganizationRoleSvc = (*organizationRoleSvc)(nil)

func NewOrganizationRoleSvc() OrganizationRoleSvc {
	return &organizationRoleSvc{}
}

func (svc *organizationRoleSvc) Create(ctx *gin.Context, req *dtotenant.OrganizationRoleCreateReq) (*dtotenant.OrganizationRoleCreateResp, error) {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Create] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetDetailError)
	}
	if !organizationVisibleToTenant(orgEntity, gctx.GetTenantID(ctx)) {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	insertEntity := &model.OrganizationRoleEntity{
		TenantID:       req.TenantID,
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Description:    req.Description,
		Type:           req.Type,
		CreatedBy:      gctx.GetUserID(ctx),
	}
	if insertEntity.TenantID == "" {
		insertEntity.TenantID = gctx.GetTenantID(ctx)
	}

	if err := dao.NewOrganizationRoleDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleCreateError)
	}
	return &dtotenant.OrganizationRoleCreateResp{
		OrganizationRoleID: insertEntity.ID,
	}, nil
}

func (svc *organizationRoleSvc) Delete(ctx *gin.Context, req *dtotenant.OrganizationRoleDeleteReq) error {
	orgRoleEntity, err := dao.NewOrganizationRoleDao().GetByID(ctx, req.OrganizationRoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleDeleteError)
	}
	if !organizationRoleVisibleToTenant(orgRoleEntity, gctx.GetTenantID(ctx)) {
		return code.GetError(code.OrganizationRoleNotExistError)
	}

	userID := gctx.GetUserID(ctx)
	if err := dao.NewOrganizationRoleDao().Delete(ctx, req.OrganizationRoleID, userID); err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleDeleteError)
	}
	return nil
}

func (svc *organizationRoleSvc) Update(ctx *gin.Context, req *dtotenant.OrganizationRoleUpdateReq) error {
	orgRoleEntity, err := dao.NewOrganizationRoleDao().GetByID(ctx, req.OrganizationRoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleUpdateError)
	}
	if !organizationRoleVisibleToTenant(orgRoleEntity, gctx.GetTenantID(ctx)) {
		return code.GetError(code.OrganizationRoleNotExistError)
	}

	userID := gctx.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":       gctx.GetTenantID(ctx),
		"organization_id": req.OrganizationID,
		"name":            req.Name,
		"description":     req.Description,
		"type":            req.Type,
		"updated_by":      userID,
	}
	if err := dao.NewOrganizationRoleDao().UpdateMap(ctx, req.OrganizationRoleID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationRoleUpdateError)
	}
	return nil
}

func (svc *organizationRoleSvc) Detail(ctx *gin.Context, req *dtotenant.OrganizationRoleDetailReq) (*dtotenant.OrganizationRoleDetailResp, error) {
	orgRoleEntity, err := dao.NewOrganizationRoleDao().GetByID(ctx, req.OrganizationRoleID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganizationrole.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationRoleGetDetailError)
	}
	if !organizationRoleVisibleToTenant(orgRoleEntity, gctx.GetTenantID(ctx)) {
		return nil, code.GetError(code.OrganizationRoleNotExistError)
	}

	resp := &dtotenant.OrganizationRoleDetailResp{
		OrganizationRoleID: orgRoleEntity.ID,
		OrganizationRoleBaseInfo: dtotenant.OrganizationRoleBaseInfo{
			TenantID:       orgRoleEntity.TenantID,
			OrganizationID: orgRoleEntity.OrganizationID,
			Name:           orgRoleEntity.Name,
			Description:    orgRoleEntity.Description,
			Type:           orgRoleEntity.Type,
		},
	}
	return resp, nil
}

func (svc *organizationRoleSvc) PageList(ctx *gin.Context, req *dtotenant.OrganizationRolePageListReq) (*dtotenant.OrganizationRolePageListResp, error) {
	cond := &dao.OrganizationRoleCond{
		BaseCond: &gormdao.BaseCond{
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

	list := make([]dtotenant.OrganizationRolePageListItem, 0, len(orgRoleEntityList))
	for _, v := range orgRoleEntityList {
		list = append(list, dtotenant.OrganizationRolePageListItem{
			OrganizationRoleID: v.ID,
			OrganizationRoleBaseInfo: dtotenant.OrganizationRoleBaseInfo{
				TenantID:       v.TenantID,
				OrganizationID: v.OrganizationID,
				Name:           v.Name,
				Description:    v.Description,
				Type:           v.Type,
			},
		})
	}
	return &dtotenant.OrganizationRolePageListResp{
		List:  list,
		Total: total,
	}, nil
}
