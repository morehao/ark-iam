package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type ApplicationCond struct {
	*genericdao.BaseCond
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
		db.Where(tableName + ".name = ?", c.Name)
	}
	if c.Type != "" {
		db.Where(tableName + ".type = ?", c.Type)
	}
	if c.Status != "" {
		db.Where(tableName + ".status = ?", c.Status)
	}
	if c.Code != "" {
		db.Where(tableName + ".code = ?", c.Code)
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
