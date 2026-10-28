package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PersonCond struct {
	*gormdao.BaseCond
	IDs          []string // 主键 IN 批量查询
	Username     string
	PrimaryEmail string
	PrimaryPhone string
	Name         string
	Status       model.PersonStatus
}

func (c *PersonCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if len(c.IDs) > 0 {
		db.Where(clause.IN{Column: col("id"), Values: toAnySlice(c.IDs)})
	}
	if c.Username != "" {
		db.Where(clause.Eq{Column: col("username"), Value: c.Username})
	}
	if c.PrimaryEmail != "" {
		db.Where(clause.Eq{Column: col("primary_email"), Value: c.PrimaryEmail})
	}
	if c.PrimaryPhone != "" {
		db.Where(clause.Eq{Column: col("primary_phone"), Value: c.PrimaryPhone})
	}
	if c.Name != "" {
		db.Where(clause.Eq{Column: col("name"), Value: c.Name})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
}

type PersonDao struct {
	*gormdao.Dao[model.PersonEntity, model.PersonEntityList, string]
}

func NewPersonDao(opts ...DaoOption) *PersonDao {
	return &PersonDao{
		Dao: gormdao.NewDao[model.PersonEntity, model.PersonEntityList, string](
			model.TableNamePerson, "PersonDao",
			resolveDBGetter(opts...),
		),
	}
}
