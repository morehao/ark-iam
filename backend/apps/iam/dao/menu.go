package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type MenuCond struct {
	*gormdao.BaseCond
	AppID    uint
	ParentID uint
	Name     string
	Code     string
	Type     string
	Status   string
}

func (c *MenuCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.AppID != 0 {
		db.Where(tableName+".app_id = ?", c.AppID)
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
	if c.Type != "" {
		db.Where(tableName+".type = ?", c.Type)
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
}

type MenuDao struct {
	*gormdao.Dao[model.MenuEntity, model.MenuEntityList]
}

func NewMenuDao() *MenuDao {
	return &MenuDao{
		Dao: gormdao.NewDao[model.MenuEntity, model.MenuEntityList](
			model.TableNameMenu, "MenuDao",
			dbclient.IamDB,
		),
	}
}
