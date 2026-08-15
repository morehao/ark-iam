package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type ApplicationClientCond struct {
	*gormdao.BaseCond
	TenantID string
	AppID    string
	Code   string
	Name   string
	Type   string
	Status string
}

func (c *ApplicationClientCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.AppID != "" {
		db.Where(tableName+".app_id = ?", c.AppID)
	}
	if c.Code != "" {
		db.Where(tableName+".code = ?", c.Code)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.Type != "" {
		db.Where(tableName+".type = ?", c.Type)
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
}

type ApplicationClientDao struct {
	*gormdao.Dao[model.ApplicationClientEntity, model.ApplicationClientEntityList, string]
}

func NewApplicationClientDao(opts ...DaoOption) *ApplicationClientDao {
	return &ApplicationClientDao{
		Dao: gormdao.NewDao[model.ApplicationClientEntity, model.ApplicationClientEntityList, string](
			model.TableNameApplicationClient, "ApplicationClientDao",
			resolveDBGetter(opts...),
		),
	}
}
