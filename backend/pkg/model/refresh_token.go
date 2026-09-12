package model

import (
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/datatypes"
)

const TableNameRefreshToken = "refresh_token"

type RefreshTokenEntity struct {
	gormdao.BaseEntity
	PersonID            string `gorm:"column:person_id;type:varchar(36);not null;default:'';comment:自然人ID"`
	TenantID            string `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	UserID              string `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:用户ID"`
	ApplicationClientID string `gorm:"column:application_client_id;type:varchar(36);not null;default:'';comment:OIDC客户端ID"`
	SessionID           string `gorm:"column:session_id;type:varchar(64);not null;default:'';comment:会话ID"`
	Token               string `gorm:"column:token;type:varchar(256);not null;default:'';comment:token哈希"`
	// Scopes 授权时授予的 scope 列表（JSON 数组）。刷新时必须还原原始 scope，
	// 否则刷新后 token 的 scope 会缩水（RFC 6749 §6）。可空：历史数据无该列。
	Scopes datatypes.JSON `gorm:"column:scopes;type:json;comment:授权scope"`
	// AMR 原始认证方法引用（JSON 数组），刷新与审计时还原。可空：历史数据无该列。
	AMR datatypes.JSON `gorm:"column:amr;type:json;comment:认证方法引用"`
	// AuthTime 原始认证时间，刷新 token 时还原到 auth_time 声明。
	AuthTime      *time.Time `gorm:"column:auth_time;comment:原始认证时间"`
	ClientType    string     `gorm:"column:client_type;type:varchar(32);not null;default:'';comment:客户端类型"`
	ClientIP      string     `gorm:"column:client_ip;type:varchar(64);not null;default:'';comment:客户端IP"`
	UserAgent     string     `gorm:"column:user_agent;type:varchar(512);not null;default:'';comment:用户代理信息"`
	ExpiredAt     *time.Time `gorm:"column:expired_at;comment:过期时间"`
	RevokedAt     *time.Time `gorm:"column:revoked_at;comment:撤销时间"`
	LastRotatedAt *time.Time `gorm:"column:last_rotated_at;comment:最后轮换时间"`
	CreatedBy     string     `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy     string     `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy     string     `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (RefreshTokenEntity) TableName() string {
	return TableNameRefreshToken
}

type RefreshTokenEntityList []RefreshTokenEntity

func (l RefreshTokenEntityList) ToMap() map[string]RefreshTokenEntity {
	m := make(map[string]RefreshTokenEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
