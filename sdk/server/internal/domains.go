// Package server 是 sdk 的门面 / 编排层。
//
// 设计（见 .dsh/docs/plans/2026-11-10-iam-into-sdk-server-plan.md）：
//
//   - sdk/server 只做编排：按 Config.Domains 选择业务域，组装 KeySource（同进程 auth
//     产出 / 独立进程远程 JWKS 双路径），转发调用 apps/{auth,platformadmin,tenantadmin}
//     各域实现库；**不含业务实现**。
//   - 依赖方向（分层倒置）：进程入口 → sdk/server → apps 各域库 → pkg。
//     sdk 调用 apps；apps 不 import sdk（唯一例外：apps/rpapi → sdk/rp 叶子依赖）。
//   - 提供 New(Domains)/Mount/RunMigrations/MountConfig/KeySource/DBGetter/Redis/Close。
package internal

// Domain 是 IAM 业务域的枚举。使用者通过 Config.Domains 选择挂哪些域，
// 从而实现「可嵌入 + 可组合」：既支持形态 1 的全量挂载，也支持形态 2 的最小集。
//
// 设计依据见 .dsh/docs/specs/2026-11-06-embeddable-iam-design.md §4.3。
type Domain string

const (
	// —— 恒含域（任何形态都必须开启）——
	DomainCoreAuth Domain = "core-auth" // 登录/登出/会话/me/userinfo
	DomainInstall  Domain = "install"   // /install 初始化引导（空库自举必需）

	// —— 简易用户体系核心域（形态 2 默认集）——
	DomainPerson     Domain = "person"
	DomainUser       Domain = "user"
	DomainDepartment Domain = "department"
	DomainRole       Domain = "role"
	DomainMenu       Domain = "menu"

	// —— 可选企业级域（形态 1 全量 / 按需开启）——
	DomainTenant      Domain = "tenant"
	DomainApplication Domain = "application"
	DomainOAuthClient Domain = "oauth-client"
	DomainConnector   Domain = "connector"
	DomainDomain      Domain = "domain"
	DomainAudit       Domain = "audit"
	DomainAPIKey      Domain = "api-key"
	DomainMachineUser Domain = "machine-user"
	DomainInvite      Domain = "invite"
	DomainTenantApp   Domain = "tenant-app"
	DomainSelfService Domain = "self-service" // 自助注册/加入租户
)

// DefaultConsoleDomains 是「部门/人员/角色/菜单」简易用户体系的最小集（形态 2 推荐）。
// 含登录/会话 + 安装引导 + 人员 + 用户 + 部门 + 角色 + 菜单。
var DefaultConsoleDomains = []Domain{
	DomainCoreAuth,
	DomainInstall,
	DomainPerson,
	DomainUser,
	DomainDepartment,
	DomainRole,
	DomainMenu,
}

// allDomains 是全部域的集合（形态 1 / gateway 全量挂载用）。
var allDomains = []Domain{
	DomainCoreAuth, DomainInstall,
	DomainPerson, DomainUser, DomainDepartment, DomainRole, DomainMenu,
	DomainTenant, DomainApplication, DomainOAuthClient, DomainConnector,
	DomainDomain, DomainAudit, DomainAPIKey, DomainMachineUser,
	DomainInvite, DomainTenantApp, DomainSelfService,
}

// resolveDomains 归一化使用者传入的域集合：空 → 全量；否则返回去重后的副本。
func resolveDomains(in []Domain) []Domain {
	if len(in) == 0 {
		out := make([]Domain, len(allDomains))
		copy(out, allDomains)
		return out
	}
	seen := make(map[Domain]struct{}, len(in))
	out := make([]Domain, 0, len(in))
	for _, d := range in {
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

// ErrDomainDependency 表示选定域缺少必要依赖域。
type ErrDomainDependency struct {
	Domain Domain
	Need   Domain
}

func (e *ErrDomainDependency) Error() string {
	return "sdk/server: domain " + string(e.Domain) + " requires domain " + string(e.Need)
}

// domainDeps 声明域间硬依赖（选 A 必须同时选 B）。
// 说明：tenantadmin 的人员/角色等页依赖「菜单树」读取（DomainMenu）；platform 的角色授权
// 依赖 menu。这里声明最小必要依赖，避免运行期 404。
var domainDeps = map[Domain][]Domain{
	// 部门/人员/角色在 tenantadmin，菜单写/读树在 platform+tenant 的 menu 域；保留 role→menu。
	DomainRole: {DomainMenu},
}

// validateDomainDeps 校验域依赖闭合。
func validateDomainDeps(domains []Domain) error {
	set := make(map[Domain]bool, len(domains))
	for _, d := range domains {
		set[d] = true
	}
	for _, d := range domains {
		for _, need := range domainDeps[d] {
			if !set[need] {
				return &ErrDomainDependency{Domain: d, Need: need}
			}
		}
	}
	return nil
}
