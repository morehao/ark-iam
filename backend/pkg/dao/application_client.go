package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApplicationClientCond struct {
	*gormdao.BaseCond
	TenantID string
	AppID    string
	Code     string
	Name     string
	Source   model.ApplicationClientSource
	Status   model.ApplicationClientStatus
}

func (c *ApplicationClientCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.AppID != "" {
		db.Where(clause.Eq{Column: col("app_id"), Value: c.AppID})
	}
	if c.Code != "" {
		db.Where(clause.Eq{Column: col("code"), Value: c.Code})
	}
	if c.Name != "" {
		db.Where(clause.Eq{Column: col("name"), Value: c.Name})
	}
	if c.Source != "" {
		db.Where(clause.Eq{Column: col("source"), Value: c.Source})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
}

type ApplicationClientDao struct {
	*gormdao.Dao[model.ApplicationClientEntity, model.ApplicationClientEntityList, string]
}

func NewApplicationClientDao(opts ...DaoOption) *ApplicationClientDao {
	return &ApplicationClientDao{
		Dao: gormdao.NewDao[model.ApplicationClientEntity, model.ApplicationClientEntityList, string](
			model.TableNameApplicationClient, "ApplicationClientDao",
			resolveDBGetter(opts...),
		),
	}
}
