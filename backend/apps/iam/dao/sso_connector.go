package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type SsoConnectorCond struct {
	*genericdao.BaseCond
	TenantID      uint
	ProviderName  string
	ConnectorName string
}

func (c *SsoConnectorCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.ProviderName != "" {
		db.Where(tableName+".provider_name = ?", c.ProviderName)
	}
	if c.ConnectorName != "" {
		db.Where(tableName+".connector_name = ?", c.ConnectorName)
	}
}

type SsoConnectorDao struct {
	*genericdao.GenericDao[model.SsoConnectorEntity, model.SsoConnectorEntityList]
}

func NewSsoConnectorDao() *SsoConnectorDao {
	return &SsoConnectorDao{
		GenericDao: genericdao.NewGenericDao[model.SsoConnectorEntity, model.SsoConnectorEntityList](
			model.TableNameSsoConnector, "SsoConnectorDao",
			dbclient.IamDB,
		),
	}
}