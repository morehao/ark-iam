package dao

import (
	"time"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApiKeyCond struct {
	*gormdao.BaseCond
	TenantID    string
	OwnerUserID string // 归属用户id(真实用户本人或服务账号)
	Name        string
	KeyHash     string
	RevokedAt   *time.Time
}

func (c *ApiKeyCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.OwnerUserID != "" {
		db.Where(clause.Eq{Column: col("owner_user_id"), Value: c.OwnerUserID})
	}
	if c.Name != "" {
		db.Where(clause.Like{Column: col("name"), Value: "%" + c.Name + "%"})
	}
	if c.KeyHash != "" {
		db.Where(clause.Eq{Column: col("key_hash"), Value: c.KeyHash})
	}
	if c.RevokedAt != nil {
		if c.RevokedAt.IsZero() {
			db.Where(clause.Eq{Column: col("revoked_at"), Value: nil})
		} else {
			db.Where(clause.Eq{Column: col("revoked_at"), Value: c.RevokedAt})
		}
	}
}

type ApiKeyDao struct {
	*gormdao.Dao[model.ApiKeyEntity, model.ApiKeyEntityList, string]
}

func NewApiKeyDao(opts ...DaoOption) *ApiKeyDao {
	return &ApiKeyDao{
		Dao: gormdao.NewDao[model.ApiKeyEntity, model.ApiKeyEntityList, string](
			model.TableNameApiKey, "ApiKeyDao",
			resolveDBGetter(opts...),
		),
	}
}
