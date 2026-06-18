package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type UserCond struct {
	*genericdao.BaseCond
	TenantID      uint
	PersonID      uint
	Username      string
	PrimaryEmail  string
	PrimaryPhone  string
	Name          string
	IsSuspended   *int8
}

func (c *UserCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.PersonID != 0 {
		db.Where(tableName+".person_id = ?", c.PersonID)
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

type UserDao struct {
	*genericdao.GenericDao[model.UserEntity, model.UserEntityList]
}

func NewUserDao() *UserDao {
	return &UserDao{
		GenericDao: genericdao.NewGenericDao[model.UserEntity, model.UserEntityList](
			model.TableNameUser, "UserDao",
			dbclient.IamDB,
		),
	}
}
