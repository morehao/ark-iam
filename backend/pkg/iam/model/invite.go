package model

import (
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameInvite = "tenant_invite"

// InviteStatus 邀请单状态。
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"   // 待使用
	InviteStatusAccepted InviteStatus = "accepted"  // 已使用
	InviteStatusRevoked  InviteStatus = "revoked"   // 已撤销
	InviteStatusExpired  InviteStatus = "expired"   // 已过期
)

// InviteEntity 加入租户的邀请单：租户 owner/管理员生成，凭证持有者凭 inviteCode 加入该租户。
// 用户侧能否自助加入的开关由租户策略 AllowJoinByInvite 控制（见 Application.TenantPolicy）。
type InviteEntity struct {
	gormdao.BaseEntity
	TenantID  string       `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:归属租户"`
	Code      string       `gorm:"column:code;type:varchar(64);not null;default:'';comment:邀请码"`
	Status    InviteStatus `gorm:"column:status;type:varchar(32);not null;default:'pending';comment:状态"`
	ExpiresAt *time.Time   `gorm:"column:expires_at;comment:过期时间,空为永久"`
	CreatedBy string       `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy string       `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy string       `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (InviteEntity) TableName() string {
	return TableNameInvite
}

type InviteEntityList []InviteEntity
