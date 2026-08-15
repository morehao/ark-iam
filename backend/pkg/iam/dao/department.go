package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type DepartmentCond struct {
	*gormdao.BaseCond
	TenantID     string
	ParentID     string
	Name         string
	Code         string
	LeaderUserID string
}

func (c *DepartmentCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.ParentID != "" {
		db.Where(tableName+".parent_id = ?", c.ParentID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.Code != "" {
		db.Where(tableName+".code = ?", c.Code)
	}
	if c.LeaderUserID != "" {
		db.Where(tableName+".leader_user_id = ?", c.LeaderUserID)
	}
}

type DepartmentDao struct {
	*gormdao.Dao[model.DepartmentEntity, model.DepartmentEntityList, string]
}

func NewDepartmentDao(opts ...DaoOption) *DepartmentDao {
	return &DepartmentDao{
		Dao: gormdao.NewDao[model.DepartmentEntity, model.DepartmentEntityList, string](
			model.TableNameDepartment, "DepartmentDao",
			resolveDBGetter(opts...),
		),
	}
}
