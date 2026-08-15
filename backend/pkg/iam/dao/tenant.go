package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type TenantCond struct {
	*gormdao.BaseCond
	CreatedBy   string
	DbUser      string
	DeletedBy   string
	IsSuspended int8
	Name        string
	Tag         string
	UpdatedBy   string
}

func (c *TenantCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.CreatedBy != "" {
		db.Where(tableName+".created_by = ?", c.CreatedBy)
	}
	if c.DbUser != "" {
		db.Where(tableName+".db_user = ?", c.DbUser)
	}
	if c.DeletedBy != "" {
		db.Where(tableName+".deleted_by = ?", c.DeletedBy)
	}
	if c.IsSuspended != 0 {
		db.Where(tableName+".is_suspended = ?", c.IsSuspended)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.Tag != "" {
		db.Where(tableName+".tag = ?", c.Tag)
	}
	if c.UpdatedBy != "" {
		db.Where(tableName+".updated_by = ?", c.UpdatedBy)
	}
}

type TenantDao struct {
	*gormdao.Dao[model.TenantEntity, model.TenantEntityList, string]
}

func NewTenantDao(opts ...DaoOption) *TenantDao {
	return &TenantDao{
		Dao: gormdao.NewDao[model.TenantEntity, model.TenantEntityList, string](
			model.TableNameTenant, "TenantDao",
			resolveDBGetter(opts...),
		),
	}
}
