package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type OrganizationRoleUserCond struct {
	*gormdao.BaseCond
	TenantID           uint
	OrganizationID     uint
	OrganizationRoleID uint
	UserID             uint
}

func (c *OrganizationRoleUserCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.OrganizationID != 0 {
		db.Where(tableName+".organization_id = ?", c.OrganizationID)
	}
	if c.OrganizationRoleID != 0 {
		db.Where(tableName+".organization_role_id = ?", c.OrganizationRoleID)
	}
	if c.UserID != 0 {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
}

type OrganizationRoleUserDao struct {
	*gormdao.Dao[model.OrganizationRoleUserEntity, model.OrganizationRoleUserEntityList]
}

func NewOrganizationRoleUserDao(opts ...DaoOption) *OrganizationRoleUserDao {
	return &OrganizationRoleUserDao{
		Dao: gormdao.NewDao[model.OrganizationRoleUserEntity, model.OrganizationRoleUserEntityList](
			model.TableNameOrganizationRoleUser, "OrganizationRoleUserDao",
			resolveDBGetter(opts...),
		),
	}
}
