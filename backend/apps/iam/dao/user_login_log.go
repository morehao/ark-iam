package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type UserLoginLogCond struct {
	*genericdao.BaseCond
	TenantID uint
	UserID   uint
	LoginIP  string
}

func (c *UserLoginLogCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.UserID != 0 {
		db.Where(tableName + ".user_id = ?", c.UserID)
	}
	if c.LoginIP != "" {
		db.Where(tableName + ".login_ip = ?", c.LoginIP)
	}
}

type UserLoginLogDao struct {
	*genericdao.GenericDao[model.UserLoginLogEntity, model.UserLoginLogEntityList]
}

func NewUserLoginLogDao() *UserLoginLogDao {
	return &UserLoginLogDao{
		GenericDao: genericdao.NewGenericDao[model.UserLoginLogEntity, model.UserLoginLogEntityList](
			model.TableNameUserLoginLog, "UserLoginLogDao",
			dbclient.IamDB,
		),
	}
}