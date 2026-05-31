package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type OAuthClientCond struct {
	*genericdao.BaseCond
	TenantID      uint
	AppID uint
	ClientID      string
	Name          string
	Type          string
	Status        string
}

func (c *OAuthClientCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.AppID != 0 {
		db.Where(tableName + ".app_id = ?", c.AppID)
	}
	if c.ClientID != "" {
		db.Where(tableName + ".client_id = ?", c.ClientID)
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
}

type OAuthClientDao struct {
	*genericdao.GenericDao[model.OAuthClientEntity, model.OAuthClientEntityList]
}

func NewOAuthClientDao() *OAuthClientDao {
	return &OAuthClientDao{
		GenericDao: genericdao.NewGenericDao[model.OAuthClientEntity, model.OAuthClientEntityList](
			model.TableNameOAuthClient, "OAuthClientDao",
			dbclient.IamDB,
		),
	}
}
