package model

import (
	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameConsoleBinding = "iam_console_binding"

// ConsoleBindingKind 控制台 binding 类型。一个 Host 可有多行（host + kind），表达同源双控制台。
type ConsoleBindingKind string

const (
	ConsoleBindingKindPlatform ConsoleBindingKind = "platform" // 平台控制台（平台 Host）
	ConsoleBindingKindTenant   ConsoleBindingKind = "tenant"   // D 租户控制台（D Host）
	ConsoleBindingKindCentral  ConsoleBindingKind = "central"  // 中央/tenantadmin（P1 现状）
)

// ConsoleBindingStatus binding 状态：draft（建租中，不对外）/ enable（可用）/ disable（停用，可恢复）。
type ConsoleBindingStatus string

const (
	ConsoleBindingStatusDraft   ConsoleBindingStatus = "draft"
	ConsoleBindingStatusEnable  ConsoleBindingStatus = "enable"
	ConsoleBindingStatusDisable ConsoleBindingStatus = "disable"
)

// ConsoleBindingEntity Host → 租户/issuer/client/audience/redirect_uri 的路由事实源。
// /config.json、/v1/auth/*、/v1/tenant/*、back-channel 都读同一绑定。
//
// 作用域（关键）：保留 tenant_id 但**不登记** tenantScopeSkipTables；
// 读取用 CrossTenantContext，写入用 ExplicitTenantContext(binding.tenant_id)。
// 见 docs/design/p0-host-binding-contract.md §4 与 tenant-custom-domain-redesign.md §4.6。
type ConsoleBindingEntity struct {
	gormdao.BaseEntity
	Host string `gorm:"column:host;type:varchar(256);not null;default:'';comment:归一化Host"`
	// TenantID 该 Host 的真实租户 id；平台行=平台 seed 租户 id（不用字面量 "platform"）。
	TenantID    string               `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:真实租户id(平台行=平台seed租户id)"`
	Issuer      string               `gorm:"column:issuer;type:varchar(256);not null;default:'';comment:该Host的issuer"`
	ClientID    string               `gorm:"column:client_id;type:varchar(64);not null;default:'';comment:控制台client_id"`
	Audience    string               `gorm:"column:audience;type:varchar(64);not null;default:'';comment:业务API audience"`
	RedirectURI string               `gorm:"column:redirect_uri;type:varchar(512);not null;default:'';comment:控制台回调"`
	Kind        ConsoleBindingKind   `gorm:"column:kind;type:varchar(16);not null;default:'tenant';comment:类型: platform/tenant/central"`
	Status      ConsoleBindingStatus `gorm:"column:status;type:varchar(16);not null;default:'draft';comment:状态: draft/enable/disable"`
}

func (ConsoleBindingEntity) TableName() string {
	return TableNameConsoleBinding
}

type ConsoleBindingEntityList []ConsoleBindingEntity

func (l ConsoleBindingEntityList) ToMap() map[string]ConsoleBindingEntity {
	m := make(map[string]ConsoleBindingEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
