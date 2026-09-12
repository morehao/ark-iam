package model

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameApiKey = "api_key"

type ApiKeyEntity struct {
	gormdao.BaseEntity
	TenantID    string          `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	OwnerUserID string          `gorm:"column:owner_user_id;type:varchar(36);not null;default:'';comment:归属用户id(真实用户本人或服务账号)"`
	Name        string          `gorm:"column:name;type:varchar(128);not null;default:'';comment:密钥名称"`
	KeyHash     string          `gorm:"column:key_hash;type:varchar(256);not null;default:'';comment:密钥哈希"`
	KeyPrefix   string          `gorm:"column:key_prefix;type:varchar(16);not null;default:'';comment:密钥前缀(前7位)"`
	Scope       json.RawMessage `gorm:"column:scope;type:json;not null;default:'{}';comment:权限范围"`
	ExpiredAt   *time.Time      `gorm:"column:expired_at;comment:过期时间"`
	LastUsedAt  sql.NullTime    `gorm:"column:last_used_at;comment:最后使用时间"`
	RevokedAt   *time.Time      `gorm:"column:revoked_at;comment:撤销时间"`
	CreatedBy   string          `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人id"`
	UpdatedBy   string          `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人id"`
	DeletedBy   string          `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人id"`
}

func (ApiKeyEntity) TableName() string {
	return TableNameApiKey
}

type ApiKeyEntityList []ApiKeyEntity

func (l ApiKeyEntityList) ToMap() map[string]ApiKeyEntity {
	m := make(map[string]ApiKeyEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
