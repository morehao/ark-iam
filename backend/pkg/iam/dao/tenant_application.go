package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type TenantApplicationCond struct {
	*gormdao.BaseCond
	TenantID string
	AppID    string
	Status   string
}

func (c *TenantApplicationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.AppID != "" {
		db.Where(tableName+".app_id = ?", c.AppID)
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
}

type TenantApplicationDao struct {
	*gormdao.Dao[model.TenantApplicationEntity, model.TenantApplicationEntityList, string]
}

func NewTenantApplicationDao(opts ...DaoOption) *TenantApplicationDao {
	return &TenantApplicationDao{
		Dao: gormdao.NewDao[model.TenantApplicationEntity, model.TenantApplicationEntityList, string](
			model.TableNameTenantApplication, "TenantApplicationDao",
			resolveDBGetter(opts...),
		),
	}
}
