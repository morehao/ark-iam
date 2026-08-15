package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type LogCond struct {
	*gormdao.BaseCond
	TenantID uint
	Key      string
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
	*gormdao.Dao[model.LogEntity, model.LogEntityList, uint]
}

func NewLogDao(opts ...DaoOption) *LogDao {
	return &LogDao{
		Dao: gormdao.NewDao[model.LogEntity, model.LogEntityList, uint](
			model.TableNameLog, "LogDao",
			resolveDBGetter(opts...),
		),
	}
}
