package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type ApplicationCond struct {
	*genericdao.BaseCond
	TenantID uint
	Name     string
	Type     string
	Status   string
	ClientID string
}

func (c *ApplicationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.Name != "" {
		db.Where(tableName + ".name = ?", c.Name)
	}
	if c.Type != "" {
		db.Where(tableName + ".type = ?", c.Type)
	}
	if c.Status != "" {
		db.Where(tableName + ".status = ?", c.Status)
	}
	if c.ClientID != "" {
		db.Where(tableName + ".client_id = ?", c.ClientID)
	}
}

type ApplicationDao struct {
	*genericdao.GenericDao[model.ApplicationEntity, model.ApplicationEntityList]
}

func NewApplicationDao() *ApplicationDao {
	return &ApplicationDao{
		GenericDao: genericdao.NewGenericDao[model.ApplicationEntity, model.ApplicationEntityList](
			model.TableNameApplication, "ApplicationDao",
			dbclient.IamDB,
		),
	}
}
