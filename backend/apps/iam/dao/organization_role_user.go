package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type OrganizationRoleUserCond struct {
	*genericdao.BaseCond
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
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.OrganizationID != 0 {
		db.Where(tableName + ".organization_id = ?", c.OrganizationID)
	}
	if c.OrganizationRoleID != 0 {
		db.Where(tableName + ".organization_role_id = ?", c.OrganizationRoleID)
	}
	if c.UserID != 0 {
		db.Where(tableName + ".user_id = ?", c.UserID)
	}
}

type OrganizationRoleUserDao struct {
	*genericdao.GenericDao[model.OrganizationRoleUserEntity, model.OrganizationRoleUserEntityList]
}

func NewOrganizationRoleUserDao() *OrganizationRoleUserDao {
	return &OrganizationRoleUserDao{
		GenericDao: genericdao.NewGenericDao[model.OrganizationRoleUserEntity, model.OrganizationRoleUserEntityList](
			model.TableNameOrganizationRoleUser, "OrganizationRoleUserDao",
			dbclient.IamDB,
		),
	}
}