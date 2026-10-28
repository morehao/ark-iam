package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TenantCond struct {
	*gormdao.BaseCond
	IDs       []string // 批量按主键取（名称回填等场景）
	CreatedBy string
	DbUser    string
	DeletedBy string
	Keyword   string // 模糊搜索：租户名 LIKE
	Name      string
	Status    model.TenantStatus // 空值表示不过滤
	Tag       string
	UpdatedBy string
}

func (c *TenantCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if len(c.IDs) > 0 {
		db.Where(clause.IN{Column: col("id"), Values: toAnySlice(c.IDs)})
	}
	if c.CreatedBy != "" {
		db.Where(clause.Eq{Column: col("created_by"), Value: c.CreatedBy})
	}
	if c.DbUser != "" {
		db.Where(clause.Eq{Column: col("db_user"), Value: c.DbUser})
	}
	if c.DeletedBy != "" {
		db.Where(clause.Eq{Column: col("deleted_by"), Value: c.DeletedBy})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
	if c.Keyword != "" {
		db.Where(clause.Like{Column: col("name"), Value: "%" + c.Keyword + "%"})
	}
	if c.Name != "" {
		db.Where(clause.Eq{Column: col("name"), Value: c.Name})
	}
	if c.Tag != "" {
		db.Where(clause.Eq{Column: col("tag"), Value: c.Tag})
	}
	if c.UpdatedBy != "" {
		db.Where(clause.Eq{Column: col("updated_by"), Value: c.UpdatedBy})
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
