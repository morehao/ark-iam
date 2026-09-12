package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type DepartmentUserCond struct {
	*gormdao.BaseCond
	TenantID     string
	DepartmentID string
	UserID       string
	UserIDs      []string
	RelationType model.DeptUserRelationType
}

func (c *DepartmentUserCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.DepartmentID != "" {
		db.Where(tableName+".department_id = ?", c.DepartmentID)
	}
	if c.UserID != "" {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
	if len(c.UserIDs) > 0 {
		db.Where(tableName+".user_id IN ?", c.UserIDs)
	}
	if c.RelationType != "" {
		db.Where(tableName+".relation_type = ?", c.RelationType)
	}
}

type DepartmentUserDao struct {
	*gormdao.Dao[model.DepartmentUserEntity, model.DepartmentUserEntityList, string]
}

func NewDepartmentUserDao(opts ...DaoOption) *DepartmentUserDao {
	return &DepartmentUserDao{
		Dao: gormdao.NewDao[model.DepartmentUserEntity, model.DepartmentUserEntityList, string](
			model.TableNameDepartmentUser, "DepartmentUserDao",
			resolveDBGetter(opts...),
		),
	}
}
