package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type UserDepartmentRelationCond struct {
	*genericdao.BaseCond
	TenantID     uint
	UserID       uint
	DepartmentID uint
	IsPrimary    int8
}

func (c *UserDepartmentRelationCond) BuildCondition(db *gorm.DB, tableName string) {
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
	if c.IsPrimary != 0 {
		db.Where(tableName + ".is_primary = ?", c.IsPrimary)
	}
}

type UserDepartmentRelationDao struct {
	*genericdao.GenericDao[model.UserDepartmentRelationEntity, model.UserDepartmentRelationEntityList]
}

func NewUserDepartmentRelationDao() *UserDepartmentRelationDao {
	return &UserDepartmentRelationDao{
		GenericDao: genericdao.NewGenericDao[model.UserDepartmentRelationEntity, model.UserDepartmentRelationEntityList](
			model.TableNameUserDepartmentRelation, "UserDepartmentRelationDao",
			dbclient.IamDB,
		),
	}
}