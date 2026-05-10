package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type PersonCond struct {
	*genericdao.BaseCond
	Username     string
	PrimaryEmail string
	PrimaryPhone string
	Name         string
	IsSuspended  *int8
}

func (c *PersonCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.Username != "" {
		db.Where(tableName+".username = ?", c.Username)
	}
	if c.PrimaryEmail != "" {
		db.Where(tableName+".primary_email = ?", c.PrimaryEmail)
	}
	if c.PrimaryPhone != "" {
		db.Where(tableName+".primary_phone = ?", c.PrimaryPhone)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.IsSuspended != nil {
		db.Where(tableName+".is_suspended = ?", *c.IsSuspended)
	}
}

type PersonDao struct {
	*genericdao.GenericDao[model.PersonEntity, model.PersonEntityList]
}

func NewPersonDao() *PersonDao {
	return &PersonDao{
		GenericDao: genericdao.NewGenericDao[model.PersonEntity, model.PersonEntityList](
			model.TableNamePerson, "PersonDao",
			dbclient.IamDB,
		),
	}
}
