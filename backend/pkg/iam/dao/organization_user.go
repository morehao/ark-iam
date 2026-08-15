package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type OrganizationUserCond struct {
	*gormdao.BaseCond
	TenantID       string
	OrganizationID string
	UserID         string
	RelationType   string
	// IsPrimary 过滤主归属（仅配合 RelationType=member 使用）
	IsPrimary *bool
}

func (c *OrganizationUserCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.OrganizationID != "" {
		db.Where(tableName+".organization_id = ?", c.OrganizationID)
	}
	if c.UserID != "" {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
	if c.RelationType != "" {
		db.Where(tableName+".relation_type = ?", c.RelationType)
	}
	if c.IsPrimary != nil {
		db.Where(tableName+".is_primary = ?", *c.IsPrimary)
	}
}

type OrganizationUserDao struct {
	*gormdao.Dao[model.OrganizationUserEntity, model.OrganizationUserEntityList, string]
}

func NewOrganizationUserDao(opts ...DaoOption) *OrganizationUserDao {
	return &OrganizationUserDao{
		Dao: gormdao.NewDao[model.OrganizationUserEntity, model.OrganizationUserEntityList, string](
			model.TableNameOrganizationUser, "OrganizationUserDao",
			resolveDBGetter(opts...),
		),
	}
}
