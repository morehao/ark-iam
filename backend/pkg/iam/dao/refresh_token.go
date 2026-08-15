package dao

import (
	"context"
	"time"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type RefreshTokenCond struct {
	*gormdao.BaseCond
	TenantID string
	UserID   string
	Token    string
}

func (c *RefreshTokenCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.UserID != "" {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
	if c.Token != "" {
		db.Where(tableName+".token = ?", c.Token)
	}
}

type RefreshTokenDao struct {
	*gormdao.Dao[model.RefreshTokenEntity, model.RefreshTokenEntityList, string]
}

func NewRefreshTokenDao(opts ...DaoOption) *RefreshTokenDao {
	return &RefreshTokenDao{
		Dao: gormdao.NewDao[model.RefreshTokenEntity, model.RefreshTokenEntityList, string](
			model.TableNameRefreshToken, "RefreshTokenDao",
			resolveDBGetter(opts...),
		),
	}
}

func (d *RefreshTokenDao) RevokeByPersonID(ctx context.Context, personID string) error {
	now := time.Now()
	return dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).
		Where("person_id = ?", personID).
		Where("revoked_at IS NULL").
		Update("revoked_at", &now).Error
}
