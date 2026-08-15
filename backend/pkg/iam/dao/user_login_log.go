package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type UserLoginLogCond struct {
	*gormdao.BaseCond
	TenantID string
	UserID   string
	LoginIP  string
}

func (c *UserLoginLogCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.UserID != "" {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
	if c.LoginIP != "" {
		db.Where(tableName+".login_ip = ?", c.LoginIP)
	}
}

type UserLoginLogDao struct {
	*gormdao.Dao[model.UserLoginLogEntity, model.UserLoginLogEntityList, string]
}

func NewUserLoginLogDao(opts ...DaoOption) *UserLoginLogDao {
	return &UserLoginLogDao{
		Dao: gormdao.NewDao[model.UserLoginLogEntity, model.UserLoginLogEntityList, string](
			model.TableNameUserLoginLog, "UserLoginLogDao",
			resolveDBGetter(opts...),
		),
	}
}
