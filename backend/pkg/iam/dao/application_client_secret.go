package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type ApplicationClientSecretCond struct {
	*gormdao.BaseCond
	ApplicationClientID uint
	Name          string
}

func (c *ApplicationClientSecretCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.ApplicationClientID != 0 {
		db.Where(tableName+".application_client_id = ?", c.ApplicationClientID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
}

type ApplicationClientSecretDao struct {
	*gormdao.Dao[model.ApplicationClientSecretEntity, model.ApplicationClientSecretEntityList]
}

func NewApplicationClientSecretDao() *ApplicationClientSecretDao {
	return &ApplicationClientSecretDao{
		Dao: gormdao.NewDao[model.ApplicationClientSecretEntity, model.ApplicationClientSecretEntityList](
			model.TableNameApplicationClientSecret, "ApplicationClientSecretDao",
			dbclient.IamDB, gormdao.WithoutSoftDelete(),
		),
	}
}
