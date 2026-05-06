package svcorganization

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoorganization"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objorganization"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type OrganizationSvc interface {
	Create(ctx *gin.Context, req *dtoorganization.OrganizationCreateReq) (*dtoorganization.OrganizationCreateResp, error)
	Delete(ctx *gin.Context, req *dtoorganization.OrganizationDeleteReq) error
	Update(ctx *gin.Context, req *dtoorganization.OrganizationUpdateReq) error
	Detail(ctx *gin.Context, req *dtoorganization.OrganizationDetailReq) (*dtoorganization.OrganizationDetailResp, error)
	PageList(ctx *gin.Context, req *dtoorganization.OrganizationPageListReq) (*dtoorganization.OrganizationPageListResp, error)
}

type organizationSvc struct {
}

var _ OrganizationSvc = (*organizationSvc)(nil)

func NewOrganizationSvc() OrganizationSvc {
	return &organizationSvc{}
}

func (svc *organizationSvc) Create(ctx *gin.Context, req *dtoorganization.OrganizationCreateReq) (*dtoorganization.OrganizationCreateResp, error) {
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
	return &dtoorganization.OrganizationCreateResp{
		OrganizationID: insertEntity.ID,
	}, nil
}

func (svc *organizationSvc) Delete(ctx *gin.Context, req *dtoorganization.OrganizationDeleteReq) error {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationDeleteError)
	}
	if orgEntity == nil || orgEntity.ID == 0 {
		return code.GetError(code.OrganizationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewOrganizationDao().Delete(ctx, req.OrganizationID, userID); err != nil {
		glog.Errorf(ctx, "[svcorganization.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationDeleteError)
	}
	return nil
}

func (svc *organizationSvc) Update(ctx *gin.Context, req *dtoorganization.OrganizationUpdateReq) error {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OrganizationUpdateError)
	}
	if orgEntity == nil || orgEntity.ID == 0 {
		return code.GetError(code.OrganizationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":       req.TenantID,
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

func (svc *organizationSvc) Detail(ctx *gin.Context, req *dtoorganization.OrganizationDetailReq) (*dtoorganization.OrganizationDetailResp, error) {
	orgEntity, err := dao.NewOrganizationDao().GetByID(ctx, req.OrganizationID)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetDetailError)
	}
	if orgEntity == nil || orgEntity.ID == 0 {
		return nil, code.GetError(code.OrganizationNotExistError)
	}

	resp := &dtoorganization.OrganizationDetailResp{
		OrganizationID: orgEntity.ID,
		OrganizationBaseInfo: objorganization.OrganizationBaseInfo{
			TenantID:      orgEntity.TenantID,
			Name:          orgEntity.Name,
			Description:   orgEntity.Description,
			IsMFARequired: orgEntity.IsMFARequired,
		},
	}
	return resp, nil
}

func (svc *organizationSvc) PageList(ctx *gin.Context, req *dtoorganization.OrganizationPageListReq) (*dtoorganization.OrganizationPageListResp, error) {
	cond := &dao.OrganizationCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		Name:     req.Name,
	}
	orgEntityList, total, err := dao.NewOrganizationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcorganization.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OrganizationGetPageListError)
	}

	list := make([]dtoorganization.OrganizationPageListItem, 0, len(orgEntityList))
	for _, v := range orgEntityList {
		list = append(list, dtoorganization.OrganizationPageListItem{
			OrganizationID: v.ID,
			OrganizationBaseInfo: objorganization.OrganizationBaseInfo{
				TenantID:      v.TenantID,
				Name:          v.Name,
				Description:   v.Description,
				IsMFARequired: v.IsMFARequired,
			},
		})
	}
	return &dtoorganization.OrganizationPageListResp{
		List:  list,
		Total: total,
	}, nil
}