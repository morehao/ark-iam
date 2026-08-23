package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type MenuCond struct {
	*gormdao.BaseCond
	AppID      string
	ParentID   string
	Name       string
	Code       string
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
