package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type OrganizationRoleUserRelationCond struct {
	*genericdao.BaseCond
	TenantID           uint
	OrganizationID     uint
	OrganizationRoleID uint
	UserID             uint
}

func (c *OrganizationRoleUserRelationCond) BuildCondition(db *gorm.DB, tableName string) {
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

type OrganizationRoleUserRelationDao struct {
	*genericdao.GenericDao[model.OrganizationRoleUserRelationEntity, model.OrganizationRoleUserRelationEntityList]
}

func NewOrganizationRoleUserRelationDao() *OrganizationRoleUserRelationDao {
	return &OrganizationRoleUserRelationDao{
		GenericDao: genericdao.NewGenericDao[model.OrganizationRoleUserRelationEntity, model.OrganizationRoleUserRelationEntityList](
			model.TableNameOrganizationRoleUserRelation, "OrganizationRoleUserRelationDao",
			dbclient.IamDB,
		),
	}
}