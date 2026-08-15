package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type OrganizationRoleCond struct {
	*gormdao.BaseCond
	TenantID       uint
	OrganizationID uint
	Name           string
}

func (c *OrganizationRoleCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.OrganizationID != 0 {
		db.Where(tableName+".organization_id = ?", c.OrganizationID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
}

type OrganizationRoleDao struct {
	*gormdao.Dao[model.OrganizationRoleEntity, model.OrganizationRoleEntityList, uint]
}

func NewOrganizationRoleDao(opts ...DaoOption) *OrganizationRoleDao {
	return &OrganizationRoleDao{
		Dao: gormdao.NewDao[model.OrganizationRoleEntity, model.OrganizationRoleEntityList, uint](
			model.TableNameOrganizationRole, "OrganizationRoleDao",
			resolveDBGetter(opts...),
		),
	}
}
