package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type RoleScopeCond struct {
	*genericdao.BaseCond
	TenantID uint
	RoleID   uint
	ScopeID  uint
}

func (c *RoleScopeCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.RoleID != 0 {
		db.Where(tableName + ".role_id = ?", c.RoleID)
	}
	if c.ScopeID != 0 {
		db.Where(tableName + ".scope_id = ?", c.ScopeID)
	}
}

type RoleScopeDao struct {
	*genericdao.GenericDao[model.RoleScopeEntity, model.RoleScopeEntityList]
}

func NewRoleScopeDao() *RoleScopeDao {
	return &RoleScopeDao{
		GenericDao: genericdao.NewGenericDao[model.RoleScopeEntity, model.RoleScopeEntityList](
			model.TableNameRoleScope, "RoleScopeDao",
			dbclient.IamDB,
		),
	}
}