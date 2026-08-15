package dao

import (
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type ApiKeyCond struct {
	*gormdao.BaseCond
	TenantID  uint
	Name      string
	KeyHash   string
	RevokedAt *time.Time
}

func (c *ApiKeyCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.Name != "" {
		db.Where(tableName+".name LIKE ?", "%"+c.Name+"%")
	}
	if c.KeyHash != "" {
		db.Where(tableName+".key_hash = ?", c.KeyHash)
	}
	if c.RevokedAt != nil {
		if c.RevokedAt.IsZero() {
			db.Where(tableName + ".revoked_at IS NULL")
		} else {
			db.Where(tableName+".revoked_at = ?", c.RevokedAt)
		}
	}
}

type ApiKeyDao struct {
	*gormdao.Dao[model.ApiKeyEntity, model.ApiKeyEntityList, uint]
}

func NewApiKeyDao(opts ...DaoOption) *ApiKeyDao {
	return &ApiKeyDao{
		Dao: gormdao.NewDao[model.ApiKeyEntity, model.ApiKeyEntityList, uint](
			model.TableNameApiKey, "ApiKeyDao",
			resolveDBGetter(opts...),
		),
	}
}
