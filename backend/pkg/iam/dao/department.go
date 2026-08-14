package dao

import (
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type DepartmentCond struct {
	*gormdao.BaseCond
	TenantID     uint
	ParentID     uint
	Name         string
	Code         string
	LeaderUserID uint
}

func (c *DepartmentCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.ParentID != 0 {
		db.Where(tableName+".parent_id = ?", c.ParentID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.Code != "" {
		db.Where(tableName+".code = ?", c.Code)
	}
	if c.LeaderUserID != 0 {
		db.Where(tableName+".leader_user_id = ?", c.LeaderUserID)
	}
}

type DepartmentDao struct {
	*gormdao.Dao[model.DepartmentEntity, model.DepartmentEntityList]
}

func NewDepartmentDao() *DepartmentDao {
	return &DepartmentDao{
		Dao: gormdao.NewDao[model.DepartmentEntity, model.DepartmentEntityList](
			model.TableNameDepartment, "DepartmentDao",
			dbclient.IamDB,
		),
	}
}
