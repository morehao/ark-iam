package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type RoleMenuCond struct {
	*gormdao.BaseCond
	TenantID uint
	RoleID   uint
	MenuID   uint
}

func (c *RoleMenuCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.RoleID != 0 {
		db.Where(tableName+".role_id = ?", c.RoleID)
	}
	if c.MenuID != 0 {
		db.Where(tableName+".menu_id = ?", c.MenuID)
	}
}

type RoleMenuDao struct {
	*gormdao.Dao[model.RoleMenuEntity, model.RoleMenuEntityList, uint]
}

func NewRoleMenuDao(opts ...DaoOption) *RoleMenuDao {
	return &RoleMenuDao{
		Dao: gormdao.NewDao[model.RoleMenuEntity, model.RoleMenuEntityList, uint](
			model.TableNameRoleMenu, "RoleMenuDao",
			resolveDBGetter(opts...),
		),
	}
}
