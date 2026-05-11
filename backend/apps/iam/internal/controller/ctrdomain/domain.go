package ctrdomain

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtodomain"
	"github.com/morehao/ark-iam/iam/internal/service/svcdomain"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type DomainCtr interface {
	Create(ctx *gin.Context)
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
// @Router /v1/iam/domain/create [post]
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
// @Router /v1/iam/domain/pageList [post]
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
// @Summary 删除域名
// @accept application/json
// @Produce application/json
// @Param req body dtodomain.DeleteDomainReq true "删除域名"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/domain/delete [post]
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