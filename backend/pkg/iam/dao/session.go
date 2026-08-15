package dao

import (
	"context"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type SessionCond struct {
	*gormdao.BaseCond
	PersonID string
	TenantID string
	UserID   string
}

func (c *SessionCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.PersonID != "" {
		db.Where(tableName+".person_id = ?", c.PersonID)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.UserID != "" {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
}

type SessionDao struct {
	*gormdao.Dao[model.RefreshTokenEntity, model.RefreshTokenEntityList, string]
}

func NewSessionDao(opts ...DaoOption) *SessionDao {
	return &SessionDao{
		Dao: gormdao.NewDao[model.RefreshTokenEntity, model.RefreshTokenEntityList, string](
			model.TableNameRefreshToken, "SessionDao",
			resolveDBGetter(opts...),
		),
	}
}

func (c *SessionCond) RevokeAll(ctx context.Context) error {
	db := dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken)
	c.BuildCondition(db, model.TableNameRefreshToken)
	return db.Updates(map[string]any{"revoked_at": gorm.Expr("NOW()")}).Error
}
