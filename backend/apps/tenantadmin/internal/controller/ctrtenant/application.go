package ctrtenant

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/ark-iam/tenantadmin/internal/service/svctenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// ApplicationCtr 租户应用视角接口：订阅应用清单 + 应用准入视角（谁能进这个应用）。
type ApplicationCtr interface {
	List(ctx *gin.Context)
	ListUsers(ctx *gin.Context)
}

type applicationCtr struct {
	appAccessSvc svctenant.ApplicationAccessSvc
}

var _ ApplicationCtr = (*applicationCtr)(nil)

func NewApplicationCtr() ApplicationCtr {
	return &applicationCtr{
		appAccessSvc: svctenant.NewApplicationAccessSvc(),
	}
}

// @Tags 租户应用
// @Summary 当前租户订阅的启用应用列表（角色归属 / 应用准入的应用选项，含系统内置应用）
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.TenantApplicationsResp}
// @Router /v1/tenant/applications [get]
func (ctr *applicationCtr) List(ctx *gin.Context) {
	res, err := ctr.appAccessSvc.ListTenantApplications(ctx.Request.Context())
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 租户应用
// @Summary 应用视角：被准入使用该应用的成员列表
// @accept application/json
// @Produce application/json
// @Param appID path string true "应用ID"
// @Success 200 {object} gincontext.DtoRender{data=dtotenant.ApplicationUsersListResp}
// @Router /v1/tenant/applications/{appID}/users [get]
func (ctr *applicationCtr) ListUsers(ctx *gin.Context) {
	var req dtotenant.ApplicationUsersListReq
	req.Operator = identity.OperatorFromGin(ctx)
	if err := gincontext.BindPathParams(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.appAccessSvc.ListApplicationUsers(ctx.Request.Context(), &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
