package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.UserID != "" {
		db.Where(clause.Eq{Column: col("user_id"), Value: c.UserID})
	}
	if c.LoginIP != "" {
		db.Where(clause.Eq{Column: col("login_ip"), Value: c.LoginIP})
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
