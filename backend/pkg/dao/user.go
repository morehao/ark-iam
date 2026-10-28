package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserCond struct {
	*gormdao.BaseCond
	TenantID     string
	PersonID     string
	UserType     model.UserType   // 账号类型过滤：member(真实用户)/machine(服务账号)
	Source       model.UserSource // 来源过滤：builtin(随租户创建内置)/manual(手工创建)
	IDs          []string         // 主键 IN 批量查询
	Username     string
	PrimaryEmail string
	PrimaryPhone string
	Name         string
	Status       model.UserStatus
	OwnerType    model.OwnerType
	Keyword      string // 模糊搜索: 租户内姓名 LIKE 或关联 person 的 username/email/phone LIKE
}

func (c *UserCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.PersonID != "" {
		db.Where(clause.Eq{Column: col("person_id"), Value: c.PersonID})
	}
	if c.UserType != "" {
		db.Where(clause.Eq{Column: col("user_type"), Value: c.UserType})
	}
	if c.Source != "" {
		db.Where(clause.Eq{Column: col("source"), Value: c.Source})
	}
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
	if c.OwnerType != "" {
		db.Where(clause.Eq{Column: col("owner_type"), Value: c.OwnerType})
	}
	if c.Keyword != "" {
		k := "%" + c.Keyword + "%"
		// 姓名 LIKE，或关联 person 的 username/email/phone LIKE。
		// 子查询在 OR 分支里用 clause.Expr 表达；表名/列名经 quoteCol 按方言引用。
		db.Where(clause.Or(
			clause.Like{Column: col("name"), Value: k},
			clause.Expr{
				SQL: quoteCol(db, tableName, "person_id") +
					" IN (SELECT id FROM " + quoteIdent(db, model.TableNamePerson) +
					" WHERE username LIKE ? OR primary_email LIKE ? OR primary_phone LIKE ?)",
				Vars: []interface{}{k, k, k},
			},
		))
	}
}

type UserDao struct {
	*gormdao.Dao[model.UserEntity, model.UserEntityList, string]
}

func NewUserDao(opts ...DaoOption) *UserDao {
	return &UserDao{
		Dao: gormdao.NewDao[model.UserEntity, model.UserEntityList, string](
			model.TableNameUser, "UserDao",
			resolveDBGetter(opts...),
		),
	}
}
