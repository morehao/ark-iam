package svctenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objtenant"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type TenantSvc interface {
	Create(ctx *gin.Context, req *dtotenant.TenantCreateReq) (*dtotenant.TenantCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.TenantDeleteReq) error
	Update(ctx *gin.Context, req *dtotenant.TenantUpdateReq) error
	Detail(ctx *gin.Context, req *dtotenant.TenantDetailReq) (*dtotenant.TenantDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenant.TenantPageListReq) (*dtotenant.TenantPageListResp, error)
}

type tenantSvc struct {
}

var _ TenantSvc = (*tenantSvc)(nil)

func NewTenantSvc() TenantSvc {
	return &tenantSvc{}
}

// Create 创建租户管理
func (svc *tenantSvc) Create(ctx *gin.Context, req *dtotenant.TenantCreateReq) (*dtotenant.TenantCreateResp, error) {
	insertEntity := &model.TenantEntity{
		DbUser:      req.DbUser,
		IsSuspended: req.IsSuspended,
		Name:        req.Name,
		Tag:         req.Tag,
	}

	if err := dao.NewTenantDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svctenant.TenantCreate] dao Create fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantCreateError)
	}
	return &dtotenant.TenantCreateResp{
		TenantID: insertEntity.ID,
	}, nil
}

// Delete 删除租户管理
func (svc *tenantSvc) Delete(ctx *gin.Context, req *dtotenant.TenantDeleteReq) error {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantDelete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantDeleteError)
	}
	if tenantEntity == nil || tenantEntity.ID == 0 {
		return code.GetError(code.TenantNotExistError)
	}

	userID := gincontext.GetUserID(ctx)

	if err := dao.NewTenantDao().Delete(ctx, req.TenantID, userID); err != nil {
		glog.Errorf(ctx, "[svctenant.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantDeleteError)
	}
	return nil
}

// Update 更新租户管理
func (svc *tenantSvc) Update(ctx *gin.Context, req *dtotenant.TenantUpdateReq) error {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantUpdate] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantUpdateError)
	}
	if tenantEntity == nil || tenantEntity.ID == 0 {
		return code.GetError(code.TenantNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"db_user":      req.DbUser,
		"is_suspended": req.IsSuspended,
		"name":         req.Name,
		"tag":          req.Tag,
		"updated_by":   userID,
	}
	if err := dao.NewTenantDao().UpdateMap(ctx, req.TenantID, updateMap); err != nil {
		glog.Errorf(ctx, "[svctenant.TenantUpdate] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantUpdateError)
	}
	return nil
}

// Detail 根据id获取租户管理
func (svc *tenantSvc) Detail(ctx *gin.Context, req *dtotenant.TenantDetailReq) (*dtotenant.TenantDetailResp, error) {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantDetail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantGetDetailError)
	}
	if tenantEntity == nil || tenantEntity.ID == 0 {
		return nil, code.GetError(code.TenantNotExistError)
	}
	resp := &dtotenant.TenantDetailResp{
		TenantID: tenantEntity.ID,
		TenantBaseInfo: objtenant.TenantBaseInfo{
			DbUser:      tenantEntity.DbUser,
			IsSuspended: tenantEntity.IsSuspended,
			Name:        tenantEntity.Name,
			Tag:         tenantEntity.Tag,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: tenantEntity.CreatedAt.Unix(),
			UpdatedAt: tenantEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

// PageList 分页获取租户管理列表
func (svc *tenantSvc) PageList(ctx *gin.Context, req *dtotenant.TenantPageListReq) (*dtotenant.TenantPageListResp, error) {
	cond := &dao.TenantCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
	}
	tenantEntityList, total, err := dao.NewTenantDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantPageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantGetPageListError)
	}
	list := make([]dtotenant.TenantPageListItem, 0, len(tenantEntityList))
	for _, v := range tenantEntityList {
		list = append(list, dtotenant.TenantPageListItem{
			TenantID: v.ID,
			TenantBaseInfo: objtenant.TenantBaseInfo{
				DbUser:      v.DbUser,
				IsSuspended: v.IsSuspended,
				Name:        v.Name,
				Tag:         v.Tag,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtotenant.TenantPageListResp{
		List:  list,
		Total: total,
	}, nil
}
