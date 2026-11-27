package ctrdomain

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtodomain"
	"github.com/morehao/ark-iam/apps/platformadmin/service/svcdomain"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// DomainCtr 域名管理控制器。动作端点走 POST /domains/{id}/{action}（AGENTS.md 路由规范 R2）。
type DomainCtr interface {
	Create(ctx *gin.Context)
	PageList(ctx *gin.Context)
	Detail(ctx *gin.Context)
	Delete(ctx *gin.Context)

	Verify(ctx *gin.Context)
	TLSReady(ctx *gin.Context)
	Activate(ctx *gin.Context)
	Disable(ctx *gin.Context)
	Enable(ctx *gin.Context)
	Retire(ctx *gin.Context)
	Release(ctx *gin.Context)
}

type domainCtr struct {
	domainSvc svcdomain.DomainSvc
}

var _ DomainCtr = (*domainCtr)(nil)

func NewDomainCtr(db gormdao.DBGetter) DomainCtr {
	return &domainCtr{domainSvc: svcdomain.NewDomainSvc(db)}
}

// @Tags 域名管理
// @Summary 创建域名
// @accept application/json
// @Produce application/json
// @Param req body dtodomain.DomainCreateReq true "创建域名"
// @Success 200 {object} gincontext.DtoRender{data=dtodomain.DomainCreateResp}
// @Router /v1/platform/domains [post]
func (ctr *domainCtr) Create(ctx *gin.Context) {
	var req dtodomain.DomainCreateReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.Create(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 域名管理
// @Summary 域名列表分页
// @Param req query dtodomain.DomainPageListReq true "域名列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtodomain.DomainPageListResp}
// @Router /v1/platform/domains [get]
func (ctr *domainCtr) PageList(ctx *gin.Context) {
	var req dtodomain.DomainPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.PageList(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 域名管理
// @Summary 域名详情
// @Param domainID path string true "域名ID"
// @Success 200 {object} gincontext.DtoRender{data=dtodomain.DomainDetailResp}
// @Router /v1/platform/domains/{domainID} [get]
func (ctr *domainCtr) Detail(ctx *gin.Context) {
	var req dtodomain.DomainDetailReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.Detail(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 域名管理
// @Summary 删除未上线域名
// @Param domainID path string true "域名ID"
// @Router /v1/platform/domains/{domainID} [delete]
func (ctr *domainCtr) Delete(ctx *gin.Context) {
	var req dtodomain.DomainDeleteReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.domainSvc.Delete(ctx.Request.Context(), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

func (ctr *domainCtr) Verify(ctx *gin.Context) {
	var req dtodomain.DomainVerifyReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.Verify(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *domainCtr) TLSReady(ctx *gin.Context) {
	var req dtodomain.DomainTLSReadyReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.TLSReady(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *domainCtr) Activate(ctx *gin.Context) {
	var req dtodomain.DomainActivateReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.Activate(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *domainCtr) Disable(ctx *gin.Context) {
	var req dtodomain.DomainDisableReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.Disable(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *domainCtr) Enable(ctx *gin.Context) {
	var req dtodomain.DomainEnableReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.Enable(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *domainCtr) Retire(ctx *gin.Context) {
	var req dtodomain.DomainRetireReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.Retire(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *domainCtr) Release(ctx *gin.Context) {
	var req dtodomain.DomainReleaseReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.Release(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
