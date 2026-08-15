package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type OrganizationRoleUserCond struct {
	*gormdao.BaseCond
	TenantID           string
	OrganizationID     string
	OrganizationRoleID string
	UserID             string
}

func (c *OrganizationRoleUserCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.OrganizationID != "" {
		db.Where(tableName+".organization_id = ?", c.OrganizationID)
	}
	if c.OrganizationRoleID != "" {
		db.Where(tableName+".organization_role_id = ?", c.OrganizationRoleID)
	}
	if c.UserID != "" {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
}

type OrganizationRoleUserDao struct {
	*gormdao.Dao[model.OrganizationRoleUserEntity, model.OrganizationRoleUserEntityList, string]
}

func NewOrganizationRoleUserDao(opts ...DaoOption) *OrganizationRoleUserDao {
	return &OrganizationRoleUserDao{
		Dao: gormdao.NewDao[model.OrganizationRoleUserEntity, model.OrganizationRoleUserEntityList, string](
			model.TableNameOrganizationRoleUser, "OrganizationRoleUserDao",
			resolveDBGetter(opts...),
		),
	}
}
