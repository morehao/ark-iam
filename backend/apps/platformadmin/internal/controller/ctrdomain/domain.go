package ctrdomain

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtodomain"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svcdomain"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type DomainCtr interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

type domainCtr struct {
	domainSvc svcdomain.DomainSvc
}

var _ DomainCtr = (*domainCtr)(nil)

func NewDomainCtr() DomainCtr {
	return &domainCtr{
		domainSvc: svcdomain.NewDomainSvc(),
	}
}

// @Tags 域名管理
// @Summary 创建域名
// @accept application/json
// @Produce application/json
// @Param req body dtodomain.CreateDomainReq true "创建域名"
// @Success 200 {object} gincontext.DtoRender{data=dtodomain.DomainCreateResp}
// @Router /v1/domain/create [post]
func (ctr *domainCtr) Create(ctx *gin.Context) {
	var req dtodomain.CreateDomainReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 域名管理
// @Summary 域名列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtodomain.DomainPageListReq true "域名列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtodomain.DomainPageListResp}
// @Router /v1/domain/pageList [post]
func (ctr *domainCtr) PageList(ctx *gin.Context) {
	var req dtodomain.DomainPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 域名管理
// @Summary 修改域名
// @accept application/json
// @Produce application/json
// @Param req body dtodomain.UpdateDomainReq true "修改域名"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/domain/update [post]
func (ctr *domainCtr) Update(ctx *gin.Context) {
	var req dtodomain.UpdateDomainReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.domainSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 域名管理
// @Summary 域名详情
// @accept application/json
// @Produce application/json
// @Param req query dtodomain.DomainDetailReq true "域名详情"
// @Success 200 {object} gincontext.DtoRender{data=dtodomain.DomainDetailResp}
// @Router /v1/domain/detail [get]
func (ctr *domainCtr) Detail(ctx *gin.Context) {
	var req dtodomain.DomainDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.domainSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 域名管理
// @Summary 删除域名
// @accept application/json
// @Produce application/json
// @Param req body dtodomain.DeleteDomainReq true "删除域名"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/domain/delete [post]
func (ctr *domainCtr) Delete(ctx *gin.Context) {
	var req dtodomain.DeleteDomainReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.domainSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}
