package svcsystem

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtosystem"
	"github.com/morehao/ark-iam/iam/object/objsystem"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type LogSvc interface {
	Detail(ctx *gin.Context, req *dtosystem.LogDetailReq) (*dtosystem.LogDetailResp, error)
	PageList(ctx *gin.Context, req *dtosystem.LogPageListReq) (*dtosystem.LogPageListResp, error)
}

type logSvc struct {
}

var _ LogSvc = (*logSvc)(nil)

func NewLogSvc() LogSvc {
	return &logSvc{}
}

func (svc *logSvc) Detail(ctx *gin.Context, req *dtosystem.LogDetailReq) (*dtosystem.LogDetailResp, error) {
	logEntity, err := dao.NewLogDao().GetByID(ctx, req.LogID)
	if err != nil {
		glog.Errorf(ctx, "[svcsystem.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.LogGetDetailError)
	}
	if logEntity == nil || logEntity.ID == 0 {
		return nil, code.GetError(code.LogNotExistError)
	}

	var payload any
	if err := json.Unmarshal(logEntity.Payload, &payload); err != nil {
		glog.Errorf(ctx, "[svcsystem.Detail] json.Unmarshal fail, err:%v", err)
		return nil, code.GetError(code.LogGetDetailError)
	}

	resp := &dtosystem.LogDetailResp{
		LogID: logEntity.ID,
		LogBaseInfo: objsystem.LogBaseInfo{
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

func (svc *logSvc) PageList(ctx *gin.Context, req *dtosystem.LogPageListReq) (*dtosystem.LogPageListResp, error) {
	cond := &dao.LogCond{
		BaseCond: &genericdao.BaseCond{
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

	list := make([]dtosystem.LogPageListItem, 0, len(logEntityList))
	for _, v := range logEntityList {
		var payload any
		if err := json.Unmarshal(v.Payload, &payload); err != nil {
			glog.Errorf(ctx, "[svcsystem.PageList] json.Unmarshal fail, err:%v", err)
			continue
		}
		list = append(list, dtosystem.LogPageListItem{
			LogID: v.ID,
			LogBaseInfo: objsystem.LogBaseInfo{
				TenantID: v.TenantID,
				Key:      v.Key,
				Payload:  payload,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtosystem.LogPageListResp{
		List:  list,
		Total: total,
	}, nil
}