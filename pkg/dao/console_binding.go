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

// ConsoleBindingCond Host binding 查询条件。
type ConsoleBindingCond struct {
	*gormdao.BaseCond
	Host     string
	TenantID string
	Kind     model.ConsoleBindingKind
	Status   model.ConsoleBindingStatus
}

func (c *ConsoleBindingCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.Host != "" {
		db.Where(clause.Eq{Column: col("host"), Value: c.Host})
	}
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.Kind != "" {
		db.Where(clause.Eq{Column: col("kind"), Value: c.Kind})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
}

// ConsoleBindingDao 承载 iam_console_binding 的读写。
//
// 作用域约定（DG-DDL-1）：本表保留 tenant_id 但**不登记** tenantScopeSkipTables。
//   - 读取（按 Host 反查路由）跨租户：调用方/本 DAO 内部使用 dbclient.CrossTenantContext；
//   - 写入（建/改某租户的 binding）：使用 dbclient.ExplicitTenantContext(targetTenantID)。
//
// 这样既不依赖请求 ctx 的隐式作用域（协议入口无租户时 fail-closed），也不静默放宽成跨租户写。
type ConsoleBindingDao struct {
	*gormdao.Dao[model.ConsoleBindingEntity, model.ConsoleBindingEntityList, string]
}

func NewConsoleBindingDao(opts ...DaoOption) *ConsoleBindingDao {
	return &ConsoleBindingDao{
		Dao: gormdao.NewDao[model.ConsoleBindingEntity, model.ConsoleBindingEntityList, string](
			model.TableNameConsoleBinding, "ConsoleBindingDao",
			resolveDBGetter(opts...),
		),
	}
}

// GetByHost 按 Host 反查全部未删除 binding（跨租户读取；只读路由解析用）。
// 返回多行：合并 Host（gateway/dev）可有 platform/tenant 两行，调用方按 kind/Origin 选。
func (d *ConsoleBindingDao) GetByHost(ctx context.Context, host string) (model.ConsoleBindingEntityList, error) {
	var list model.ConsoleBindingEntityList
	err := d.DB(dbclient.CrossTenantContext(ctx)).
		Model(&model.ConsoleBindingEntity{}).
		Where("host = ?", host).
		Order("kind").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// GetByHostAndKind 按 (host, kind) 取单行未删除 binding（跨租户读取）。
func (d *ConsoleBindingDao) GetByHostAndKind(ctx context.Context, host string, kind model.ConsoleBindingKind) (*model.ConsoleBindingEntity, error) {
	var entity model.ConsoleBindingEntity
	err := d.DB(dbclient.CrossTenantContext(ctx)).
		Model(&model.ConsoleBindingEntity{}).
		Where("host = ? AND kind = ?", host, kind).
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// List 按条件分页/列表（平台管理员查看 Host binding；跨租户读取）。
func (d *ConsoleBindingDao) List(ctx context.Context, cond *ConsoleBindingCond) (model.ConsoleBindingEntityList, error) {
	db := d.DB(dbclient.CrossTenantContext(ctx)).Model(&model.ConsoleBindingEntity{}).Table(model.TableNameConsoleBinding)
	cond.BuildCondition(db, model.TableNameConsoleBinding)
	var list model.ConsoleBindingEntityList
	if err := db.Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// Create 建 binding（在调用方事务内；写入目标租户作用域）。
// 必须显式 ExplicitTenantContext(binding.TenantID)，不依赖请求 ctx 作用域。
func (d *ConsoleBindingDao) Create(ctx context.Context, tx *gorm.DB, binding *model.ConsoleBindingEntity) error {
	ctx = dbclient.ExplicitTenantContext(ctx, binding.TenantID)
	return d.WithTx(tx).DB(ctx).Create(binding).Error
}

// UpdateStatus 改 binding 状态（在调用方事务内；写入目标租户作用域）。
func (d *ConsoleBindingDao) UpdateStatus(ctx context.Context, tx *gorm.DB, id, tenantID string, status model.ConsoleBindingStatus, updatedBy string) error {
	ctx = dbclient.ExplicitTenantContext(ctx, tenantID)
	return d.WithTx(tx).DB(ctx).Model(&model.ConsoleBindingEntity{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{"status": status, "updated_by": updatedBy}).Error
}

// SoftDelete 软删 binding（offboard/release 用；在调用方事务内；写入目标租户作用域）。
func (d *ConsoleBindingDao) SoftDelete(ctx context.Context, tx *gorm.DB, id, tenantID, deletedBy string) (bool, error) {
	ctx = dbclient.ExplicitTenantContext(ctx, tenantID)
	now := time.Now()
	res := d.WithTx(tx).DB(ctx).Model(&model.ConsoleBindingEntity{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{"status": model.ConsoleBindingStatusDisable, "deleted_at": now, "deleted_by": deletedBy})
	return res.RowsAffected > 0, res.Error
}
