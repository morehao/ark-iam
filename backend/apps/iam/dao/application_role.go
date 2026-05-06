package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type ApplicationRoleCond struct {
	*genericdao.BaseCond
	TenantID      uint
	ApplicationID uint
	RoleID        uint
}

func (c *ApplicationRoleCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.ApplicationID != 0 {
		db.Where(tableName + ".application_id = ?", c.ApplicationID)
	}
	if c.RoleID != 0 {
		db.Where(tableName + ".role_id = ?", c.RoleID)
	}
}

type ApplicationRoleDao struct {
	*genericdao.GenericDao[model.ApplicationRoleEntity, model.ApplicationRoleEntityList]
}

func NewApplicationRoleDao() *ApplicationRoleDao {
	return &ApplicationRoleDao{
		GenericDao: genericdao.NewGenericDao[model.ApplicationRoleEntity, model.ApplicationRoleEntityList](
			model.TableNameApplicationRole, "ApplicationRoleDao",
			dbclient.IamDB,
		),
	}
}