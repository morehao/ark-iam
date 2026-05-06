package ctrdepartment

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtodepartment"
	"github.com/morehao/ark-iam/iam/internal/service/svcdepartment"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type DepartmentCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	Tree(ctx *gin.Context)
}

type departmentCtr struct {
	departmentSvc svcdepartment.DepartmentSvc
}

var _ DepartmentCtr = (*departmentCtr)(nil)

func NewDepartmentCtr() DepartmentCtr {
	return &departmentCtr{
		departmentSvc: svcdepartment.NewDepartmentSvc(),
	}
}

func (ctr *departmentCtr) Create(ctx *gin.Context) {
	var req dtodepartment.DepartmentCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.departmentSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *departmentCtr) Delete(ctx *gin.Context) {
	var req dtodepartment.DepartmentDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.departmentSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *departmentCtr) Update(ctx *gin.Context) {
	var req dtodepartment.DepartmentUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.departmentSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

func (ctr *departmentCtr) Detail(ctx *gin.Context) {
	var req dtodepartment.DepartmentDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.departmentSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *departmentCtr) PageList(ctx *gin.Context) {
	var req dtodepartment.DepartmentPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.departmentSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *departmentCtr) Tree(ctx *gin.Context) {
	var req dtodepartment.DepartmentTreeReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.departmentSvc.Tree(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}