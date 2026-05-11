package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameUserLoginLog = "user_login_log"

type UserLoginLogEntity struct {
	gorm.Model
	PersonID  uint      `gorm:"column:person_id;type:bigint unsigned;not null;default 0;comment:自然人ID"`
	TenantID  uint      `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id"`
	UserID    uint      `gorm:"column:user_id;type:bigint unsigned;not null;default 0;comment:用户ID"`
	LoginType string    `gorm:"column:login_type;type:varchar(32);not null;default '';comment:登录类型"`
	LoginIP   string    `gorm:"column:login_ip;type:varchar(64);comment:登录IP地址"`
	UserAgent string    `gorm:"column:user_agent;type:varchar(512);comment:用户代理信息"`
	LoginTime time.Time `gorm:"column:login_time;type:datetime;not null;default CURRENT_TIMESTAMP;comment:登录时间"`
	CreatedBy uint      `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人ID"`
	UpdatedBy uint      `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人ID"`
	DeletedBy uint      `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人ID"`
}

func (UserLoginLogEntity) TableName() string {
	return TableNameUserLoginLog
}

type UserLoginLogEntityList []UserLoginLogEntity

func (l UserLoginLogEntityList) ToMap() map[uint]UserLoginLogEntity {
	m := make(map[uint]UserLoginLogEntity)
	for _, v := range l {
		m[v.ID] = v
	}
	return m
}
