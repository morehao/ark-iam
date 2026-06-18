package dao

import (
	"context"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type RefreshTokenCond struct {
	*genericdao.BaseCond
	TenantID     uint
	UserID       uint
	Token        string
}

func (c *RefreshTokenCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.UserID != 0 {
		db.Where(tableName + ".user_id = ?", c.UserID)
	}
	if c.Token != "" {
		db.Where(tableName + ".token = ?", c.Token)
	}
}

type RefreshTokenDao struct {
	*genericdao.GenericDao[model.RefreshTokenEntity, model.RefreshTokenEntityList]
}

func NewRefreshTokenDao() *RefreshTokenDao {
	return &RefreshTokenDao{
		GenericDao: genericdao.NewGenericDao[model.RefreshTokenEntity, model.RefreshTokenEntityList](
			model.TableNameRefreshToken, "RefreshTokenDao",
			dbclient.IamDB,
		),
	}
}

func (d *RefreshTokenDao) RevokeByPersonID(ctx context.Context, personID uint) error {
	now := time.Now()
	return dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).
		Where("person_id = ?", personID).
		Where("revoked_at IS NULL").
		Update("revoked_at", &now).Error
}
