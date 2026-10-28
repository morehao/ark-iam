package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	col := condCols(tableName)
	if c.ApplicationClientID != "" {
		db.Where(clause.Eq{Column: col("application_client_id"), Value: c.ApplicationClientID})
	}
	if c.Name != "" {
		db.Where(clause.Eq{Column: col("name"), Value: c.Name})
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
