package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type SystemCond struct {
	*genericdao.BaseCond
	TenantID uint
	Key      string
}

func (c *SystemCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.Key != "" {
		db.Where(tableName+".key = ?", c.Key)
	}
}

type SystemDao struct {
	*genericdao.GenericDao[model.SystemEntity, model.SystemEntityList]
}

func NewSystemDao() *SystemDao {
	return &SystemDao{
		GenericDao: genericdao.NewGenericDao[model.SystemEntity, model.SystemEntityList](
			model.TableNameSystem, "SystemDao",
			dbclient.IamDB,
		),
	}
}