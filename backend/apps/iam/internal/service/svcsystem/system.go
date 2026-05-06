package svcsystem

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtosystem"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objsystem"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type SystemSvc interface {
	Create(ctx *gin.Context, req *dtosystem.SystemCreateReq) (*dtosystem.SystemCreateResp, error)
	Delete(ctx *gin.Context, req *dtosystem.SystemDeleteReq) error
	Update(ctx *gin.Context, req *dtosystem.SystemUpdateReq) error
	Detail(ctx *gin.Context, req *dtosystem.SystemDetailReq) (*dtosystem.SystemDetailResp, error)
	PageList(ctx *gin.Context, req *dtosystem.SystemPageListReq) (*dtosystem.SystemPageListResp, error)
}

type systemSvc struct {
}

var _ SystemSvc = (*systemSvc)(nil)

func NewSystemSvc() SystemSvc {
	return &systemSvc{}
}

func (svc *systemSvc) Create(ctx *gin.Context, req *dtosystem.SystemCreateReq) (*dtosystem.SystemCreateResp, error) {
	valueJson, err := json.Marshal(req.Value)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Create] json.Marshal fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SystemCreateError)
	}

	insertEntity := &model.SystemEntity{
		TenantID:  req.TenantID,
		Key:       req.Key,
		Value:     valueJson,
		CreatedBy: gincontext.GetUserID(ctx),
	}

	if err := dao.NewSystemDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcsystem.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SystemCreateError)
	}
	return &dtosystem.SystemCreateResp{
		SystemID: insertEntity.ID,
	}, nil
}

func (svc *systemSvc) Delete(ctx *gin.Context, req *dtosystem.SystemDeleteReq) error {
	systemEntity, err := dao.NewSystemDao().GetByID(ctx, req.SystemID)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SystemDeleteError)
	}
	if systemEntity == nil || systemEntity.ID == 0 {
		return code.GetError(code.SystemNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewSystemDao().Delete(ctx, req.SystemID, userID); err != nil {
		glog.Errorf(ctx, "[svcsystem.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SystemDeleteError)
	}
	return nil
}

func (svc *systemSvc) Update(ctx *gin.Context, req *dtosystem.SystemUpdateReq) error {
	systemEntity, err := dao.NewSystemDao().GetByID(ctx, req.SystemID)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SystemUpdateError)
	}
	if systemEntity == nil || systemEntity.ID == 0 {
		return code.GetError(code.SystemNotExistError)
	}

	valueJson, err := json.Marshal(req.Value)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Update] json.Marshal fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.SystemUpdateError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":  req.TenantID,
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

func (svc *systemSvc) Detail(ctx *gin.Context, req *dtosystem.SystemDetailReq) (*dtosystem.SystemDetailResp, error) {
	systemEntity, err := dao.NewSystemDao().GetByID(ctx, req.SystemID)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SystemGetDetailError)
	}
	if systemEntity == nil || systemEntity.ID == 0 {
		return nil, code.GetError(code.SystemNotExistError)
	}

	var value any
	if err := json.Unmarshal(systemEntity.Value, &value); err != nil {
		glog.Errorf(ctx, "[svcsystem.Detail] json.Unmarshal fail, err:%v", err)
		return nil, code.GetError(code.SystemGetDetailError)
	}

	resp := &dtosystem.SystemDetailResp{
		SystemID: systemEntity.ID,
		SystemBaseInfo: objsystem.SystemBaseInfo{
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

func (svc *systemSvc) PageList(ctx *gin.Context, req *dtosystem.SystemPageListReq) (*dtosystem.SystemPageListResp, error) {
	cond := &dao.SystemCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		Key:      req.Key,
	}
	systemEntityList, total, err := dao.NewSystemDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.SystemGetPageListError)
	}

	list := make([]dtosystem.SystemPageListItem, 0, len(systemEntityList))
	for _, v := range systemEntityList {
		var value any
		if err := json.Unmarshal(v.Value, &value); err != nil {
			glog.Errorf(ctx, "[svcsystem.PageList] json.Unmarshal fail, err:%v", err)
			continue
		}
		list = append(list, dtosystem.SystemPageListItem{
			SystemID: v.ID,
			SystemBaseInfo: objsystem.SystemBaseInfo{
				TenantID: v.TenantID,
				Key:      v.Key,
				Value:    value,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtosystem.SystemPageListResp{
		List:  list,
		Total: total,
	}, nil
}