package svctenant

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objaudit"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type logScopeRepository interface {
	GetByID(ctx context.Context, id uint) (*model.LogEntity, error)
}

var newLogScopeRepo = func() logScopeRepository {
	return dao.NewLogDao()
}

func logVisibleToTenant(entity *model.LogEntity, tenantID uint) bool {
	return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}

type LogSvc interface {
	Detail(ctx *gin.Context, req *dtotenant.LogDetailReq) (*dtotenant.LogDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenant.LogPageListReq) (*dtotenant.LogPageListResp, error)
}

type logSvc struct {
}

var _ LogSvc = (*logSvc)(nil)

func NewLogSvc() LogSvc {
	return &logSvc{}
}

func (svc *logSvc) Detail(ctx *gin.Context, req *dtotenant.LogDetailReq) (*dtotenant.LogDetailResp, error) {
	logEntity, err := newLogScopeRepo().GetByID(ctx, req.LogID)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.LogGetDetailError)
	}
	if !logVisibleToTenant(logEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.LogNotExistError)
	}

	var payload any
	if err := json.Unmarshal(logEntity.Payload, &payload); err != nil {
		glog.Errorf(ctx, "[svcsystem.Detail] json.Unmarshal fail, err:%v", err)
		return nil, code.GetError(code.LogGetDetailError)
	}

	resp := &dtotenant.LogDetailResp{
		LogID: logEntity.ID,
		LogBaseInfo: objaudit.LogBaseInfo{
			TenantID: logEntity.TenantID,
			Key:      logEntity.Key,
			Payload:  payload,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: logEntity.CreatedAt.Unix(),
			UpdatedAt: logEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *logSvc) PageList(ctx *gin.Context, req *dtotenant.LogPageListReq) (*dtotenant.LogPageListResp, error) {
	cond := &dao.LogCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		Key:      req.Key,
	}
	logEntityList, total, err := dao.NewLogDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.LogGetPageListError)
	}

	list := make([]dtotenant.LogPageListItem, 0, len(logEntityList))
	for _, v := range logEntityList {
		var payload any
		if err := json.Unmarshal(v.Payload, &payload); err != nil {
			glog.Errorf(ctx, "[svcsystem.PageList] json.Unmarshal fail, err:%v", err)
			continue
		}
		list = append(list, dtotenant.LogPageListItem{
			LogID: v.ID,
			LogBaseInfo: objaudit.LogBaseInfo{
				TenantID: v.TenantID,
				Key:      v.Key,
				Payload:  payload,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtotenant.LogPageListResp{
		List:  list,
		Total: total,
	}, nil
}
