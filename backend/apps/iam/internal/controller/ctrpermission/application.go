package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/iam/internal/service/svcapplication"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type ApplicationCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
	ListRoles(ctx *gin.Context)
	AssignRoles(ctx *gin.Context)
	RemoveRole(ctx *gin.Context)
	ListSecrets(ctx *gin.Context)
	CreateSecret(ctx *gin.Context)
	DeleteSecret(ctx *gin.Context)
}

type applicationCtr struct {
	applicationSvc svcapplication.ApplicationSvc
}

var _ ApplicationCtr = (*applicationCtr)(nil)

func NewApplicationCtr() ApplicationCtr {
	return &applicationCtr{
		applicationSvc: svcapplication.NewApplicationSvc(),
	}
}

// @Tags 应用管理
// @Summary 创建应用管理
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.ApplicationCreateReq true "创建应用管理"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.ApplicationCreateResp}
// @Router /v1/iam/application/create [post]
func (ctr *applicationCtr) Create(ctx *gin.Context) {
	var req dtoapplication.ApplicationCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 删除应用管理
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.ApplicationDeleteReq true "删除应用管理"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/delete [post]
func (ctr *applicationCtr) Delete(ctx *gin.Context) {
	var req dtoapplication.ApplicationDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.applicationSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

// @Tags 应用管理
// @Summary 修改应用管理
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.ApplicationUpdateReq true "修改应用管理"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/update [post]
func (ctr *applicationCtr) Update(ctx *gin.Context) {
	var req dtoapplication.ApplicationUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.applicationSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

// @Tags 应用管理
// @Summary 应用管理详情
// @accept application/json
// @Produce application/json
// @Param req query dtoapplication.ApplicationDetailReq true "应用管理详情"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.ApplicationDetailResp}
// @Router /v1/iam/application/detail [get]
func (ctr *applicationCtr) Detail(ctx *gin.Context) {
	var req dtoapplication.ApplicationDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 应用管理列表分页
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.ApplicationPageListReq true "应用管理列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.ApplicationPageListResp}
// @Router /v1/iam/application/pageList [post]
func (ctr *applicationCtr) PageList(ctx *gin.Context) {
	var req dtoapplication.ApplicationPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 应用角色列表
// @accept application/json
// @Produce application/json
// @Param req query dtoapplication.ApplicationRoleListReq true "应用角色列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.ApplicationRoleListResp}
// @Router /v1/iam/application/roles [get]
func (ctr *applicationCtr) ListRoles(ctx *gin.Context) {
	var req dtoapplication.ApplicationRoleListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.ListRoles(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 分配角色
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.AssignApplicationRolesReq true "分配角色"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/assignRoles [post]
func (ctr *applicationCtr) AssignRoles(ctx *gin.Context) {
	var req dtoapplication.AssignApplicationRolesReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.applicationSvc.AssignRoles(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "分配成功")
}

// @Tags 应用管理
// @Summary 移除角色
// @accept application/json
// @Produce application/json
// @Param roleId path int true "角色ID"
// @Param applicationId query int true "应用ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/roles/{roleId} [delete]
func (ctr *applicationCtr) RemoveRole(ctx *gin.Context) {
	var uriReq struct {
		RoleID uint64 `uri:"roleId" binding:"required"`
	}
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	var queryReq struct {
		ApplicationID uint64 `form:"applicationId" binding:"required"`
	}
	if err := ctx.ShouldBindQuery(&queryReq); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	req := dtoapplication.RemoveApplicationRoleReq{
		ApplicationID: queryReq.ApplicationID,
		RoleID:        uriReq.RoleID,
	}
	if err := ctr.applicationSvc.RemoveRole(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "移除成功")
}

// @Tags 应用管理
// @Summary 应用密钥列表
// @accept application/json
// @Produce application/json
// @Param req query dtoapplication.ApplicationSecretListReq true "应用密钥列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.ApplicationSecretListResp}
// @Router /v1/iam/application/secrets [get]
func (ctr *applicationCtr) ListSecrets(ctx *gin.Context) {
	var req dtoapplication.ApplicationSecretListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.ListSecrets(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 创建应用密钥
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.CreateApplicationSecretReq true "创建应用密钥"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.CreateApplicationSecretResp}
// @Router /v1/iam/application/secrets [post]
func (ctr *applicationCtr) CreateSecret(ctx *gin.Context) {
	var req dtoapplication.CreateApplicationSecretReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.CreateSecret(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 删除应用密钥
// @accept application/json
// @Produce application/json
// @Param secretId path int true "密钥ID"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/secrets/{secretId} [delete]
func (ctr *applicationCtr) DeleteSecret(ctx *gin.Context) {
	var req dtoapplication.DeleteApplicationSecretReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.applicationSvc.DeleteSecret(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}
