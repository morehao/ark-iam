package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TenantApplicationCond struct {
	*gormdao.BaseCond
	TenantID string
	AppID    string
	AppIDs   []string // 批量点查（准入替换的预检一次校验多个应用）
	Status   model.TenantApplicationStatus
}

func (c *TenantApplicationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.AppID != "" {
		db.Where(clause.Eq{Column: col("app_id"), Value: c.AppID})
	}
	if len(c.AppIDs) > 0 {
		db.Where(clause.IN{Column: col("app_id"), Values: toAnySlice(c.AppIDs)})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
}

type TenantApplicationDao struct {
	*gormdao.Dao[model.TenantApplicationEntity, model.TenantApplicationEntityList, string]
}

func NewTenantApplicationDao(opts ...DaoOption) *TenantApplicationDao {
	return &TenantApplicationDao{
		Dao: gormdao.NewDao[model.TenantApplicationEntity, model.TenantApplicationEntityList, string](
			model.TableNameTenantApplication, "TenantApplicationDao",
			resolveDBGetter(opts...),
		),
	}
}
