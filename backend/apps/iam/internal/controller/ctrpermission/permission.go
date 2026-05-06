package ctrpermission

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/iam/internal/dto/dtopermission"
	"github.com/morehao/ark-iam/iam/internal/service/svcapplication"
	"github.com/morehao/ark-iam/iam/internal/service/svcpermission"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type PermissionCtr interface {
	CreateRole(ctx *gin.Context)
	DeleteRole(ctx *gin.Context)
	UpdateRole(ctx *gin.Context)
	DetailRole(ctx *gin.Context)
	PageListRole(ctx *gin.Context)
	CreateMenu(ctx *gin.Context)
	DeleteMenu(ctx *gin.Context)
	UpdateMenu(ctx *gin.Context)
	DetailMenu(ctx *gin.Context)
	PageListMenu(ctx *gin.Context)
	TreeMenu(ctx *gin.Context)
	CreateResource(ctx *gin.Context)
	DeleteResource(ctx *gin.Context)
	UpdateResource(ctx *gin.Context)
	DetailResource(ctx *gin.Context)
	PageListResource(ctx *gin.Context)
	CreateScope(ctx *gin.Context)
	DeleteScope(ctx *gin.Context)
	UpdateScope(ctx *gin.Context)
	DetailScope(ctx *gin.Context)
	PageListScope(ctx *gin.Context)
	CreateRoleMenu(ctx *gin.Context)
	DeleteRoleMenu(ctx *gin.Context)
	PageListRoleMenu(ctx *gin.Context)
	CreateRoleScope(ctx *gin.Context)
	DeleteRoleScope(ctx *gin.Context)
	PageListRoleScope(ctx *gin.Context)
	CreateUserRole(ctx *gin.Context)
	DeleteUserRole(ctx *gin.Context)
	PageListUserRole(ctx *gin.Context)
}

type permissionCtr struct {
	permissionSvc svcpermission.PermissionSvc
}

var _ PermissionCtr = (*permissionCtr)(nil)

func NewPermissionCtr() PermissionCtr {
	return &permissionCtr{
		permissionSvc: svcpermission.NewPermissionSvc(),
	}
}

func (ctr *permissionCtr) CreateRole(ctx *gin.Context) {
	var req dtopermission.RoleCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.CreateRole(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) DeleteRole(ctx *gin.Context) {
	var req dtopermission.RoleDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.DeleteRole(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *permissionCtr) UpdateRole(ctx *gin.Context) {
	var req dtopermission.RoleUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.UpdateRole(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

func (ctr *permissionCtr) DetailRole(ctx *gin.Context) {
	var req dtopermission.RoleDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.DetailRole(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) PageListRole(ctx *gin.Context) {
	var req dtopermission.RolePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.PageListRole(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) CreateMenu(ctx *gin.Context) {
	var req dtopermission.MenuCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.CreateMenu(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) DeleteMenu(ctx *gin.Context) {
	var req dtopermission.MenuDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.DeleteMenu(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *permissionCtr) UpdateMenu(ctx *gin.Context) {
	var req dtopermission.MenuUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.UpdateMenu(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

func (ctr *permissionCtr) DetailMenu(ctx *gin.Context) {
	var req dtopermission.MenuDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.DetailMenu(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) PageListMenu(ctx *gin.Context) {
	var req dtopermission.MenuPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.PageListMenu(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) TreeMenu(ctx *gin.Context) {
	var req dtopermission.MenuTreeReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.TreeMenu(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) CreateResource(ctx *gin.Context) {
	var req dtopermission.ResourceCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.CreateResource(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) DeleteResource(ctx *gin.Context) {
	var req dtopermission.ResourceDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.DeleteResource(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *permissionCtr) UpdateResource(ctx *gin.Context) {
	var req dtopermission.ResourceUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.UpdateResource(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

func (ctr *permissionCtr) DetailResource(ctx *gin.Context) {
	var req dtopermission.ResourceDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.DetailResource(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) PageListResource(ctx *gin.Context) {
	var req dtopermission.ResourcePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.PageListResource(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) CreateScope(ctx *gin.Context) {
	var req dtopermission.ScopeCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.CreateScope(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) DeleteScope(ctx *gin.Context) {
	var req dtopermission.ScopeDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.DeleteScope(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *permissionCtr) UpdateScope(ctx *gin.Context) {
	var req dtopermission.ScopeUpdateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.UpdateScope(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "修改成功")
}

func (ctr *permissionCtr) DetailScope(ctx *gin.Context) {
	var req dtopermission.ScopeDetailReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.DetailScope(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) PageListScope(ctx *gin.Context) {
	var req dtopermission.ScopePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.PageListScope(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) CreateRoleMenu(ctx *gin.Context) {
	var req dtopermission.RoleMenuCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.CreateRoleMenu(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) DeleteRoleMenu(ctx *gin.Context) {
	var req dtopermission.RoleMenuDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.DeleteRoleMenu(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *permissionCtr) PageListRoleMenu(ctx *gin.Context) {
	var req dtopermission.RoleMenuPageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.PageListRoleMenu(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) CreateRoleScope(ctx *gin.Context) {
	var req dtopermission.RoleScopeCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.CreateRoleScope(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) DeleteRoleScope(ctx *gin.Context) {
	var req dtopermission.RoleScopeDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.DeleteRoleScope(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *permissionCtr) PageListRoleScope(ctx *gin.Context) {
	var req dtopermission.RoleScopePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.PageListRoleScope(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) CreateUserRole(ctx *gin.Context) {
	var req dtopermission.UserRoleCreateReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.CreateUserRole(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

func (ctr *permissionCtr) DeleteUserRole(ctx *gin.Context) {
	var req dtopermission.UserRoleDeleteReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.permissionSvc.DeleteUserRole(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}

func (ctr *permissionCtr) PageListUserRole(ctx *gin.Context) {
	var req dtopermission.UserRolePageListReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.permissionSvc.PageListUserRole(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

type ApplicationCtr interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Detail(ctx *gin.Context)
	PageList(ctx *gin.Context)
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
