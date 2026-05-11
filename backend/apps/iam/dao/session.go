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

func (c *SessionCond) RevokeAll(ctx context.Context) error {
	db := dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken)
	c.BuildCondition(db, model.TableNameRefreshToken)
	return db.Updates(map[string]any{"revoked_at": gorm.Expr("NOW()")}).Error
}