package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApplicationCond struct {
	*gormdao.BaseCond
	IDs     []string
	Name    string
	Source  model.AppSource
	Status  model.AppStatus
	Code    string
	SeedKey string // 种子身份键：内置应用的稳定标识（定位内置应用请用它，而不是可被改名的 Code）
}

func (c *ApplicationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if len(c.IDs) > 0 {
		db.Where(clause.IN{Column: col("id"), Values: toAnySlice(c.IDs)})
	}
	if c.Name != "" {
		db.Where(clause.Eq{Column: col("name"), Value: c.Name})
	}
	if c.Source != "" {
		db.Where(clause.Eq{Column: col("source"), Value: c.Source})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
	if c.Code != "" {
		db.Where(clause.Eq{Column: col("code"), Value: c.Code})
	}
	if c.SeedKey != "" {
		db.Where(clause.Eq{Column: col("seed_key"), Value: c.SeedKey})
	}
}

type ApplicationDao struct {
	*gormdao.Dao[model.ApplicationEntity, model.ApplicationEntityList, string]
}

func NewApplicationDao(opts ...DaoOption) *ApplicationDao {
	return &ApplicationDao{
		Dao: gormdao.NewDao[model.ApplicationEntity, model.ApplicationEntityList, string](
			model.TableNameApplication, "ApplicationDao",
			resolveDBGetter(opts...),
		),
	}
}
