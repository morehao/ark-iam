package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type OrganizationCond struct {
	*gormdao.BaseCond
	TenantID string
	Name     string
}

func (c *OrganizationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
}

type OrganizationDao struct {
	*gormdao.Dao[model.OrganizationEntity, model.OrganizationEntityList, string]
}

func NewOrganizationDao(opts ...DaoOption) *OrganizationDao {
	return &OrganizationDao{
		Dao: gormdao.NewDao[model.OrganizationEntity, model.OrganizationEntityList, string](
			model.TableNameOrganization, "OrganizationDao",
			resolveDBGetter(opts...),
		),
	}
}
