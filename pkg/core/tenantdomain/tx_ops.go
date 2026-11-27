package tenantdomain

import (
	"context"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/gorm"
)

// Tx 封装一次域名写动作的事务与作用域。
//
// 调用约定（DG-CORE-2）：应用层用 dbclient.IamDB(ctx).Transaction(...) 开事务，
// 在回调里构造 Tx{DB: tx, ...} 调用下方方法；**所有**写域名的 DAO 调用都用 WithTx(tx)。
// 本结构与方法都不 import gin/identity，只收 context.Context（DG-CORE-3）。
type Tx struct {
	DB       *gorm.DB
	TenantID string
	Operator string // 操作人 id（审计；应用层从 DTO.Operator 传入，不经 ctx）
	// BasePath 部署挂载前缀（空或 "/iam"，不含 /oidc），用于派生 issuer；由应用层从部署配置注入。
	BasePath   string
	Invalidate HostCacheInvalidator // 可选：提交后删 host 缓存
}

// domainDao 返回本 tx 的 domain DAO（持 tx）。
func (t *Tx) domainDao() *dao.DomainDao { return dao.NewDomainDao() }

// scope 返回目标租户作用域 ctx（对 tenant-scoped 的 iam_domain 读写必须显式声明）。
func (t *Tx) scope(ctx context.Context) context.Context {
	return dbclient.ExplicitTenantContext(ctx, t.TenantID)
}

// lockTenantRow 用 SELECT ... FOR UPDATE 锁 iam_tenant 行（PG 生效；SQLite 单测不模拟锁语义）。
// iam_tenant 在 skip 表（无 tenant_id），用 CrossTenantContext。返回 tenant 实体。
func (t *Tx) lockTenantRow(ctx context.Context) (*model.TenantEntity, error) {
	var entity model.TenantEntity
	err := t.DB.WithContext(dbclient.CrossTenantContext(ctx)).
		Clauses(lockClause()).
		Where("id = ? AND deleted_at IS NULL", t.TenantID).
		First(&entity).Error
	if err != nil {
		if isNotFound(err) {
			return nil, ErrTenantNotExist
		}
		return nil, err
	}
	return &entity, nil
}

// loadDomain 在目标租户作用域内按 id 取域名行（未删除）。
func (t *Tx) loadDomain(ctx context.Context, domainID string) (*model.DomainEntity, error) {
	entity, err := t.domainDao().WithTx(t.DB).GetByID(t.scope(ctx), domainID)
	if err != nil {
		return nil, err
	}
	if entity == nil || entity.TenantID != t.TenantID {
		return nil, ErrDomainNotExist
	}
	return entity, nil
}

// updateTenantIssuer 更新 iam_tenant 的派生缓存 issuer/status/domain_epoch（+1）与 updated_by。
// iam_tenant 在 skip 表，用 CrossTenantContext。
func (t *Tx) updateTenantIssuer(ctx context.Context, issuer string, status model.TenantStatus) error {
	return t.DB.WithContext(dbclient.CrossTenantContext(ctx)).
		Model(&model.TenantEntity{}).
		Where("id = ? AND deleted_at IS NULL", t.TenantID).
		Updates(map[string]any{
			"issuer":       issuer,
			"status":       status,
			"domain_epoch": gorm.Expr("domain_epoch + 1"),
			"updated_by":   t.Operator,
		}).Error
}

// bumpEpochOnly 只递增 domain_epoch（不动 issuer/status，用于 alias 变更等不影响 issuer 的场景）。
func (t *Tx) bumpEpochOnly(ctx context.Context) error {
	return t.DB.WithContext(dbclient.CrossTenantContext(ctx)).
		Model(&model.TenantEntity{}).
		Where("id = ? AND deleted_at IS NULL", t.TenantID).
		Update("domain_epoch", gorm.Expr("domain_epoch + 1")).Error
}

// cascadeAliasesOfPrimary 把指向 primaryID 的 active/disabled alias 置 disabled，
// 保留 target_domain_id、不动 retired/released 历史行（不变量 3 / §4 规则）。
// 返回受影响 alias 的 host 列表（供提交后清缓存）。
func (t *Tx) cascadeAliasesOfPrimary(ctx context.Context, primaryID string) ([]string, error) {
	scope := t.scope(ctx)
	aliases, err := t.domainDao().WithTx(t.DB).GetListByCond(scope, &dao.DomainCond{
		TenantID:       t.TenantID,
		TargetDomainID: primaryID,
	})
	if err != nil {
		return nil, err
	}
	var hosts []string
	for i := range aliases {
		a := aliases[i]
		if a.Status != model.DomainStatusActive && a.Status != model.DomainStatusDisabled {
			continue // 不动 retired/released 历史行
		}
		ok, err := t.domainDao().CompareAndSetStatusWithTx(scope, t.DB, a.ID, a.Status, model.DomainStatusDisabled,
			map[string]any{"updated_by": t.Operator})
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrConflict
		}
		hosts = append(hosts, a.Domain)
	}
	return hosts, nil
}
