package ctrlog

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtosystem"
	"github.com/morehao/ark-iam/iam/internal/service/svcsystem"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type LogCtr interface {
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type logCtr struct {
	logSvc svcsystem.LogSvc
}

var _ LogCtr = (*logCtr)(nil)

func NewLogCtr() LogCtr {
	return &logCtr{
		logSvc: svcsystem.NewLogSvc(),
	}
}

func (ctr *logCtr) Detail(ctx *gin.Context) {
	var req dtosystem.LogDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.logSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *logCtr) PageList(ctx *gin.Context) {
	var req dtosystem.LogPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.logSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}