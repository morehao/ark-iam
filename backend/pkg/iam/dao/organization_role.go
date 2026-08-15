package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type OrganizationRoleCond struct {
	*gormdao.BaseCond
	TenantID       string
	OrganizationID string
	Name           string
}

func (c *OrganizationRoleCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.OrganizationID != "" {
		db.Where(tableName+".organization_id = ?", c.OrganizationID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
}

type OrganizationRoleDao struct {
	*gormdao.Dao[model.OrganizationRoleEntity, model.OrganizationRoleEntityList, string]
}

func NewOrganizationRoleDao(opts ...DaoOption) *OrganizationRoleDao {
	return &OrganizationRoleDao{
		Dao: gormdao.NewDao[model.OrganizationRoleEntity, model.OrganizationRoleEntityList, string](
			model.TableNameOrganizationRole, "OrganizationRoleDao",
			resolveDBGetter(opts...),
		),
	}
}
