package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type LogCond struct {
	*genericdao.BaseCond
	TenantID uint
	Key     string
}

func (c *LogCond) BuildCondition(db *gorm.DB, tableName string) {
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

type LogDao struct {
	*genericdao.GenericDao[model.LogEntity, model.LogEntityList]
}

func NewLogDao() *LogDao {
	return &LogDao{
		GenericDao: genericdao.NewGenericDao[model.LogEntity, model.LogEntityList](
			model.TableNameLog, "LogDao",
			dbclient.IamDB,
		),
	}
}