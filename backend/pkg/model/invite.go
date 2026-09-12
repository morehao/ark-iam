package model

import (
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameInvite = "tenant_invite"

// InviteStatus 邀请单状态。
// 「已过期」不是存储态：由 expires_at 在读取时派生——若写成存储态，会引入
// 「已过期但定时任务还没跑到」的一致性窗口，反而更差。
// 判定位置见 docs/design/system-design.md §5.1（通道 B）。
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"  // 待使用
	InviteStatusAccepted InviteStatus = "accepted" // 已使用
	InviteStatusRevoked  InviteStatus = "revoked"  // 已撤销
)

// InviteEntity 加入租户的邀请单：租户 owner/管理员生成，凭证持有者凭 inviteCode 加入该租户。
//
// 能否加入由**两道门禁**按序判定（见 docs/design/system-design.md §5.1 通道 B）：
//  1. 应用级策略 application.AllowJoinByInvite——按调用方 access token 的 client_id 解析出应用
//     再读该开关；解析不出应用或字段未配置（NULL）一律拒绝（fail-closed），且该判定先于邀请解析；
//  2. 邀请单自身——存在、status=pending、未过期（ExpiresAt 为空表示永久）。
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
