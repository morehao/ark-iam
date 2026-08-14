package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type UserDepartmentCond struct {
	*gormdao.BaseCond
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
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.UserID != 0 {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
	if c.DepartmentID != 0 {
		db.Where(tableName+".department_id = ?", c.DepartmentID)
	}
	if c.IsPrimary != nil {
		db.Where(tableName+".is_primary = ?", *c.IsPrimary)
	}
}

type UserDepartmentDao struct {
	*gormdao.Dao[model.UserDepartmentEntity, model.UserDepartmentEntityList]
}

func NewUserDepartmentDao() *UserDepartmentDao {
	return &UserDepartmentDao{
		Dao: gormdao.NewDao[model.UserDepartmentEntity, model.UserDepartmentEntityList](
			model.TableNameUserDepartment, "UserDepartmentDao",
			dbclient.IamDB, gormdao.WithoutSoftDelete(),
		),
	}
}
