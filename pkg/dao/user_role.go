package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRoleCond struct {
	*gormdao.BaseCond
	TenantID string
	UserID   string
	RoleID   string
}

func (c *UserRoleCond) BuildCondition(db *gorm.DB, tableName string) {
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
	if c.RoleID != "" {
		db.Where(clause.Eq{Column: col("role_id"), Value: c.RoleID})
	}
}

type UserRoleDao struct {
	*gormdao.Dao[model.UserRoleEntity, model.UserRoleEntityList, string]
}

func NewUserRoleDao(opts ...DaoOption) *UserRoleDao {
	return &UserRoleDao{
		Dao: gormdao.NewDao[model.UserRoleEntity, model.UserRoleEntityList, string](
			model.TableNameUserRole, "UserRoleDao",
			resolveDBGetter(opts...),
		),
	}
}
