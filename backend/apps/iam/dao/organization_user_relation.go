package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type OrganizationUserRelationCond struct {
	*genericdao.BaseCond
	TenantID       uint
	OrganizationID uint
	UserID         uint
}

func (c *OrganizationUserRelationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.OrganizationID != 0 {
		db.Where(tableName + ".organization_id = ?", c.OrganizationID)
	}
	if c.UserID != 0 {
		db.Where(tableName + ".user_id = ?", c.UserID)
	}
}

type OrganizationUserRelationDao struct {
	*genericdao.GenericDao[model.OrganizationUserRelationEntity, model.OrganizationUserRelationEntityList]
}

func NewOrganizationUserRelationDao() *OrganizationUserRelationDao {
	return &OrganizationUserRelationDao{
		GenericDao: genericdao.NewGenericDao[model.OrganizationUserRelationEntity, model.OrganizationUserRelationEntityList](
			model.TableNameOrganizationUserRelation, "OrganizationUserRelationDao",
			dbclient.IamDB,
		),
	}
}