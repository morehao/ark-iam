package dao

import (
	"context"
	"errors"
	"time"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DomainCond struct {
	*gormdao.BaseCond
	TenantID       string
	Domain         string
	DomainRole     model.DomainRole
	DomainUsage    model.DomainUsage
	Status         model.DomainStatus
	TargetDomainID string
}

func (c *DomainCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.Domain != "" {
		db.Where(clause.Like{Column: col("domain"), Value: "%" + c.Domain + "%"})
	}
	if c.DomainRole != "" {
		db.Where(clause.Eq{Column: col("domain_role"), Value: c.DomainRole})
	}
	if c.DomainUsage != "" {
		db.Where(clause.Eq{Column: col("domain_usage"), Value: c.DomainUsage})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
	if c.TargetDomainID != "" {
		db.Where(clause.Eq{Column: col("target_domain_id"), Value: c.TargetDomainID})
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

// UpdateFieldsWithTx 在调用方事务内按 id 局部更新指定列，返回命中的行数。
//
// 供域名状态机（pkg/core/tenantdomain）在 SELECT ... FOR UPDATE 后做「带 RowsAffected 的
// 条件更新」：0 行视为并发冲突/前置状态不满足（DG-CORE-4 / redesign §4.5）。
// 只允许更新给定 columns（列名白名单由调用方保证），不写 JSON 列以外语义。
func (dao *DomainDao) UpdateFieldsWithTx(ctx context.Context, tx *gorm.DB, id string, columns map[string]any) (int64, error) {
	res := dao.WithTx(tx).DB(ctx).Model(&model.DomainEntity{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(columns)
	return res.RowsAffected, res.Error
}

// CompareAndSetStatusWithTx 条件状态迁移：仅当当前 status==from 时改为 to，返回是否命中。
// 与部分唯一索引配合：索引兜底、此处做前置状态校验（0 行=并发已被他人迁移）。
func (dao *DomainDao) CompareAndSetStatusWithTx(ctx context.Context, tx *gorm.DB, id string, from, to model.DomainStatus, extra map[string]any) (bool, error) {
	columns := map[string]any{"status": to}
	for k, v := range extra {
		columns[k] = v
	}
	res := dao.WithTx(tx).DB(ctx).Model(&model.DomainEntity{}).
		Where("id = ? AND status = ? AND deleted_at IS NULL", id, from).
		Updates(columns)
	return res.RowsAffected == 1, res.Error
}

// LockByIDWithTx 在调用方事务内 SELECT ... FOR UPDATE 锁单行（PG 生效；SQLite 单测不模拟锁语义）。
// 锁 iam_tenant 行的主入口在 tenantdomain 用 raw SQL 实现；此处提供 domain 行级锁辅助。
func (dao *DomainDao) LockByIDWithTx(ctx context.Context, tx *gorm.DB, id string) (*model.DomainEntity, error) {
	var entity model.DomainEntity
	err := dao.WithTx(tx).DB(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// GetByDomainCrossTenant 按归一化 host 跨全部租户查未删除的域名行（Host 解析入口用）。
// host 全局由 (domain) partial unique 约束（released 释放），故跨租户唯一可定位。
func (dao *DomainDao) GetByDomainCrossTenant(ctx context.Context, domain string) (*model.DomainEntity, error) {
	var entity model.DomainEntity
	err := dao.DB(dbclient.CrossTenantContext(ctx)).Model(&model.DomainEntity{}).
		Where("domain = ?", domain).First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}
