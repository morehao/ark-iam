package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/apps/tenantadmin/dto/dtotenant"
	"github.com/morehao/ark-iam/apps/tenantadmin/service/svctenant"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
)

type TenantMenuCtr interface {
	Tree(ctx *gin.Context)
}

type tenantMenuCtr struct {
	tenantMenuSvc svctenant.TenantMenuSvc
}

var _ TenantMenuCtr = (*tenantMenuCtr)(nil)

func NewTenantMenuCtr(db gormdao.DBGetter) TenantMenuCtr {
	return &tenantMenuCtr{
		tenantMenuSvc: svctenant.NewTenantMenuSvc(db),
	}
}

// @Tags 租户菜单
// @Summary 当前租户菜单树
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.MenuTreeResp}
// @Router /v1/tenant/menus/tree [get]
func (ctr *tenantMenuCtr) Tree(ctx *gin.Context) {
	req := dtotenant.MenuTreeReq{Operator: identity.OperatorFromGin(ctx)}
	res, err := ctr.tenantMenuSvc.Tree(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
