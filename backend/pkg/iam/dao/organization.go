package dao

import (
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type OrganizationCond struct {
	*gormdao.BaseCond
	TenantID uint
	Name     string
}

func (c *OrganizationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
}

type OrganizationDao struct {
	*gormdao.Dao[model.OrganizationEntity, model.OrganizationEntityList]
}

func NewOrganizationDao() *OrganizationDao {
	return &OrganizationDao{
		Dao: gormdao.NewDao[model.OrganizationEntity, model.OrganizationEntityList](
			model.TableNameOrganization, "OrganizationDao",
			dbclient.IamDB,
		),
	}
}
