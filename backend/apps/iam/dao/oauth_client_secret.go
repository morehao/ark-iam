package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type OAuthClientSecretCond struct {
	*genericdao.BaseCond
	OAuthClientID uint
	Name          string
}

func (c *OAuthClientSecretCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.OAuthClientID != 0 {
		db.Where(tableName + ".oauth_client_id = ?", c.OAuthClientID)
	}
	if c.Name != "" {
		db.Where(tableName + ".name = ?", c.Name)
	}
}

type OAuthClientSecretDao struct {
	*genericdao.GenericDao[model.OAuthClientSecretEntity, model.OAuthClientSecretEntityList]
}

func NewOAuthClientSecretDao() *OAuthClientSecretDao {
	return &OAuthClientSecretDao{
		GenericDao: genericdao.NewGenericDao[model.OAuthClientSecretEntity, model.OAuthClientSecretEntityList](
			model.TableNameOAuthClientSecret, "OAuthClientSecretDao",
			dbclient.IamDB,
		),
	}
}
