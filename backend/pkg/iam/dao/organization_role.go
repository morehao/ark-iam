package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type OrganizationRoleCond struct {
	*genericdao.BaseCond
	TenantID       uint
	OrganizationID uint
	Name           string
}

func (c *OrganizationRoleCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.OrganizationID != 0 {
		db.Where(tableName + ".organization_id = ?", c.OrganizationID)
	}
	if c.Name != "" {
		db.Where(tableName + ".name = ?", c.Name)
	}
}

type OrganizationRoleDao struct {
	*genericdao.GenericDao[model.OrganizationRoleEntity, model.OrganizationRoleEntityList]
}

func NewOrganizationRoleDao() *OrganizationRoleDao {
	return &OrganizationRoleDao{
		GenericDao: genericdao.NewGenericDao[model.OrganizationRoleEntity, model.OrganizationRoleEntityList](
			model.TableNameOrganizationRole, "OrganizationRoleDao",
			dbclient.IamDB,
		),
	}
}