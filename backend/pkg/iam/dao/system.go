package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type SystemCond struct {
	*gormdao.BaseCond
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
	*gormdao.Dao[model.SystemEntity, model.SystemEntityList, uint]
}

func NewSystemDao(opts ...DaoOption) *SystemDao {
	return &SystemDao{
		Dao: gormdao.NewDao[model.SystemEntity, model.SystemEntityList, uint](
			model.TableNameSystem, "SystemDao",
			resolveDBGetter(opts...),
		),
	}
}
