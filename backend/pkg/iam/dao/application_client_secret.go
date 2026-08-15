package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type ApplicationClientSecretCond struct {
	*gormdao.BaseCond
	ApplicationClientID string
	Name                string
}

func (c *ApplicationClientSecretCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.ApplicationClientID != "" {
		db.Where(tableName+".application_client_id = ?", c.ApplicationClientID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
}

type ApplicationClientSecretDao struct {
	*gormdao.Dao[model.ApplicationClientSecretEntity, model.ApplicationClientSecretEntityList, string]
}

func NewApplicationClientSecretDao(opts ...DaoOption) *ApplicationClientSecretDao {
	return &ApplicationClientSecretDao{
		Dao: gormdao.NewDao[model.ApplicationClientSecretEntity, model.ApplicationClientSecretEntityList, string](
			model.TableNameApplicationClientSecret, "ApplicationClientSecretDao",
			resolveDBGetter(opts...),
		),
	}
}
