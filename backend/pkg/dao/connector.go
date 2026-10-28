package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConnectorCond struct {
	*gormdao.BaseCond
	TenantID    string
	Protocol    model.ConnectorProtocol
	Provider    model.ConnectorProvider
	Status      model.ConnectorStatus
	Name        string
	DisplayName string
}

func (c *ConnectorCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.Protocol != "" {
		db.Where(clause.Eq{Column: col("protocol"), Value: c.Protocol})
	}
	if c.Provider != "" {
		db.Where(clause.Eq{Column: col("provider"), Value: c.Provider})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
	if c.Name != "" {
		db.Where(clause.Eq{Column: col("name"), Value: c.Name})
	}
	if c.DisplayName != "" {
		db.Where(clause.Eq{Column: col("display_name"), Value: c.DisplayName})
	}
}

type ConnectorDao struct {
	*gormdao.Dao[model.ConnectorEntity, model.ConnectorEntityList, string]
}

func NewConnectorDao(opts ...DaoOption) *ConnectorDao {
	return &ConnectorDao{
		Dao: gormdao.NewDao[model.ConnectorEntity, model.ConnectorEntityList, string](
			model.TableNameConnector, "ConnectorDao",
			resolveDBGetter(opts...),
		),
	}
}
