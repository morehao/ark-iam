package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type OrganizationCond struct {
	*genericdao.BaseCond
	TenantID uint
	Name     string
}

func (c *OrganizationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.Name != "" {
		db.Where(tableName + ".name = ?", c.Name)
	}
}

type OrganizationDao struct {
	*genericdao.GenericDao[model.OrganizationEntity, model.OrganizationEntityList]
}

func NewOrganizationDao() *OrganizationDao {
	return &OrganizationDao{
		GenericDao: genericdao.NewGenericDao[model.OrganizationEntity, model.OrganizationEntityList](
			model.TableNameOrganization, "OrganizationDao",
			dbclient.IamDB,
		),
	}
}