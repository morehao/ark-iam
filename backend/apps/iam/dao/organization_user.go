package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type OrganizationUserCond struct {
	*genericdao.BaseCond
	TenantID       uint
	OrganizationID uint
	UserID         uint
}

func (c *OrganizationUserCond) BuildCondition(db *gorm.DB, tableName string) {
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

type OrganizationUserDao struct {
	*genericdao.GenericDao[model.OrganizationUserEntity, model.OrganizationUserEntityList]
}

func NewOrganizationUserDao() *OrganizationUserDao {
	return &OrganizationUserDao{
		GenericDao: genericdao.NewGenericDao[model.OrganizationUserEntity, model.OrganizationUserEntityList](
			model.TableNameOrganizationUser, "OrganizationUserDao",
			dbclient.IamDB,
		),
	}
}