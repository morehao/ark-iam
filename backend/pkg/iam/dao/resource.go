package dao

import (
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type ResourceCond struct {
	*gormdao.BaseCond
	TenantID  uint
	Name      string
	Indicator string
}

func (c *ResourceCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.Indicator != "" {
		db.Where(tableName+".indicator = ?", c.Indicator)
	}
}

type ResourceDao struct {
	*gormdao.Dao[model.ResourceEntity, model.ResourceEntityList]
}

func NewResourceDao() *ResourceDao {
	return &ResourceDao{
		Dao: gormdao.NewDao[model.ResourceEntity, model.ResourceEntityList](
			model.TableNameResource, "ResourceDao",
			dbclient.IamDB,
		),
	}
}
