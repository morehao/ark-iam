package ctrrole_scope

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtorole"
	"github.com/morehao/ark-iam/iam/internal/service/svcrole_scope"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type RoleScopeCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type roleScopeCtr struct {
	roleScopeSvc svcrole_scope.RoleScopeSvc
}

var _ RoleScopeCtr = (*roleScopeCtr)(nil)

func NewRoleScopeCtr() RoleScopeCtr {
	return &roleScopeCtr{
		roleScopeSvc: svcrole_scope.NewRoleScopeSvc(),
	}
}

func (ctr *roleScopeCtr) Create(ctx *gin.Context) {
	var req dtorole.RoleScopeCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleScopeSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *roleScopeCtr) Delete(ctx *gin.Context) {
	var req dtorole.RoleScopeDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleScopeSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *roleScopeCtr) PageList(ctx *gin.Context) {
	var req dtorole.RoleScopePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleScopeSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}