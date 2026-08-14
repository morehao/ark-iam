package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type UserRoleCond struct {
	*gormdao.BaseCond
	TenantID uint
	UserID   uint
	RoleID   uint
}

func (c *UserRoleCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.UserID != 0 {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
	if c.RoleID != 0 {
		db.Where(tableName+".role_id = ?", c.RoleID)
	}
}

type UserRoleDao struct {
	*gormdao.Dao[model.UserRoleEntity, model.UserRoleEntityList]
}

func NewUserRoleDao() *UserRoleDao {
	return &UserRoleDao{
		Dao: gormdao.NewDao[model.UserRoleEntity, model.UserRoleEntityList](
			model.TableNameUserRole, "UserRoleDao",
			dbclient.IamDB, gormdao.WithoutSoftDelete(),
		),
	}
}
