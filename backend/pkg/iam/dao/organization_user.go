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
	UserIDs        []string
	RelationType   model.OrgUserRelationType
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
	if len(c.UserIDs) > 0 {
		db.Where(tableName+".user_id IN ?", c.UserIDs)
	}
	if c.RelationType != "" {
		db.Where(tableName+".relation_type = ?", c.RelationType)
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
