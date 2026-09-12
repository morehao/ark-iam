package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

// MenuOrderBySort 菜单的规范排序字段：先按控制台可维护的「排序」(sort) 升序，
// 再按 code 升序兜底（同应用内 code 唯一，保证顺序确定）。
// 菜单树按 parent_id 分组构建，该排序等价于「同一父级下按 sort 升序」。
// 菜单顺序是运维字段（种子 create_only，控制台可改），因此所有面向界面的菜单查询都必须显式带上它——
// 缺省时数据库不保证任何顺序，控制台里改「排序」将完全看不到效果。
const MenuOrderBySort = "sort, code"

type MenuCond struct {
	*gormdao.BaseCond
	AppID      string
	ParentID   string
	Name       string
	Code       string
	SeedKey    string // 种子身份键：内置菜单的稳定标识（定位内置菜单请用它，而不是可被改名的 Code）
	Type       model.MenuType
	Status     model.MenuStatus
	Visibility model.MenuVisibility
}

func (c *MenuCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.AppID != "" {
		db.Where(tableName+".app_id = ?", c.AppID)
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
	if c.SeedKey != "" {
		db.Where(tableName+".seed_key = ?", c.SeedKey)
	}
	if c.Type != "" {
		db.Where(tableName+".type = ?", c.Type)
	}
	if c.Visibility != "" {
		db.Where(tableName+".visibility = ?", c.Visibility)
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
}

type MenuDao struct {
	*gormdao.Dao[model.MenuEntity, model.MenuEntityList, string]
}

func NewMenuDao(opts ...DaoOption) *MenuDao {
	return &MenuDao{
		Dao: gormdao.NewDao[model.MenuEntity, model.MenuEntityList, string](
			model.TableNameMenu, "MenuDao",
			resolveDBGetter(opts...),
		),
	}
}
