package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type UserCond struct {
	*gormdao.BaseCond
	TenantID     string
	PersonID     string
	UserType     model.UserType // 账号类型过滤：member(真实用户)/machine(服务账号)
	IDs          []string       // 主键 IN 批量查询
	Username     string
	PrimaryEmail string
	PrimaryPhone string
	Name         string
	IsSuspended  *bool
	Keyword      string // 模糊搜索: 租户内姓名 LIKE 或关联 person 的 username/email/phone LIKE
}

func (c *UserCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.PersonID != "" {
		db.Where(tableName+".person_id = ?", c.PersonID)
	}
	if c.UserType != "" {
		db.Where(tableName+".user_type = ?", c.UserType)
	}
	if len(c.IDs) > 0 {
		db.Where(tableName+".id IN ?", c.IDs)
	}
	if c.Username != "" {
		db.Where(tableName+".username = ?", c.Username)
	}
	if c.PrimaryEmail != "" {
		db.Where(tableName+".primary_email = ?", c.PrimaryEmail)
	}
	if c.PrimaryPhone != "" {
		db.Where(tableName+".primary_phone = ?", c.PrimaryPhone)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
	if c.IsSuspended != nil {
		db.Where(tableName+".is_suspended = ?", *c.IsSuspended)
	}
	if c.Keyword != "" {
		k := "%" + c.Keyword + "%"
		db.Where(tableName+".name LIKE ? OR "+tableName+".person_id IN (SELECT id FROM "+model.TableNamePerson+" WHERE username LIKE ? OR primary_email LIKE ? OR primary_phone LIKE ?)",
			k, k, k, k)
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
