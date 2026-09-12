package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type DepartmentCond struct {
	*gormdao.BaseCond
	TenantID string
	ParentID string
	// DeptPath 子树条件（含自身）：dept_path = X OR dept_path LIKE X||'/%
	DeptPath string
	Status   model.DeptNodeStatus
	Name     string
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
	if c.DeptPath != "" {
		db.Where(tableName+".dept_path = ? OR "+tableName+".dept_path LIKE ?", c.DeptPath, c.DeptPath+"/%")
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
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
