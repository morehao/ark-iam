package ctrinstall

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/auth/internal/dto/dtoinstall"
	"github.com/morehao/ark-iam/auth/internal/service/svcinstall"
	"github.com/morehao/ark-iam/pkg/audit"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
)

// InstallCtr 是初始化引导入口的控制器。
//
// 与其它控制器最大的不同：这里的失败**必须使用真实 HTTP 状态码**（gincontext.FailWithStatus），
// 而不是项目惯用的 200 + 业务码。调用方是浏览器页面与自动化部署脚本——
// 部署脚本按 409/401/503 决定"该做什么"，网关与编排层也需要按状态码做重试/告警。
type InstallCtr struct {
	installSvc svcinstall.InstallSvc
}

func NewInstallCtr() *InstallCtr {
	return &InstallCtr{installSvc: svcinstall.NewInstallSvc()}
}

// Status 返回部署期状态（公开、只读、不含敏感信息）。
//
// @Tags 系统初始化
// @Summary 查询初始化状态
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtoinstall.InstallationStatusResp}
// @Router /install/status [get]
func (ctr *InstallCtr) Status(ctx *gin.Context) {
	res, err := ctr.installSvc.Status(ctx)
	if err != nil {
		// 状态查询失败只能是系统错误（表未建/DB 不可用）：用 503 表达"暂时给不出状态"，
		// 而不是 200 + 业务码——前端与探针需要据此区分"没初始化"与"服务不可用"。
		gincontext.FailWithStatus(ctx, http.StatusServiceUnavailable, err)
		return
	}
	gincontext.Success(ctx, res)
}

// Initialize 执行首次初始化（唯一写入口，需 X-Bootstrap-Token）。
//
// 失败路径同样写审计：/install 是未认证写面，失败记录是发现扫描行为的唯一线索。
//
// @Tags 系统初始化
// @Summary 执行系统首次初始化
// @accept application/json
// @Produce application/json
// @Param req body dtoinstall.InstallationInitializeReq true "初始化参数"
// @Success 200 {object} gincontext.DtoRender{data=dtoinstall.InstallationInitializeResp}
// @Router /install/initialize [post]
func (ctr *InstallCtr) Initialize(ctx *gin.Context) {
	// 令牌校验**先于任何参数处理**：它是"准入"而非业务规则。
	//
	// 这个顺序是安全属性，不是风格偏好：绑定失败会返回 400/107005，令牌失败返回 401/107001，
	// 若把绑定放在前面，一个**未持有令牌**的调用方就能靠"400 还是 401"判断自己的请求参数
	// 是否合法——把参数校验变成了无需准入即可使用的探测面。原先的注释已写明这条意图，
	// 但代码把绑定放在了令牌之前，实现与契约相反。
	if err := svcinstall.CheckBootstrapToken(ctx.GetHeader(svcinstall.HeaderBootstrapToken)); err != nil {
		status, _ := svcinstall.HTTPStatusOf(err)
		// 未通过准入的请求**不记录其参数**：此时 req 尚未绑定（也不该绑定），
		// 日志只留状态码与错误类型。
		ctr.fail(ctx, nil, status, err, err)
		return
	}

	var req dtoinstall.InstallationInitializeReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// 绑定失败的错误文案由 gin 生成，可能含请求体片段，因此只进日志、不进审计。
		ctr.fail(ctx, &req, http.StatusBadRequest, code.GetError(code.InstallationBadRequestError), err)
		return
	}

	res, err := ctr.installSvc.Initialize(ctx, &req)
	if err != nil {
		status, _ := svcinstall.HTTPStatusOf(err)
		ctr.fail(ctx, &req, status, err, err)
		return
	}

	// 成功审计：初始化是本系统唯一一次"无鉴权主体创建平台管理员"的操作，
	// 审计是事后唯一能回答"谁、何时、从哪个 IP 建了这套系统"的记录。
	ctr.writeAudit(ctx, res.Report.TenantID, model.AuditResultSuccess,
		fmt.Sprintf("初始化成功，创建 %d 条内置数据", len(res.Report.Changes)))
	gincontext.Success(ctx, res)
}

// fail 统一失败出口：写审计 + 按业务码映射的状态码返回。
//
// renderErr 决定响应体（稳定的业务码 + 文案），logErr 决定日志内容。分开是为了让
// "响应给客户端什么"与"日志记什么"各自取最合适的东西，而不是被迫用同一个值。
//
// 日志用 svcinstall.LogSafeReq（**剔除口令**）——/install 的入参里有明文口令，
// 直接 ToJsonString(req) 会把口令写进日志，这是本接口最容易犯、后果最严重的错误。
func (ctr *InstallCtr) fail(ctx *gin.Context, req *dtoinstall.InstallationInitializeReq, status int, renderErr, logErr error) {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	glog.Errorf(ctx, "[ctrinstall.Initialize] fail, status:%d, req:%s, err:%v", status, svcinstall.LogSafeReq(req), logErr)
	// 审计 detail 只用状态码 + 业务码，**不带 err 文案**：绑定失败的错误来自 gin，
	// 可能包含请求体片段，而请求体里有明文口令。审计是长期留存、被更多人看到的记录。
	ctr.writeAudit(ctx, "", model.AuditResultFailure,
		fmt.Sprintf("初始化失败（HTTP %d, code=%d）", status, svcinstall.BusinessCodeOf(renderErr)))
	gincontext.FailWithStatus(ctx, status, renderErr)
}

// writeAudit 写初始化审计（成功与失败共用）。
func (ctr *InstallCtr) writeAudit(ctx *gin.Context, tenantID string, result model.AuditResult, detail string) {
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     model.AuditActionInstallationInitialize,
		TenantID:   tenantID,
		TargetType: model.AuditTargetTypeInstallation,
		TargetID:   tenantID,
		Result:     result,
		Detail:     detail,
	})
}
