package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type ConnectorCond struct {
	*genericdao.BaseCond
	TenantID    uint
	ConnectorID string
}

func (c *ConnectorCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.ConnectorID != "" {
		db.Where(tableName+".connector_id = ?", c.ConnectorID)
	}
}

type ConnectorDao struct {
	*genericdao.GenericDao[model.ConnectorEntity, model.ConnectorEntityList]
}

func NewConnectorDao() *ConnectorDao {
	return &ConnectorDao{
		GenericDao: genericdao.NewGenericDao[model.ConnectorEntity, model.ConnectorEntityList](
			model.TableNameConnector, "ConnectorDao",
			dbclient.IamDB,
		),
	}
}