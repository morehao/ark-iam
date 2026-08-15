package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type RoleMenuCond struct {
	*gormdao.BaseCond
	TenantID string
	RoleID   string
	MenuID   string
}

func (c *RoleMenuCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.RoleID != "" {
		db.Where(tableName+".role_id = ?", c.RoleID)
	}
	if c.MenuID != "" {
		db.Where(tableName+".menu_id = ?", c.MenuID)
	}
}

type RoleMenuDao struct {
	*gormdao.Dao[model.RoleMenuEntity, model.RoleMenuEntityList, string]
}

func NewRoleMenuDao(opts ...DaoOption) *RoleMenuDao {
	return &RoleMenuDao{
		Dao: gormdao.NewDao[model.RoleMenuEntity, model.RoleMenuEntityList, string](
			model.TableNameRoleMenu, "RoleMenuDao",
			resolveDBGetter(opts...),
		),
	}
}
