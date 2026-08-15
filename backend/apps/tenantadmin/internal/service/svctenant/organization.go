package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

func organizationVisibleToTenant(entity *model.OrganizationEntity, tenantID uint) bool {
	return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}

type OrganizationSvc interface {
	Create(ctx *gin.Context, req *dtotenant.OrganizationCreateReq) (*dtotenant.OrganizationCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.OrganizationDeleteReq) error
	Update(ctx *gin.Context, req *dtotenant.OrganizationUpdateReq) error
	Detail(ctx *gin.Context, req *dtotenant.OrganizationDetailReq) (*dtotenant.OrganizationDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenant.OrganizationPageListReq) (*dtotenant.OrganizationPageListResp, error)
}

type organizationSvc struct {
}

var _ OrganizationSvc = (*organizationSvc)(nil)

func NewOrganizationSvc() OrganizationSvc {
	return &organizationSvc{}
}

func (svc *organizationSvc) Create(ctx *gin.Context, req *dtotenant.OrganizationCreateReq) (*dtotenant.OrganizationCreateResp, error) {
	tenantID := gincontext.GetTenantID(ctx)
	if req.TenantID == 0 {
		req.TenantID = tenantID
	}
	insertEntity := &model.OrganizationEntity{
		TenantID:      req.TenantID,
		Name:          req.Name,
		Description:   req.Description,
		IsMFARequired: req.IsMFARequired,
		CreatedBy:     gincontext.GetUserID(ctx),
	}

	if err := dao.NewOrganizationDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcorganization.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationCreateError)
	}
	return &dtotenant.OrganizationCreateResp{
		OrganizationID: insertEntity.ID,
	}, nil
}

func (svc *organizationSvc) Delete(ctx *gin.Context, req *dtotenant.OrganizationDeleteReq) error {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationDeleteError)
	}
	if !organizationVisibleToTenant(orgEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.OrganizationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewOrganizationDao().Delete(ctx, req.OrganizationID, userID); err != nil {
		glog.Errorf(ctx, "[svcorganization.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationDeleteError)
	}
	return nil
}

func (svc *organizationSvc) Update(ctx *gin.Context, req *dtotenant.OrganizationUpdateReq) error {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUpdateError)
	}
	if !organizationVisibleToTenant(orgEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.OrganizationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":       gincontext.GetTenantID(ctx),
		"name":            req.Name,
		"description":     req.Description,
		"is_mfa_required": req.IsMFARequired,
		"updated_by":      userID,
	}
	if err := dao.NewOrganizationDao().UpdateMap(ctx, req.OrganizationID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcorganization.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUpdateError)
	}
	return nil
}

func (svc *organizationSvc) Detail(ctx *gin.Context, req *dtotenant.OrganizationDetailReq) (*dtotenant.OrganizationDetailResp, error) {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetDetailError)
	}
	if !organizationVisibleToTenant(orgEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	resp := &dtotenant.OrganizationDetailResp{
		OrganizationID: orgEntity.ID,
		OrganizationBaseInfo: objtenant.OrganizationBaseInfo{
			TenantID:      orgEntity.TenantID,
			Name:          orgEntity.Name,
			Description:   orgEntity.Description,
			IsMFARequired: orgEntity.IsMFARequired,
		},
	}
	return resp, nil
}

func (svc *organizationSvc) PageList(ctx *gin.Context, req *dtotenant.OrganizationPageListReq) (*dtotenant.OrganizationPageListResp, error) {
	cond := &dao.OrganizationCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: gincontext.GetTenantID(ctx),
		Name:     req.Name,
	}
	orgEntityList, total, err := dao.NewOrganizationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetPageListError)
	}

	list := make([]dtotenant.OrganizationPageListItem, 0, len(orgEntityList))
	for _, v := range orgEntityList {
		list = append(list, dtotenant.OrganizationPageListItem{
			OrganizationID: v.ID,
			OrganizationBaseInfo: objtenant.OrganizationBaseInfo{
				TenantID:      v.TenantID,
				Name:          v.Name,
				Description:   v.Description,
				IsMFARequired: v.IsMFARequired,
			},
		})
	}
	return &dtotenant.OrganizationPageListResp{
		List:  list,
		Total: total,
	}, nil
}
