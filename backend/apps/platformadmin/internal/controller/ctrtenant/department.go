package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/platformadmin/internal/service/svctenant"
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
	departmentSvc svctenant.DepartmentSvc
}

var _ DepartmentCtr = (*departmentCtr)(nil)

func NewDepartmentCtr() DepartmentCtr {
	return &departmentCtr{
		departmentSvc: svctenant.NewDepartmentSvc(),
	}
}

// @Tags 部门
// @Summary 创建部门
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.DepartmentCreateReq true "创建部门"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.DepartmentCreateResp}
// @Router /v1/iam/department/create [post]
func (ctr *departmentCtr) Create(ctx *gin.Context) {
	var req dtotenant.DepartmentCreateReq
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

// @Tags 部门
// @Summary 删除部门
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.DepartmentDeleteReq true "删除部门"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/department/delete [post]
func (ctr *departmentCtr) Delete(ctx *gin.Context) {
	var req dtotenant.DepartmentDeleteReq
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

// @Tags 部门
// @Summary 修改部门
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.DepartmentUpdateReq true "修改部门"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/department/update [post]
func (ctr *departmentCtr) Update(ctx *gin.Context) {
	var req dtotenant.DepartmentUpdateReq
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

// @Tags 部门
// @Summary 部门详情
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.DepartmentDetailReq true "部门详情"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.DepartmentDetailResp}
// @Router /v1/iam/department/detail [get]
func (ctr *departmentCtr) Detail(ctx *gin.Context) {
	var req dtotenant.DepartmentDetailReq
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

// @Tags 部门
// @Summary 部门列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.DepartmentPageListReq true "部门列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.DepartmentPageListResp}
// @Router /v1/iam/department/pageList [post]
func (ctr *departmentCtr) PageList(ctx *gin.Context) {
	var req dtotenant.DepartmentPageListReq
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

// @Tags 部门
// @Summary 部门树
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.DepartmentTreeReq true "部门树"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.DepartmentTreeResp}
// @Router /v1/iam/department/tree [get]
func (ctr *departmentCtr) Tree(ctx *gin.Context) {
	var req dtotenant.DepartmentTreeReq
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
