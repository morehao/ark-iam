package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type DepartmentUserCtr interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	PageList(ctx *gin.Context)
}

type departmentUserCtr struct {
	departmentUserSvc svctenant.DepartmentUserSvc
}

var _ DepartmentUserCtr = (*departmentUserCtr)(nil)

func NewDepartmentUserCtr() DepartmentUserCtr {
	return &departmentUserCtr{
		departmentUserSvc: svctenant.NewDepartmentUserSvc(),
	}
}

// @Tags 部门关系
// @Summary 添加部门关系（primary/secondary/leader）
// @accept application/json
// @Produce application/json
// @Param departmentID path string true "部门ID"
// @Param req body dtotenant.DepartmentUserCreateReq true "添加关系"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.DepartmentUserCreateResp}
// @Router /v1/tenant/departments/{departmentID}/users [post]
func (ctr *departmentUserCtr) Create(ctx *gin.Context) {
	var req dtotenant.DepartmentUserCreateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.departmentUserSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 部门关系
// @Summary 更新部门关系（relationType）
// @accept application/json
// @Produce application/json
// @Param departmentID path string true "部门ID"
// @Param userID path string true "用户ID"
// @Param req body dtotenant.DepartmentUserUpdateReq true "更新关系"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/departments/{departmentID}/users/{userID} [put]
func (ctr *departmentUserCtr) Update(ctx *gin.Context) {
	var req dtotenant.DepartmentUserUpdateReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.departmentUserSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 部门关系
// @Summary 移除部门关系（含 primary/secondary/leader）
// @accept application/json
// @Produce application/json
// @Param departmentID path string true "部门ID"
// @Param userID path string true "用户ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/tenant/departments/{departmentID}/users/{userID} [delete]
func (ctr *departmentUserCtr) Delete(ctx *gin.Context) {
	var req dtotenant.DepartmentUserDeleteReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.departmentUserSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "移除成功")
}

// @Tags 部门关系
// @Summary 部门关系分页
// @accept application/json
// @Produce application/json
// @Param departmentID path string true "部门ID"
// @Param req query dtotenant.DepartmentUserPageListReq true "关系分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.DepartmentUserPageListResp}
// @Router /v1/tenant/departments/{departmentID}/users [get]
func (ctr *departmentUserCtr) PageList(ctx *gin.Context) {
	var req dtotenant.DepartmentUserPageListReq
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.departmentUserSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
