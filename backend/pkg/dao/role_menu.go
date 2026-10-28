package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RoleMenuCond struct {
	*gormdao.BaseCond
	TenantID string
	RoleID   string
	RoleIDs  []string
	MenuID   string
}

func (c *RoleMenuCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.RoleID != "" {
		db.Where(clause.Eq{Column: col("role_id"), Value: c.RoleID})
	}
	if len(c.RoleIDs) > 0 {
		db.Where(clause.IN{Column: col("role_id"), Values: toAnySlice(c.RoleIDs)})
	}
	if c.MenuID != "" {
		db.Where(clause.Eq{Column: col("menu_id"), Value: c.MenuID})
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
