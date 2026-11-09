package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserApplicationCond 用户应用准入查询条件。
// 批量维度（UserIDs/AppIDs）供门禁与列表接口一次点查多条，避免 N+1。
type UserApplicationCond struct {
	*gormdao.BaseCond
	TenantID string
	UserID   string
	AppID    string
	Status   model.UserApplicationStatus
	UserIDs  []string
	AppIDs   []string
}

func (c *UserApplicationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.UserID != "" {
		db.Where(clause.Eq{Column: col("user_id"), Value: c.UserID})
	}
	if c.AppID != "" {
		db.Where(clause.Eq{Column: col("app_id"), Value: c.AppID})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
	if len(c.UserIDs) > 0 {
		db.Where(clause.IN{Column: col("user_id"), Values: toAnySlice(c.UserIDs)})
	}
	if len(c.AppIDs) > 0 {
		db.Where(clause.IN{Column: col("app_id"), Values: toAnySlice(c.AppIDs)})
	}
}

type UserApplicationDao struct {
	*gormdao.Dao[model.UserApplicationEntity, model.UserApplicationEntityList, string]
}

func NewUserApplicationDao(opts ...DaoOption) *UserApplicationDao {
	return &UserApplicationDao{
		Dao: gormdao.NewDao[model.UserApplicationEntity, model.UserApplicationEntityList, string](
			model.TableNameUserApplication, "UserApplicationDao",
			resolveDBGetter(opts...),
		),
	}
}
