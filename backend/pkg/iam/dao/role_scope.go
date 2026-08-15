package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type RoleScopeCond struct {
	*gormdao.BaseCond
	TenantID string
	RoleID   string
	ScopeID  string
}

func (c *RoleScopeCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.RoleID != "" {
		db.Where(tableName+".role_id = ?", c.RoleID)
	}
	if c.ScopeID != "" {
		db.Where(tableName+".scope_id = ?", c.ScopeID)
	}
}

type RoleScopeDao struct {
	*gormdao.Dao[model.RoleScopeEntity, model.RoleScopeEntityList, string]
}

func NewRoleScopeDao(opts ...DaoOption) *RoleScopeDao {
	return &RoleScopeDao{
		Dao: gormdao.NewDao[model.RoleScopeEntity, model.RoleScopeEntityList, string](
			model.TableNameRoleScope, "RoleScopeDao",
			resolveDBGetter(opts...),
		),
	}
}
