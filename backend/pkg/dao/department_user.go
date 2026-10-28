package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.DepartmentID != "" {
		db.Where(clause.Eq{Column: col("department_id"), Value: c.DepartmentID})
	}
	if c.UserID != "" {
		db.Where(clause.Eq{Column: col("user_id"), Value: c.UserID})
	}
	if len(c.UserIDs) > 0 {
		db.Where(clause.IN{Column: col("user_id"), Values: toAnySlice(c.UserIDs)})
	}
	if c.RelationType != "" {
		db.Where(clause.Eq{Column: col("relation_type"), Value: c.RelationType})
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
