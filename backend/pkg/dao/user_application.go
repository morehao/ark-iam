package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
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
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.UserID != "" {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
	if c.AppID != "" {
		db.Where(tableName+".app_id = ?", c.AppID)
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
	if len(c.UserIDs) > 0 {
		db.Where(tableName+".user_id IN ?", c.UserIDs)
	}
	if len(c.AppIDs) > 0 {
		db.Where(tableName+".app_id IN ?", c.AppIDs)
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
