package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type ApplicationSecretCond struct {
	*genericdao.BaseCond
	ApplicationID uint
	Name          string
}

func (c *ApplicationSecretCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.ApplicationID != 0 {
		db.Where(tableName + ".application_id = ?", c.ApplicationID)
	}
	if c.Name != "" {
		db.Where(tableName + ".name = ?", c.Name)
	}
}

type ApplicationSecretDao struct {
	*genericdao.GenericDao[model.ApplicationSecretEntity, model.ApplicationSecretEntityList]
}

func NewApplicationSecretDao() *ApplicationSecretDao {
	return &ApplicationSecretDao{
		GenericDao: genericdao.NewGenericDao[model.ApplicationSecretEntity, model.ApplicationSecretEntityList](
			model.TableNameApplicationSecret, "ApplicationSecretDao",
			dbclient.IamDB,
		),
	}
}
