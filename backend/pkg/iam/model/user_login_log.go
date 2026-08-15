package model

import (
	"time"

	"github.com/morehao/golib/dbaccess/gormdao"
)

const TableNameUserLoginLog = "user_login_log"

type UserLoginLogEntity struct {
	gormdao.BaseEntity
	PersonID  string    `gorm:"column:person_id;type:varchar(36);not null;default:'';comment:自然人ID"`
	TenantID  string    `gorm:"column:tenant_id;type:varchar(36);not null;default:'';comment:租户id"`
	UserID    string    `gorm:"column:user_id;type:varchar(36);not null;default:'';comment:用户ID"`
	LoginType string    `gorm:"column:login_type;type:varchar(32);not null;default:'';comment:登录类型"`
	LoginIP   string    `gorm:"column:login_ip;type:varchar(64);comment:登录IP地址"`
	UserAgent string    `gorm:"column:user_agent;type:varchar(512);comment:用户代理信息"`
	LoginTime time.Time `gorm:"column:login_time;not null;default:CURRENT_TIMESTAMP;comment:登录时间"`
	CreatedBy string    `gorm:"column:created_by;type:varchar(36);not null;default:'';comment:创建人ID"`
	UpdatedBy string    `gorm:"column:updated_by;type:varchar(36);not null;default:'';comment:更新人ID"`
	DeletedBy string    `gorm:"column:deleted_by;type:varchar(36);not null;default:'';comment:删除人ID"`
}

func (UserLoginLogEntity) TableName() string {
	return TableNameUserLoginLog
}

type UserLoginLogEntityList []UserLoginLogEntity

func (l UserLoginLogEntityList) ToMap() map[string]UserLoginLogEntity {
	m := make(map[string]UserLoginLogEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
