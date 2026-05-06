package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type RoleMenuCond struct {
	*genericdao.BaseCond
	TenantID uint
	RoleID   uint
	MenuID   uint
}

func (c *RoleMenuCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.RoleID != 0 {
		db.Where(tableName + ".role_id = ?", c.RoleID)
	}
	if c.MenuID != 0 {
		db.Where(tableName + ".menu_id = ?", c.MenuID)
	}
}

type RoleMenuDao struct {
	*genericdao.GenericDao[model.RoleMenuEntity, model.RoleMenuEntityList]
}

func NewRoleMenuDao() *RoleMenuDao {
	return &RoleMenuDao{
		GenericDao: genericdao.NewGenericDao[model.RoleMenuEntity, model.RoleMenuEntityList](
			model.TableNameRoleMenu, "RoleMenuDao",
			dbclient.IamDB,
		),
	}
}