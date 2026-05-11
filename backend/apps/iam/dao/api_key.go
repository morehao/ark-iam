package dao

import (
	"context"
	"time"

	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type ApiKeyCond struct {
	*genericdao.BaseCond
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
	*genericdao.GenericDao[model.ApiKeyEntity, model.ApiKeyEntityList]
	dbGetter genericdao.DBGetter
}

func NewApiKeyDao() *ApiKeyDao {
	return &ApiKeyDao{
		GenericDao: genericdao.NewGenericDao[model.ApiKeyEntity, model.ApiKeyEntityList](
			model.TableNameApiKey, "ApiKeyDao",
			dbclient.IamDB,
		),
		dbGetter: dbclient.IamDB,
	}
}

func NewApiKeyDaoWithDB(dbGetter genericdao.DBGetter) *ApiKeyDao {
	return &ApiKeyDao{
		GenericDao: genericdao.NewGenericDao[model.ApiKeyEntity, model.ApiKeyEntityList](
			model.TableNameApiKey, "ApiKeyDao",
			dbGetter,
		),
		dbGetter: dbGetter,
	}
}

func (d *ApiKeyDao) Insert(ctx context.Context, entity *model.ApiKeyEntity) error {
	return d.GenericDao.Insert(ctx, entity)
}

func (d *ApiKeyDao) GetByID(ctx context.Context, id uint) (*model.ApiKeyEntity, error) {
	var entity model.ApiKeyEntity
	db := d.dbGetter(ctx).Table(d.TableName)
	if err := db.Where("id = ?", id).First(&entity).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	if entity.ID == 0 {
		return nil, nil
	}
	return &entity, nil
}

func (d *ApiKeyDao) GetPageListByCond(ctx context.Context, cond *ApiKeyCond, page, pageSize int) (model.ApiKeyEntityList, int64, error) {
	var total int64
	db := d.dbGetter(ctx).Model(&model.ApiKeyEntity{}).Table(d.TableName)
	cond.BuildCondition(db, d.TableName)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list model.ApiKeyEntityList
	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (d *ApiKeyDao) UpdateLastUsedAt(ctx context.Context, id uint) error {
	return d.dbGetter(ctx).Model(&model.ApiKeyEntity{}).Table(d.TableName).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}

func (d *ApiKeyDao) Revoke(ctx context.Context, id uint) error {
	return d.dbGetter(ctx).Model(&model.ApiKeyEntity{}).Table(d.TableName).
		Where("id = ?", id).
		Update("revoked_at", time.Now()).Error
}

func (d *ApiKeyDao) Delete(ctx context.Context, id uint, deletedBy uint) error {
	db := d.dbGetter(ctx).Model(&model.ApiKeyEntity{}).Table(d.TableName).Where("id = ?", id)
	if err := db.Update("deleted_by", deletedBy).Error; err != nil {
		return err
	}
	return db.Delete(&model.ApiKeyEntity{}).Error
}
