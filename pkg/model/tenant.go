package model

import (
	"fmt"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameTenant = "iam_tenant"

type TenantType string

const (
	TenantTypeCustomer TenantType = "customer"
	TenantTypePlatform TenantType = "platform"
)

// TenantStatus 租户生命周期状态（字符串枚举）。非 active 的租户不允许其成员登录，也不允许签发/轮换令牌。
//   - provisioning：D 建租中（尚未有 active primary），不可签发；
//   - active：可服务、可签发；active D 恰一个 active primary；
//   - suspended：offboard 后；primary disabled，issuer 已清空，不可签发。
//
// 依据 docs/design/p0-tenant-lifecycle-contract.md §1 / tenant-custom-domain-redesign.md §3.2。
type TenantStatus string

const (
	TenantStatusProvisioning TenantStatus = "provisioning" // D 建租中，不可签发
	TenantStatusActive       TenantStatus = "active"       // 正常，可签发
	TenantStatusSuspended    TenantStatus = "suspended"    // 已挂起（offboard 后）
)

// IsActive 租户是否处于可服务状态。仅 active 放行，其余取值（含空值）一律按不可用处理，
// 避免将来新增状态时被默认放行。
func (t *TenantEntity) IsActive() bool {
	return t != nil && t.Status == TenantStatusActive
}

// NormalizeTenantStatus 白名单校验租户状态（"update 场景保留原值"口径）。
// 仅接受已定义的枚举常量，非法值（含空值）回退为 active。
//
// 注意：这是历史语义，供 update/未显式指定时的默认填充使用。区分 create 场景的
// 「空=未指定」（普通 create→active、dedicated create→provisioning）请用 ParseTenantStatus。
func NormalizeTenantStatus(status TenantStatus) TenantStatus {
	switch status {
	case TenantStatusProvisioning, TenantStatusActive, TenantStatusSuspended:
		return status
	default:
		return TenantStatusActive
	}
}

// ParseTenantStatus 解析外部传入的租户状态，返回 (值, 是否为空, error)。
//   - 空字符串 → ("", true, nil)：调用方按场景决定默认值
//     （普通 create→active、dedicated create→provisioning、update→保留原值）；
//   - 非空且合法 → (值, false, nil)；
//   - 非空非法 → ("", false, 错误)：映射 101212，**不得回退 active**。
//
// 依据 p0-tenant-lifecycle-contract.md §1「状态解析」。
func ParseTenantStatus(raw string) (TenantStatus, bool, error) {
	if raw == "" {
		return "", true, nil
	}
	switch TenantStatus(raw) {
	case TenantStatusProvisioning, TenantStatusActive, TenantStatusSuspended:
		return TenantStatus(raw), false, nil
	default:
		return "", false, fmt.Errorf("invalid tenant status: %q", raw)
	}
}

// TenantIsolationMode 租户隔离模式（D/S 分级）。P0 只写 dedicated 新租户，shared 保留既有行为。
type TenantIsolationMode string

const (
	TenantIsolationDedicated TenantIsolationMode = "dedicated" // 专属：自定义域 + 独立 issuer
	TenantIsolationShared    TenantIsolationMode = "shared"    // 共享：中央 issuer
)

// TenantEntity 租户。
//
// code 的唯一性口径与其余软删除表一致：**部分唯一索引**（WHERE deleted_at IS NULL），
// 使「软删一个租户 → 用同一 code 再建」不会被历史行撞键。
// GORM AutoMigrate 不支持部分索引，事实源是 automigrate.go 的 partialUniqueIndexes（幂等 DDL）。
// 注意：本列原来的普通 `uniqueIndex` tag 已删除——AutoMigrate 只增不删，
// 存量库上的旧唯一索引需要**删库重建**才会消失（见 AGENTS.md「Schema 变更约定」）。
type TenantEntity struct {
	gormdao.BaseEntity
	Code      string       `gorm:"column:code;type:varchar(64);not null;default:'';comment:租户编码"`
	CreatedBy string       `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	DbUser    string       `gorm:"column:db_user;type:varchar(64);not null;default:'';comment:数据库用户"`
	DeletedBy string       `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
	Status    TenantStatus `gorm:"column:status;type:varchar(32);not null;default:'active';comment:租户状态: provisioning-建租中, active-正常, suspended-已挂起"`
	// IsolationMode 隔离模式：dedicated（专属，自定义域 + 独立 issuer）/ shared（共享，中央 issuer）。
	// P0 只写 dedicated 新租户；create-only，通用控制台不允许 PUT 改（见 p0-tenant-lifecycle-contract §1）。
	IsolationMode TenantIsolationMode `gorm:"column:isolation_mode;type:varchar(16);not null;default:'shared';comment:隔离模式: dedicated-专属, shared-共享"`
	// Issuer D 租户的派生缓存 issuer（https://<primary-host><basePath>/oidc）；事实源是 iam_domain 的
	// active primary 行，本列是同事务派生缓存，S/平台为空（用平台 issuer）。
	// 唯一性由 partial unique 索引 iam_idx_tenant_issuer_dedicated_unique 兜底。
	Issuer string `gorm:"column:issuer;type:varchar(256);not null;default:'';comment:派生缓存issuer(事实源为active primary域)"`
	// DomainEpoch 域名/issuer 变更版本号；host 缓存和签发门禁读它，域名写操作同事务 +1。
	DomainEpoch int64      `gorm:"column:domain_epoch;type:bigint;not null;default:0;comment:域名/issuer变更版本号"`
	Name        string     `gorm:"column:name;type:varchar(128);not null;default:'';comment:租户名称"`
	Type        TenantType `gorm:"column:type;type:varchar(32);not null;default:'customer';comment:租户类型: customer-客户租户, platform-平台租户"`
	Tag         string     `gorm:"column:tag;type:varchar(64);not null;default:'';comment:标签"`
	UpdatedBy   string     `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
}

func (TenantEntity) TableName() string {
	return TableNameTenant
}

type TenantEntityList []TenantEntity

func (l TenantEntityList) ToMap() map[string]TenantEntity {
	m := make(map[string]TenantEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
