package dao

import (
	"context"

	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type SessionCond struct {
	*genericdao.BaseCond
	PersonID uint
	TenantID uint
	UserID   uint
}

func (c *SessionCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.PersonID != 0 {
		db.Where(tableName + ".person_id = ?", c.PersonID)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.UserID != 0 {
		db.Where(tableName + ".user_id = ?", c.UserID)
	}
}

type SessionDao struct {
	*genericdao.GenericDao[model.RefreshTokenEntity, model.RefreshTokenEntityList]
}

func NewSessionDao() *SessionDao {
	return &SessionDao{
		GenericDao: genericdao.NewGenericDao[model.RefreshTokenEntity, model.RefreshTokenEntityList](
			model.TableNameRefreshToken, "SessionDao",
			dbclient.IamDB,
		),
	}
}

func (d *SessionDao) RevokeByID(ctx context.Context, id, personID, tenantID, userID uint) error {
	return d.UpdateMap(ctx, id, map[string]any{"revoked_at": gorm.Expr("NOW()")})
}

func (d *SessionDao) RevokeAllByUserID(ctx context.Context, personID, tenantID, userID uint) error {
	return dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).
		Where("person_id = ? AND tenant_id = ? AND user_id = ? AND revoked_at IS NULL", personID, tenantID, userID).
		Update("revoked_at", gorm.Expr("NOW()")).Error
}