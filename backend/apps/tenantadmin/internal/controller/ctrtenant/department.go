package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type DepartmentCtr interface {
	Create(ctx *gin.Context)
	Tree(ctx *gin.Context)
	Children(ctx *gin.Context)
	Update(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
	Delete(ctx *gin.Context)
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
// @Summary 创建部门节点
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.DepartmentCreateReq true "创建部门"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.DepartmentCreateResp}
// @Router /v1/tenant/departments [post]
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
// @Summary 部门树
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.DepartmentTreeReq true "部门树查询"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.DepartmentTreeResp}
// @Router /v1/tenant/departments/tree [get]
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

// @Tags 部门
// @Summary 某部门直属子部门分页列表
// @accept application/json
// @Produce application/json
// @Param departmentID path string true "部门ID(父节点)"
// @Param req query dtotenant.DepartmentChildrenReq true "子部门查询"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.DepartmentChildrenResp}
// @Router /v1/tenant/departments/{departmentID}/children [get]
func (ctr *departmentCtr) Children(ctx *gin.Context) {
	var req dtotenant.DepartmentChildrenReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.departmentSvc.Children(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 部门
// @Summary 修改部门（改 parentID 即移动节点）
// @accept application/json
// @Produce application/json
// @Param departmentID path string true "部门ID"
// @Param req body dtotenant.DepartmentUpdateReq true "修改部门"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/departments/{departmentID} [put]
func (ctr *departmentCtr) Update(ctx *gin.Context) {
	var req dtotenant.DepartmentUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
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
// @Summary 更新部门状态（启停用）
// @accept application/json
// @Produce application/json
// @Param departmentID path string true "部门ID"
// @Param req body dtotenant.DepartmentStatusReq true "更新状态"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/departments/{departmentID} [patch]
func (ctr *departmentCtr) UpdateStatus(ctx *gin.Context) {
	var req dtotenant.DepartmentStatusReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.departmentSvc.UpdateStatus(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 部门
// @Summary 删除部门（有子节点/成员需 ?cascade=1）
// @accept application/json
// @Produce application/json
// @Param departmentID path string true "部门ID"
// @Param cascade query bool false "级联删除子树与成员"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/departments/{departmentID} [delete]
func (ctr *departmentCtr) Delete(ctx *gin.Context) {
	var req dtotenant.DepartmentDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.departmentSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}
