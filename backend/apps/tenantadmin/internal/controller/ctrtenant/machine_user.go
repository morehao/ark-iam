package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type MachineUserCtr interface {
	PageList(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	UpdateStatus(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Detail(ctx *gin.Context)
	ListRoles(ctx *gin.Context)
	UpdateRoles(ctx *gin.Context)
}

type machineUserCtr struct {
	machineUserSvc svctenant.MachineUserSvc
}

var _ MachineUserCtr = (*machineUserCtr)(nil)

func NewMachineUserCtr() MachineUserCtr {
	return &machineUserCtr{
		machineUserSvc: svctenant.NewMachineUserSvc(),
	}
}

// @Tags 服务账号
// @Summary 服务账号列表分页
// @accept application/json
// @Produce application/json
// @Param req query dtotenant.MachineUserPageListReq true "服务账号列表分页"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.MachineUserPageListResp}
// @Router /v1/tenant/machine-users [get]
func (ctr *machineUserCtr) PageList(ctx *gin.Context) {
	var req dtotenant.MachineUserPageListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.machineUserSvc.PageList(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 服务账号
// @Summary 创建服务账号
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.MachineUserCreateReq true "创建服务账号"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.MachineUserCreateResp}
// @Router /v1/tenant/machine-users [post]
func (ctr *machineUserCtr) Create(ctx *gin.Context) {
	var req dtotenant.MachineUserCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.machineUserSvc.Create(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 服务账号
// @Summary 修改服务账号
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.MachineUserUpdateReq true "修改服务账号"
// @Success 200 {object} gincontext.DtoRender
// @Router /v1/tenant/machine-users/{machineUserID} [put]
func (ctr *machineUserCtr) Update(ctx *gin.Context) {
	var req dtotenant.MachineUserUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.machineUserSvc.Update(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

// @Tags 服务账号
// @Summary 挂起/启用服务账号
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.MachineUserStatusReq true "挂起/启用服务账号"
// @Success 200 {object} gincontext.DtoRender
// @Router /v1/tenant/machine-users/{machineUserID} [patch]
func (ctr *machineUserCtr) UpdateStatus(ctx *gin.Context) {
	var req dtotenant.MachineUserStatusReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.machineUserSvc.UpdateStatus(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

// @Tags 服务账号
// @Summary 删除服务账号
// @accept application/json
// @Produce application/json
// @Param machineUserID path string true "服务账号ID"
// @Success 200 {object} gincontext.DtoRender
// @Router /v1/tenant/machine-users/{machineUserID} [delete]
func (ctr *machineUserCtr) Delete(ctx *gin.Context) {
	var req dtotenant.MachineUserDeleteReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.machineUserSvc.Delete(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}

// @Tags 服务账号
// @Summary 服务账号详情
// @accept application/json
// @Produce application/json
// @Param machineUserID path string true "服务账号ID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.MachineUserDetailResp}
// @Router /v1/tenant/machine-users/{machineUserID} [get]
func (ctr *machineUserCtr) Detail(ctx *gin.Context) {
	var req dtotenant.MachineUserDetailReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.machineUserSvc.Detail(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 服务账号
// @Summary 服务账号已分配角色
// @accept application/json
// @Produce application/json
// @Param machineUserID path string true "服务账号ID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.UserRolesListResp}
// @Router /v1/tenant/machine-users/{machineUserID}/roles [get]
func (ctr *machineUserCtr) ListRoles(ctx *gin.Context) {
	var req dtotenant.MachineUserRolesListReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.machineUserSvc.ListRoles(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 服务账号
// @Summary 全量替换服务账号角色
// @accept application/json
// @Produce application/json
// @Param req body dtotenant.MachineUserRolesUpdateReq true "全量替换服务账号角色"
// @Success 200 {object} gincontext.DtoRender
// @Router /v1/tenant/machine-users/{machineUserID}/roles [put]
func (ctr *machineUserCtr) UpdateRoles(ctx *gin.Context) {
	var req dtotenant.MachineUserRolesUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.machineUserSvc.UpdateRoles(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, nil)
}
