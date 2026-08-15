package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type ApplicationCond struct {
	*gormdao.BaseCond
	Name   string
	Type   string
	Status string
	Code   string
}

func (c *ApplicationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.Type != "" {
		db.Where(tableName+".type = ?", c.Type)
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
	if c.Code != "" {
		db.Where(tableName+".code = ?", c.Code)
	}
}

type ApplicationDao struct {
	*gormdao.Dao[model.ApplicationEntity, model.ApplicationEntityList, string]
}

func NewApplicationDao(opts ...DaoOption) *ApplicationDao {
	return &ApplicationDao{
		Dao: gormdao.NewDao[model.ApplicationEntity, model.ApplicationEntityList, string](
			model.TableNameApplication, "ApplicationDao",
			resolveDBGetter(opts...),
		),
	}
}
