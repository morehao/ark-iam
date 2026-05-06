package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type UserIdentityCond struct {
	*genericdao.BaseCond
	TenantID   uint
	UserID     uint
	Issuer     string
	IdentityID string
}

func (c *UserIdentityCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.UserID != 0 {
		db.Where(tableName + ".user_id = ?", c.UserID)
	}
	if c.Issuer != "" {
		db.Where(tableName + ".issuer = ?", c.Issuer)
	}
	if c.IdentityID != "" {
		db.Where(tableName + ".identity_id = ?", c.IdentityID)
	}
}

type UserIdentityDao struct {
	*genericdao.GenericDao[model.UserIdentityEntity, model.UserIdentityEntityList]
}

func NewUserIdentityDao() *UserIdentityDao {
	return &UserIdentityDao{
		GenericDao: genericdao.NewGenericDao[model.UserIdentityEntity, model.UserIdentityEntityList](
			model.TableNameUserIdentity, "UserIdentityDao",
			dbclient.IamDB,
		),
	}
}