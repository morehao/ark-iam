package dao

import (
	"context"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
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
	*gormdao.Dao[model.ApiKeyEntity, model.ApiKeyEntityList]
	dbGetter gormdao.DBGetter
}

func NewApiKeyDao() *ApiKeyDao {
	return &ApiKeyDao{
		Dao: gormdao.NewDao[model.ApiKeyEntity, model.ApiKeyEntityList](
			model.TableNameApiKey, "ApiKeyDao",
			dbclient.IamDB,
		),
		dbGetter: dbclient.IamDB,
	}
}

func NewApiKeyDaoWithDB(dbGetter gormdao.DBGetter) *ApiKeyDao {
	return &ApiKeyDao{
		Dao: gormdao.NewDao[model.ApiKeyEntity, model.ApiKeyEntityList](
			model.TableNameApiKey, "ApiKeyDao",
			dbGetter,
		),
		dbGetter: dbGetter,
	}
}

func (d *ApiKeyDao) Delete(ctx context.Context, id uint, deletedBy uint) error {
	return d.dbGetter(ctx).Model(&model.ApiKeyEntity{}).Table(d.TableName).Where("id = ?", id).
		Updates(map[string]any{"deleted_by": deletedBy, "deleted_at": time.Now()}).Error
}
