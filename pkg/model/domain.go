package model

import (
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameDomain = "iam_domain"

// DomainStatus 域名生命周期状态（单一状态机，替代原 verification_status）。
// 状态流转见 docs/design/p0-tenant-lifecycle-contract.md §4 与
// tenant-custom-domain-redesign.md §3.2：
//
//	draft → verified → tls_ready → active → disabled → retired → released
//	                                         active/disabled → retired → released
//	任意 draft/verified/tls_ready → released（取消未上线域名）
//
// released 是终态，host 可重新登记；retired 仍占用 host。
type DomainStatus string

const (
	DomainStatusDraft    DomainStatus = "draft"     // 已占 host，未验证；不建 TLS，不解析/不签发
	DomainStatusVerified DomainStatus = "verified"  // 域控证明通过（verified_at）；不解析/不签发
	DomainStatusTLSReady DomainStatus = "tls_ready" // 证书就绪（tls_ready_at）；不解析/不签发
	DomainStatusActive   DomainStatus = "active"    // 唯一可用态；primary 可签发，login/both alias 可 302
	DomainStatusDisabled DomainStatus = "disabled"  // 停止解析/签发，可回 active
	DomainStatusRetired  DomainStatus = "retired"   // 墓碑；host 仍占用；协议请求 400/404
	DomainStatusReleased DomainStatus = "released"  // 隔离期满/审批后软删；host 可重新登记（终态）
)

// DomainRole 域名角色。
type DomainRole string

const (
	DomainRolePrimary DomainRole = "primary" // 主域：每 active D 租户恰一个 active primary；作为 issuer 来源
	DomainRoleAlias   DomainRole = "alias"   // 别名：302 到 primary，不签 token
	// DomainRoleBranded 仅 P1/P2 预留，P0 不消费。
	DomainRoleBranded DomainRole = "branded"
)

// DomainUsage 域名用途。
type DomainUsage string

const (
	DomainUsageLogin DomainUsage = "login" // 登录宿主（可作为 issuer / 302 target）
	DomainUsageEmail DomainUsage = "email" // 邮箱域发现（P0 只登记，不路由/不签发）
	DomainUsageBoth  DomainUsage = "both"  // 登录 + 邮箱
)

// DomainVerificationMethod 域控证明方式。
type DomainVerificationMethod string

const (
	DomainVerificationManual DomainVerificationMethod = "manual"  // 人工留证
	DomainVerificationDNSTXT DomainVerificationMethod = "dns_txt" // DNS TXT 挑战
)

// DomainEntity 域名行。iam_domain 的 primary 行是 host 与 issuer 的事实源；
// iam_tenant.issuer 是其同事务派生缓存。
//
// 设计依据：docs/design/tenant-custom-domain-redesign.md §4.2、§4.4（索引）、
// p0-tenant-lifecycle-contract.md §4（状态机与不变量）。
type DomainEntity struct {
	gormdao.BaseEntity
	TenantID string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	Domain   string `gorm:"column:domain;type:varchar(256);not null;default:'';comment:归一化host(小写/punycode/无scheme/端口/路径)"`
	// DomainRole 角色：primary / alias（branded 仅 P1/P2 预留）。
	DomainRole DomainRole `gorm:"column:domain_role;type:varchar(16);not null;default:'alias';comment:域名角色: primary-主域, alias-别名, branded-品牌(P1/P2预留)"`
	// DomainUsage 用途：login / email / both。primary 必须是 login/both。
	DomainUsage DomainUsage `gorm:"column:domain_usage;type:varchar(16);not null;default:'login';comment:域名用途: login-登录, email-邮箱, both-两者"`
	// TargetDomainID alias/email 的目标 primary 行 id（同租户）；primary 自身为空；D alias 必填。
	TargetDomainID string `gorm:"column:target_domain_id;type:varchar(36);not null;default:'';comment:alias/email目标primary行id(同租户);primary为空;D alias必填"`
	// Status 单一状态机（替代原 verification_status）。
	Status             DomainStatus             `gorm:"column:status;type:varchar(16);not null;default:'draft';comment:域名状态: draft/verified/tls_ready/active/disabled/retired/released"`
	VerificationMethod DomainVerificationMethod `gorm:"column:verification_method;type:varchar(16);not null;default:'manual';comment:域控证明方式: manual/dns_txt"`
	VerificationToken  string                   `gorm:"column:verification_token;type:varchar(64);not null;default:'';comment:DNS TXT挑战值(P0可空)"`
	VerifiedAt         *time.Time               `gorm:"column:verified_at;comment:域控证明时间"`
	TLSReadyAt         *time.Time               `gorm:"column:tls_ready_at;comment:证书就绪时间"`
	RetiredAt          *time.Time               `gorm:"column:retired_at;comment:退役时间"`
	ReleasedAt         *time.Time               `gorm:"column:released_at;comment:释放时间(released为终态,host可重新登记)"`
	// ReleaseApproval 释放审批单；仅正式 release（退役后 90 天+审批）填写，取消（未上线）不填。
	ReleaseApproval string `gorm:"column:release_approval;type:varchar(64);not null;default:'';comment:释放审批单(仅正式release填)"`
	RetireReason    string `gorm:"column:retire_reason;type:varchar(256);not null;default:'';comment:退役原因/审批单"`
	CreatedBy       string `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy       string `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy       string `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (DomainEntity) TableName() string {
	return TableNameDomain
}

type DomainEntityList []DomainEntity

func (l DomainEntityList) ToMap() map[string]DomainEntity {
	m := make(map[string]DomainEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
