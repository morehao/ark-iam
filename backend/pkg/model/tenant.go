package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameTenant = "tenant"

type TenantType string

const (
	TenantTypeCustomer TenantType = "customer"
	TenantTypePlatform TenantType = "platform"
)

// TenantStatus 租户生命周期状态（字符串枚举）。非 active 的租户不允许其成员登录，也不允许签发/轮换令牌。
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"    // 正常
	TenantStatusSuspended TenantStatus = "suspended" // 已挂起
)

// IsActive 租户是否处于可服务状态。仅 active 放行，其余取值（含空值）一律按不可用处理，
// 避免将来新增状态时被默认放行。
func (t *TenantEntity) IsActive() bool {
	return t != nil && t.Status == TenantStatusActive
}

// NormalizeTenantStatus 白名单校验租户状态：仅接受已定义的枚举常量，
// 非法值（含空值，例如前端未传该字段）回退为 active。
func NormalizeTenantStatus(status TenantStatus) TenantStatus {
	switch status {
	case TenantStatusActive, TenantStatusSuspended:
		return status
	default:
		return TenantStatusActive
	}
}

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
	Status    TenantStatus `gorm:"column:status;type:varchar(32);not null;default:'active';comment:租户状态: active-正常, suspended-已挂起"`
	Name      string       `gorm:"column:name;type:varchar(128);not null;default:'';comment:租户名称"`
	Type      TenantType   `gorm:"column:type;type:varchar(32);not null;default:'customer';comment:租户类型: customer-客户租户, platform-平台租户"`
	Tag       string       `gorm:"column:tag;type:varchar(64);not null;default:'';comment:标签"`
	UpdatedBy string       `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
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
