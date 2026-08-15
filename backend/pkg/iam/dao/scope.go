package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type ScopeCond struct {
	*gormdao.BaseCond
	TenantID   uint
	ResourceID uint
	Name       string
}

func (c *ScopeCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.ResourceID != 0 {
		db.Where(tableName+".resource_id = ?", c.ResourceID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
}

type ScopeDao struct {
	*gormdao.Dao[model.ScopeEntity, model.ScopeEntityList, uint]
}

func NewScopeDao(opts ...DaoOption) *ScopeDao {
	return &ScopeDao{
		Dao: gormdao.NewDao[model.ScopeEntity, model.ScopeEntityList, uint](
			model.TableNameScope, "ScopeDao",
			resolveDBGetter(opts...),
		),
	}
}
