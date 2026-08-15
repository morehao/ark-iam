package dao

import (
	"context"
	"errors"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type DomainCond struct {
	*gormdao.BaseCond
	TenantID string
	Domain   string
}

func (c *DomainCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.Domain != "" {
		db.Where(tableName+".domain LIKE ?", "%"+c.Domain+"%")
	}
}

type DomainDao struct {
	*gormdao.Dao[model.DomainEntity, model.DomainEntityList, string]
}

func NewDomainDao(opts ...DaoOption) *DomainDao {
	return &DomainDao{
		Dao: gormdao.NewDao[model.DomainEntity, model.DomainEntityList, string](
			model.TableNameDomain, "DomainDao",
			resolveDBGetter(opts...),
		),
	}
}

func (dao *DomainDao) GetByTenantAndDomain(ctx context.Context, tenantID string, domain string) (*model.DomainEntity, error) {
	db := dao.DB(ctx).Table(model.TableNameDomain)
	var entity model.DomainEntity
	err := db.
		Where("tenant_id = ? AND domain = ?", tenantID, domain).
		Where("deleted_at IS NULL").
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (dao *DomainDao) Delete(ctx context.Context, id string, deletedBy string) error {
	db := dao.DB(ctx).Table(model.TableNameDomain)
	now := time.Now()
	if err := db.Where("id = ?", id).Updates(map[string]any{
		"deleted_at": now,
		"deleted_by": deletedBy,
	}).Error; err != nil {
		return err
	}
	return nil
}
