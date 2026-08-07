package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type LogCtr interface {
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type logCtr struct {
	logSvc svctenant.LogSvc
}

var _ LogCtr = (*logCtr)(nil)

func NewLogCtr() LogCtr {
	return &logCtr{
		logSvc: svctenant.NewLogSvc(),
	}
}

// @Tags 日志
// @Summary 日志详情
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.LogDetailReq true "日志详情"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.LogDetailResp}
// @Router /v1/iam/log/detail [get]
func (ctr *logCtr) Detail(ctx *gin.Context) {
	var req dtotenant.LogDetailReq
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

// @Tags 日志
// @Summary 日志列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.LogPageListReq true "日志列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.LogPageListResp}
// @Router /v1/iam/log/pageList [post]
func (ctr *logCtr) PageList(ctx *gin.Context) {
	var req dtotenant.LogPageListReq
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