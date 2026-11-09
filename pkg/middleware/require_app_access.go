package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/application"
	"github.com/morehao/ark-iam/pkg/identity"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
)

// RequireAppAccess 控制台请求期应用准入闸口（L1 准入）。
//
// 回答的问题只有一个：**该成员能不能用这个应用**（能不能进这个控制台）。
// 它不回答"在这个租户里能不能做管理"——那是 L2（服务层的 requireSystemAdmin），两层不得互相替代。
//
// 判定链（与登录期门禁同源，判定式唯一实现在 pkg/core/application.Usable）：
//
//	令牌 client_id → client_id 所属应用（AppIDByClientID）
//	              → CheckUsable(令牌租户, 调用方成员, 应用) ⟺ 应用启用 ∧ 租户订阅启用 ∧ 用户准入启用
//
// 用「令牌租户」而不是写死平台租户：准入是租户内的概念——t_platform 的成员对平台控制台应用有准入，
// 某业务租户的成员对该租户订阅的 tenant_admin 应用有准入。攻击者拿自建租户的令牌来查平台控制台应用时
// 必然查不到准入行，因此必然被拒。
//
// 处置口径：
//   - 缺参（client_id / 成员 / 租户任一为空）→ 视为无准入，拒绝（fail-closed）。不查库，
//     因为"身份不完整"不是"有身份"，且空租户会以 ErrEmptyScopeValue 变成系统错误、掩盖真实语义；
//   - 客户端不存在 / 未绑定应用（appID 为空）→ 拒绝（应用作用域不可判定即不放行）；
//   - 判定查询出错 → **如实上抛**（系统错误不伪装成"无权限"，见 AGENTS.md err 与边界分离）；
//   - 判定不通过 → 110016 AuthAppAccessDeniedError + Abort。
//
// 挂载位置必须在 OIDCAuth 之后：后者负责验签、写入租户作用域与成员身份（含 client_id），
// 本中间件只读这些已校验的取值。
//
// 回滚：摘掉挂载即可（无数据变更）。
func RequireAppAccess() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientID := identity.ClientIDFromContext(ctx)
		userID := gincontext.GetUserIDString(ctx)
		tenantID := gincontext.GetTenantIDString(ctx)
		if clientID == "" || userID == "" || tenantID == "" {
			glog.Warnf(ctx, "[RequireAppAccess] incomplete identity, deny, tenantID:%s, userID:%s, clientID:%s", tenantID, userID, clientID)
			denyAppAccess(ctx)
			return
		}

		appID, err := application.AppIDByClientID(ctx, clientID)
		if err != nil {
			glog.Errorf(ctx, "[RequireAppAccess] resolve app by clientID fail, err:%v, clientID:%s", err, clientID)
			gincontext.Fail(ctx, err)
			ctx.Abort()
			return
		}
		if appID == "" {
			glog.Warnf(ctx, "[RequireAppAccess] client not bound to any application, deny, clientID:%s, tenantID:%s, userID:%s", clientID, tenantID, userID)
			denyAppAccess(ctx)
			return
		}

		usable, err := application.CheckUsable(ctx, tenantID, userID, appID)
		if err != nil {
			glog.Errorf(ctx, "[RequireAppAccess] check app access fail, err:%v, tenantID:%s, userID:%s, appID:%s", err, tenantID, userID, appID)
			gincontext.Fail(ctx, err)
			ctx.Abort()
			return
		}
		if !usable {
			glog.Warnf(ctx, "[RequireAppAccess] app access denied, tenantID:%s, userID:%s, appID:%s, clientID:%s", tenantID, userID, appID, clientID)
			denyAppAccess(ctx)
			return
		}

		ctx.Next()
	}
}

// denyAppAccess 以既有的 110016 拒绝并终止链路（码值与文案保持不变，已对接前端）。
func denyAppAccess(ctx *gin.Context) {
	gincontext.Fail(ctx, code.GetError(code.AuthAppAccessDeniedError))
	ctx.Abort()
}
