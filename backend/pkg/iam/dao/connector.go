package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type ConnectorCond struct {
	*gormdao.BaseCond
	TenantID    uint
	Protocol    string
	Provider    string
	Status      string
	Name        string
	DisplayName string
}

func (c *ConnectorCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.Protocol != "" {
		db.Where(tableName+".protocol = ?", c.Protocol)
	}
	if c.Provider != "" {
		db.Where(tableName+".provider = ?", c.Provider)
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.DisplayName != "" {
		db.Where(tableName+".display_name = ?", c.DisplayName)
	}
}

type ConnectorDao struct {
	*gormdao.Dao[model.ConnectorEntity, model.ConnectorEntityList, uint]
}

func NewConnectorDao(opts ...DaoOption) *ConnectorDao {
	return &ConnectorDao{
		Dao: gormdao.NewDao[model.ConnectorEntity, model.ConnectorEntityList, uint](
			model.TableNameConnector, "ConnectorDao",
			resolveDBGetter(opts...),
		),
	}
}
