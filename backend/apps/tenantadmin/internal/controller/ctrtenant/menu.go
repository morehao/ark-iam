package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type TenantMenuCtr interface {
	Tree(ctx *gin.Context)
}

type tenantMenuCtr struct {
	tenantMenuSvc svctenant.TenantMenuSvc
}

var _ TenantMenuCtr = (*tenantMenuCtr)(nil)

func NewTenantMenuCtr() TenantMenuCtr {
	return &tenantMenuCtr{
		tenantMenuSvc: svctenant.NewTenantMenuSvc(),
	}
}

// @Tags 租户菜单
// @Summary 当前租户菜单树
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.MenuTreeResp}
// @Router /v1/tenant/menus/tree [get]
func (ctr *tenantMenuCtr) Tree(ctx *gin.Context) {
	res, err := ctr.tenantMenuSvc.Tree(ctx)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
