package svctenant

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/gctx"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

func systemVisibleToTenant(entity *model.SystemEntity, tenantID string) bool {
	return entity != nil && entity.ID != "" && entity.TenantID == tenantID
}

type SystemSvc interface {
	Create(ctx *gin.Context, req *dtotenant.SystemCreateReq) (*dtotenant.SystemCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.SystemDeleteReq) error
	Update(ctx *gin.Context, req *dtotenant.SystemUpdateReq) error
	Detail(ctx *gin.Context, req *dtotenant.SystemDetailReq) (*dtotenant.SystemDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenant.SystemPageListReq) (*dtotenant.SystemPageListResp, error)
}

type systemSvc struct {
}

var _ SystemSvc = (*systemSvc)(nil)

func NewSystemSvc() SystemSvc {
	return &systemSvc{}
}

func (svc *systemSvc) Create(ctx *gin.Context, req *dtotenant.SystemCreateReq) (*dtotenant.SystemCreateResp, error) {
	valueJson, err := json.Marshal(req.Value)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Create] json.Marshal fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SystemCreateError)
	}

	tenantID := gctx.GetTenantID(ctx)
	if req.TenantID == "" {
		req.TenantID = tenantID
	}
	insertEntity := &model.SystemEntity{
		TenantID:  req.TenantID,
		Key:       req.Key,
		Value:     valueJson,
		CreatedBy: gctx.GetUserID(ctx),
	}

	if err := dao.NewSystemDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcsystem.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SystemCreateError)
	}
	return &dtotenant.SystemCreateResp{
		SystemID: insertEntity.ID,
	}, nil
}

func (svc *systemSvc) Delete(ctx *gin.Context, req *dtotenant.SystemDeleteReq) error {
	systemEntity, err := dao.NewSystemDao().GetByID(ctx, req.SystemID)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SystemDeleteError)
	}
	if !systemVisibleToTenant(systemEntity, gctx.GetTenantID(ctx)) {
		return code.GetError(code.SystemNotExistError)
	}

	userID := gctx.GetUserID(ctx)
	if err := dao.NewSystemDao().Delete(ctx, req.SystemID, userID); err != nil {
		glog.Errorf(ctx, "[svcsystem.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SystemDeleteError)
	}
	return nil
}

func (svc *systemSvc) Update(ctx *gin.Context, req *dtotenant.SystemUpdateReq) error {
	systemEntity, err := dao.NewSystemDao().GetByID(ctx, req.SystemID)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SystemUpdateError)
	}
	if !systemVisibleToTenant(systemEntity, gctx.GetTenantID(ctx)) {
		return code.GetError(code.SystemNotExistError)
	}

	valueJson, err := json.Marshal(req.Value)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Update] json.Marshal fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SystemUpdateError)
	}

	userID := gctx.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":  gctx.GetTenantID(ctx),
		"key":        req.Key,
		"value":      valueJson,
		"updated_by": userID,
	}
	if err := dao.NewSystemDao().UpdateMap(ctx, req.SystemID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcsystem.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SystemUpdateError)
	}
	return nil
}

func (svc *systemSvc) Detail(ctx *gin.Context, req *dtotenant.SystemDetailReq) (*dtotenant.SystemDetailResp, error) {
	systemEntity, err := dao.NewSystemDao().GetByID(ctx, req.SystemID)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SystemGetDetailError)
	}
	if !systemVisibleToTenant(systemEntity, gctx.GetTenantID(ctx)) {
		return nil, code.GetError(code.SystemNotExistError)
	}

	var value any
	if err := json.Unmarshal(systemEntity.Value, &value); err != nil {
		glog.Errorf(ctx, "[svcsystem.Detail] json.Unmarshal fail, err:%v", err)
		return nil, code.GetError(code.SystemGetDetailError)
	}

	resp := &dtotenant.SystemDetailResp{
		SystemID: systemEntity.ID,
		SystemBaseInfo: objtenant.SystemBaseInfo{
			TenantID: systemEntity.TenantID,
			Key:      systemEntity.Key,
			Value:    value,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: systemEntity.CreatedAt.Unix(),
			UpdatedAt: systemEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *systemSvc) PageList(ctx *gin.Context, req *dtotenant.SystemPageListReq) (*dtotenant.SystemPageListResp, error) {
	cond := &dao.SystemCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: gctx.GetTenantID(ctx),
		Key:      req.Key,
	}
	systemEntityList, total, err := dao.NewSystemDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SystemGetPageListError)
	}

	list := make([]dtotenant.SystemPageListItem, 0, len(systemEntityList))
	for _, v := range systemEntityList {
		var value any
		if err := json.Unmarshal(v.Value, &value); err != nil {
			glog.Errorf(ctx, "[svcsystem.PageList] json.Unmarshal fail, err:%v", err)
			continue
		}
		list = append(list, dtotenant.SystemPageListItem{
			SystemID: v.ID,
			SystemBaseInfo: objtenant.SystemBaseInfo{
				TenantID: v.TenantID,
				Key:      v.Key,
				Value:    value,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtotenant.SystemPageListResp{
		List:  list,
		Total: total,
	}, nil
}
