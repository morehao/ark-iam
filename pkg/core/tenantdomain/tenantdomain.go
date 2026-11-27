// Package tenantdomain 承载 P0 自定义域名/多 issuer 的域名状态机、不变量与事务骨架。
//
// 边界（DG-CORE-1）：本包只被应用层 service（svcdomain/svctenant）调用编排；
// **不得**反向被 pkg/core/tenant import，也不 import gin / pkg/identity（DG-CORE-3），
// 只接收 context.Context + *gorm.DB 事务。领域哨兵错误由应用层映射为功能级错误码
// （101207–101213，见 docs/design/tenant-custom-domain-redesign.md §5.3），本包不依赖 pkg/code。
//
// 设计依据：
//   - docs/design/p0-tenant-lifecycle-contract.md §3/§4（生命周期、状态机、不变量）
//   - docs/design/tenant-custom-domain-redesign.md §3.2/§4.3/§4.5（状态机、不变量、事务与作用域）
package tenantdomain

import (
	"errors"
	"time"

	"github.com/morehao/ark-iam/pkg/model"
)

// 领域哨兵错误（应用层 svcdomain/svctenant 统一映射为功能级错误码）。
var (
	// ErrDomainNotExist 域名行不存在 / 不属于目标租户。
	ErrDomainNotExist = errors.New("tenantdomain: domain not exist")
	// ErrTenantNotExist 租户不存在。
	ErrTenantNotExist = errors.New("tenantdomain: tenant not exist")
	// ErrInvalidTransition 当前状态不允许该动作（from→to 非法）。
	ErrInvalidTransition = errors.New("tenantdomain: invalid status transition")
	// ErrConflict 并发冲突：条件更新命中 0 行（前置状态已被他人改变）。
	ErrConflict = errors.New("tenantdomain: concurrent conflict")
	// ErrAlreadyActivePrimary 已存在另一个 active primary（激活前置校验）。
	ErrAlreadyActivePrimary = errors.New("tenantdomain: tenant already has an active primary")
	// ErrTargetNotActive alias 激活/enable 时 target 不满足 active。
	ErrTargetNotActive = errors.New("tenantdomain: alias target not active")
	// ErrTargetMismatch target 不是同租户 primary，或指向自身。
	ErrTargetMismatch = errors.New("tenantdomain: alias target mismatch")
	// ErrPrimaryUsageInvalid primary 的 usage 必须是 login/both。
	ErrPrimaryUsageInvalid = errors.New("tenantdomain: primary usage must be login/both")
	// ErrReferencedByAlias 删除/释放时有未 released 的 alias 引用。
	ErrReferencedByAlias = errors.New("tenantdomain: referenced by alias")
	// ErrReleaseNotDue 未满隔离期（90 天）或缺少审批单。
	ErrReleaseNotDue = errors.New("tenantdomain: release not due or missing approval")
	// ErrTenantStatusInvalid 租户状态不允许该动作。
	ErrTenantStatusInvalid = errors.New("tenantdomain: tenant status invalid")
)

// ReleaseIsolationPeriod 域名退役到可 release 的最小隔离期（90 天，见 §3 退役规则）。
const ReleaseIsolationPeriod = 90 * 24 * time.Hour

// serviceLoginUsage 返回该域名是否可做登录宿主（login/both）。email-only 不可作 issuer/302 target。
func serviceLoginUsage(u model.DomainUsage) bool {
	return u == model.DomainUsageLogin || u == model.DomainUsageBoth
}

// HostCacheInvalidator 提交成功后删除 host 路由缓存的钩子（W2 提供具体 Redis 实现）。
// 为 nil 时跳过；W1 不直接依赖 Redis/缓存实现，保持 core 干净（DG-CORE-3）。
type HostCacheInvalidator func(host string)

// 编译期确认本包只依赖 model（不引 dao/dbclient/gin，事务 SQL 在 tx_ops.go）。
var _ = model.DomainStatusDraft
