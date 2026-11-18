package ctrauth

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/auth/dto/dtoconnector"
	"github.com/morehao/ark-iam/apps/auth/service/svcauth"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// ConnectorCtr 连接器「运行时」控制器：仅暴露登录链路（authorize / callback）。
// 管理面（CRUD/工厂/测试）已迁至 platformadmin/ctrconnector，路由前缀 /v1/platform。
type ConnectorCtr interface {
	Authorize(ctx *gin.Context)
	Callback(ctx *gin.Context)
}

type connectorCtr struct {
	connectorSvc svcauth.ConnectorSvc
}

var _ ConnectorCtr = (*connectorCtr)(nil)

func NewConnectorCtr(db gormdao.DBGetter) ConnectorCtr {
	return &connectorCtr{
		connectorSvc: svcauth.NewConnectorSvc(db),
	}
}

// @Tags 连接器
// @Summary 连接器授权
// @accept application/json
// @Produce application/json
// @Param connectorID path string true "连接器ID"
// @Param req body dtoconnector.ConnectorAuthorizeReq true "连接器授权"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.ConnectorAuthorizeResp}
// @Router /v1/auth/connectors/{connectorID}/authorize [post]
func (ctr *connectorCtr) Authorize(ctx *gin.Context) {
	var uriReq struct {
		ConnectorID string `uri:"connectorID" binding:"required"`
	}
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	var bodyReq struct {
		RedirectURI  string `json:"redirectUri" binding:"required"`
		State        string `json:"state"`
		LoginHint    string `json:"loginHint"`
		ResponseMode string `json:"responseMode"`
	}
	if err := ctx.ShouldBindJSON(&bodyReq); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	req := dtoconnector.ConnectorAuthorizeReq{
		ConnectorID:  uriReq.ConnectorID,
		RedirectURI:  bodyReq.RedirectURI,
		State:        bodyReq.State,
		LoginHint:    bodyReq.LoginHint,
		ResponseMode: bodyReq.ResponseMode,
	}
	res, err := ctr.connectorSvc.Authorize(ctx.Request.Context(), &req, req.ConnectorID)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 连接器回调
// @accept application/json
// @Produce application/json
// @Param connectorID query string false "连接器ID"
// @Param code query string true "授权码"
// @Param state query string true "状态"
// @Success 200 {object} gincontext.DtoRender{data=dtoauth.LoginResp}
// @Router /v1/auth/connectors/callback [get]
func (ctr *connectorCtr) Callback(ctx *gin.Context) {
	var req struct {
		ConnectorID string `form:"connectorID"`
		Code        string `form:"code" binding:"required"`
		State       string `form:"state" binding:"required"`
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.connectorSvc.Callback(ctx.Request.Context(), &dtoconnector.ConnectorCallbackReq{
		ConnectorID: req.ConnectorID,
		Code:        req.Code,
		State:       req.State,
	})
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
