package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type ScopeCond struct {
	*genericdao.BaseCond
	TenantID   uint
	ResourceID uint
	Name       string
}

func (c *ScopeCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.ResourceID != 0 {
		db.Where(tableName + ".resource_id = ?", c.ResourceID)
	}
	if c.Name != "" {
		db.Where(tableName + ".name = ?", c.Name)
	}
}

type ScopeDao struct {
	*genericdao.GenericDao[model.ScopeEntity, model.ScopeEntityList]
}

func NewScopeDao() *ScopeDao {
	return &ScopeDao{
		GenericDao: genericdao.NewGenericDao[model.ScopeEntity, model.ScopeEntityList](
			model.TableNameScope, "ScopeDao",
			dbclient.IamDB,
		),
	}
}