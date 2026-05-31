package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type TenantApplicationCond struct {
	*genericdao.BaseCond
	TenantID      uint
	ApplicationID uint
	Status        string
}

func (c *TenantApplicationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.ApplicationID != 0 {
		db.Where(tableName + ".application_id = ?", c.ApplicationID)
	}
	if c.Status != "" {
		db.Where(tableName + ".status = ?", c.Status)
	}
}

type TenantApplicationDao struct {
	*genericdao.GenericDao[model.TenantApplicationEntity, model.TenantApplicationEntityList]
}

func NewTenantApplicationDao() *TenantApplicationDao {
	return &TenantApplicationDao{
		GenericDao: genericdao.NewGenericDao[model.TenantApplicationEntity, model.TenantApplicationEntityList](
			model.TableNameTenantApplication, "TenantApplicationDao",
			dbclient.IamDB,
		),
	}
}
