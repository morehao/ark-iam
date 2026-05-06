package dao

import (
	"context"

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

func (dao *UserIdentityDao) GetByIssuerAndIdentityID(ctx context.Context, tenantID uint, issuer, identityID string) (*model.UserIdentityEntity, error) {
	db := dbclient.IamDB(ctx)
	var entity model.UserIdentityEntity
	err := db.
		Where("tenant_id = ? AND issuer = ? AND identity_id = ? AND deleted_at IS NULL", tenantID, issuer, identityID).
		First(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}