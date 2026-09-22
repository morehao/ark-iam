package svcinstall

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/auth/internal/dto/dtoinstall"
	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/seed"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

const (
	// EnvBootstrapToken 是初始化接口的一次性共享令牌所在的环境变量名。
	//
	// 走环境变量而不是配置文件的理由：配置文件通常随镜像/仓库分发，而该令牌是
	// "部署期临时口令"，更应当由编排层的 secret 注入（容器 secret / k8s Secret）。
	EnvBootstrapToken = "BOOTSTRAP_TOKEN"
	// HeaderBootstrapToken 是携带令牌的请求头。
	HeaderBootstrapToken = "X-Bootstrap-Token"
)

// InstallSvc 是初始化引导的应用服务：唯一职责是把"页面输入 + 部署配置"翻译成
// pkg/seed.Bootstrap 的定义，并把结果翻译成对外 DTO。
//
// 边界：本服务**不做**口令校验与令牌校验（前者在 service 入参校验，后者在 controller 入口）。
type InstallSvc interface {
	// Status 返回部署期的三个状态位与控制台入口（公开、只读）。
	Status(ctx *gin.Context) (*dtoinstall.InstallationStatusResp, error)
	// Initialize 执行首次初始化（唯一写入口）。
	Initialize(ctx *gin.Context, req *dtoinstall.InstallationInitializeReq) (*dtoinstall.InstallationInitializeResp, error)
}

type installSvc struct{}

var _ InstallSvc = (*installSvc)(nil)

func NewInstallSvc() InstallSvc {
	return &installSvc{}
}

// Status 汇总部署期状态。三个状态位相互独立，页面可据此给出不同的可执行提示。
func (svc *installSvc) Status(ctx *gin.Context) (*dtoinstall.InstallationStatusResp, error) {
	// 初始化天然是跨租户操作（此刻库里还没有任何租户）。fail-closed 下不允许用
	// "ctx 碰巧没有作用域"表达跨租户，因此这里**显式**声明全部租户作用域。
	scopeCtx := dbclient.CrossTenantContext(ctx)
	db := dbclient.IamDB(scopeCtx)
	initialized, err := seed.IsInitialized(scopeCtx, db)
	if err != nil {
		glog.Errorf(ctx, "[svcinstall.Status] IsInitialized fail, err:%v", err)
		return nil, err
	}
	schemaReady, err := seed.SchemaReady(scopeCtx, db)
	if err != nil {
		glog.Errorf(ctx, "[svcinstall.Status] SchemaReady fail, err:%v", err)
		return nil, err
	}
	return &dtoinstall.InstallationStatusResp{
		Initialized:   initialized,
		TokenRequired: tokenConfigured(),
		SchemaReady:   schemaReady,
		Consoles:      consolesFromConfigForStatus(),
	}, nil
}

// Initialize 执行 L1 首次引导并回传变更明细。
//
// 关键点：**单次事务**。页面上的三步是 UI 分步，后端只有这一个写入口——
// 分步提交会让"第一步成功、第二步失败"留下一个没有管理员的租户，且没有任何恢复路径。
func (svc *installSvc) Initialize(ctx *gin.Context, req *dtoinstall.InstallationInitializeReq) (*dtoinstall.InstallationInitializeResp, error) {
	def, err := buildDefinition(req)
	if err != nil {
		return nil, err
	}

	// 同 Status：引导是跨租户操作，显式声明全部租户作用域后再交给 pkg/seed。
	scopeCtx := dbclient.CrossTenantContext(ctx)
	rep, status, err := seed.Bootstrap(scopeCtx, dbclient.IamDB(scopeCtx), def)
	if err != nil {
		glog.Errorf(ctx, "[svcinstall.Initialize] bootstrap fail, tenantName:%s, adminUsername:%s, err:%v",
			req.TenantName, req.AdminUsername, err)
		return nil, err
	}
	if status == seed.StatusAlreadyInitialized {
		// 与 Bootstrap 的自锁判定同一事实源：不做进程内"是否已初始化过"的缓存，
		// 因此重启、换副本、多副本并发都由库内事实决定。
		return nil, errAlreadyInitialized
	}
	glog.Infof(ctx, "[svcinstall.Initialize] bootstrap done, tenantID:%s, %s", rep.TenantID, rep.Summary())

	changes := make([]dtoinstall.InstallationChange, 0, len(rep.Changes))
	for _, c := range rep.Changes {
		changes = append(changes, dtoinstall.InstallationChange{Entity: c.Entity, Key: c.Key, Action: c.Action})
	}
	return &dtoinstall.InstallationInitializeResp{
		Report:        dtoinstall.InstallationReport{TenantID: rep.TenantID, Changes: changes},
		AdminUsername: req.AdminUsername,
		LoginURL:      config.Conf.OIDC.FrontendLoginURL,
		Consoles:      consolesFromConfigForStatus(),
	}, nil
}

// buildDefinition 把请求与部署配置翻译成 L1 定义。
//
// 口令只在这里转成摘要：明文既不入库也不进日志，本函数返回后明文即不再被使用。
func buildDefinition(req *dtoinstall.InstallationInitializeReq) (seed.Definition, error) {
	if err := validateInitializeReq(req); err != nil {
		return seed.Definition{}, err
	}
	hash, err := gcrypto.GeneratePasswordHash(req.AdminPassword)
	if err != nil {
		return seed.Definition{}, err
	}

	def := seed.Definition{
		TenantName:        req.TenantName,
		AdminUsername:     req.AdminUsername,
		AdminName:         req.AdminName,
		AdminEmail:        req.AdminEmail,
		AdminPhone:        req.AdminPhone,
		AdminPasswordHash: hash,
		Issuer:            req.Issuer,
		Consoles:          consolesFromConfig(),
	}
	// Issuer 为空时**不填**，交由 pkg/seed 的内置缺省（= 本部署 oidc.issuer 的历史默认值）；
	// 配置了 oidc.issuer 时以配置为准——它才是本部署真实对外发布的 issuer。
	if def.Issuer == "" {
		def.Issuer = config.Conf.OIDC.Issuer
	}
	return def, nil
}

// consolesFromConfig 把 oidc.consoles 配置翻译成 L1 的回调地址集合。
//
// 留空即零值，由 pkg/seed.withDefaults 回落到内置缺省（本地开发端口）——默认值只允许有一份，
// 因此这里刻意**不做**任何补全。
func consolesFromConfig() seed.ConsolesConfig {
	c := config.Conf.OIDC.Consoles
	return seed.ConsolesConfig{
		PlatformAdminWeb: seed.ConsoleRedirects{
			RedirectURIs:           c.PlatformAdminWeb.RedirectURIs,
			PostLogoutRedirectURIs: c.PlatformAdminWeb.PostLogoutRedirectURIs,
			BackChannelLogoutURI:   c.PlatformAdminWeb.BackChannelLogoutURI,
		},
		TenantAdminWeb: seed.ConsoleRedirects{
			RedirectURIs:           c.TenantAdminWeb.RedirectURIs,
			PostLogoutRedirectURIs: c.TenantAdminWeb.PostLogoutRedirectURIs,
			BackChannelLogoutURI:   c.TenantAdminWeb.BackChannelLogoutURI,
		},
	}
}

// consolesFromConfigForStatus 把回调地址归一为"控制台登录入口"，供页面展示。
//
// 回显的是回调地址的 origin + /login 而不是回调地址本身：前者是运维真正要打开的页面，
// 后者是 OAuth 内部跳转地址，直接给运维看只会造成困惑（且回调地址通常不是可浏览页面）。
func consolesFromConfigForStatus() dtoinstall.InstallationConsoles {
	c := consolesFromConfig()
	return dtoinstall.InstallationConsoles{
		PlatformAdminWeb: consoleEntry(c.PlatformAdminWeb),
		TenantAdminWeb:   consoleEntry(c.TenantAdminWeb),
	}
}

// consoleEntry 取控制台登出回跳地址（= 登录页）作为入口；缺失时回落到回调地址的 origin。
func consoleEntry(c seed.ConsoleRedirects) string {
	if len(c.PostLogoutRedirectURIs) > 0 {
		return c.PostLogoutRedirectURIs[0]
	}
	if len(c.RedirectURIs) > 0 {
		if u, err := url.Parse(c.RedirectURIs[0]); err == nil && u.Scheme != "" && u.Host != "" {
			return u.Scheme + "://" + u.Host + "/login"
		}
		return c.RedirectURIs[0]
	}
	return ""
}

// validateInitializeReq 校验页面输入。
//
// 与后端其它入口保持同一口径：username ≤128、email/phone 至少一项（UserContactRequiredError）、
// 口令走 credential.ValidateStrength。这些校验**不能只在前端做**——/install 是未认证写面，
// 前端校验只是体验优化。
func validateInitializeReq(req *dtoinstall.InstallationInitializeReq) error {
	if strings.TrimSpace(req.AdminUsername) == "" || len([]rune(req.AdminUsername)) > 128 {
		return errBadRequest
	}
	if strings.TrimSpace(req.AdminEmail) == "" && strings.TrimSpace(req.AdminPhone) == "" {
		return errContactRequired
	}
	if err := credential.ValidateStrength(req.AdminPassword); err != nil {
		return errPasswordWeak
	}
	return nil
}

// LogSafeReq 返回可安全写入日志的请求摘要：**剔除口令**。
//
// 单独提供而不是直接 gutil.ToJsonString(req)，是为了让"口令不入日志"这件事只有一处实现，
// 且调用方无法"顺手"把整个 req 打进日志。
func LogSafeReq(req *dtoinstall.InstallationInitializeReq) string {
	safe := *req
	safe.AdminPassword = ""
	return gutil.ToJsonString(&safe)
}
