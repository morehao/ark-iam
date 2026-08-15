package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
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
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.UserID != "" {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
	if c.RoleID != "" {
		db.Where(tableName+".role_id = ?", c.RoleID)
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
