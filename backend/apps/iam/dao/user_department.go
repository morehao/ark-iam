package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type UserDepartmentCond struct {
	*genericdao.BaseCond
	TenantID     uint
	UserID       uint
	DepartmentID uint
	IsPrimary    *int8
}

func (c *UserDepartmentCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.UserID != 0 {
		db.Where(tableName + ".user_id = ?", c.UserID)
	}
	if c.DepartmentID != 0 {
		db.Where(tableName + ".department_id = ?", c.DepartmentID)
	}
	if c.IsPrimary != nil {
		db.Where(tableName + ".is_primary = ?", *c.IsPrimary)
	}
}

type UserDepartmentDao struct {
	*genericdao.GenericDao[model.UserDepartmentEntity, model.UserDepartmentEntityList]
}

func NewUserDepartmentDao() *UserDepartmentDao {
	return &UserDepartmentDao{
		GenericDao: genericdao.NewGenericDao[model.UserDepartmentEntity, model.UserDepartmentEntityList](
			model.TableNameUserDepartment, "UserDepartmentDao",
			dbclient.IamDB,
		),
	}
}
