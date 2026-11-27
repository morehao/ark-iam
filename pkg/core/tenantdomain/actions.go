package tenantdomain

import (
	"context"
	"time"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
)

// 本文件实现域名写动作。**每个**动作都在调用方已开的 tx 内执行（t.DB 是 tx），
// 返回需要「提交后」清 host 缓存的 host 列表（调用方在 Transaction 返回 nil 后调用
// HostCacheInvalidator，避免事务未提交就删缓存导致读穿透到旧值）。
//
// 动作内部统一顺序（DG-CORE-2 / redesign §4.5）：
//   1. lockTenantRow 锁 iam_tenant 行（PG FOR UPDATE）；
//   2. loadDomain 取目标租户作用域下的域名行；
//   3. assert* 纯状态机校验（statemachine.go）；
//   4. CompareAndSetStatusWithTx 条件落库（0 行=并发冲突 ErrConflict，DG-CORE-4）；
//   5. 若影响 issuer：updateTenantIssuer（issuer + status + epoch+1）；
//   6. 级联 alias（primary disable/retire 时）。
// 不在此处 commit，也不在此处清缓存。

// Verify 域控证明通过：draft → verified，写 verified_at/verification_method/token。返回 hosts（含本域）。
func (t *Tx) Verify(ctx context.Context, domainID string, method model.DomainVerificationMethod, token string, now time.Time) ([]string, error) {
	tenant, err := t.lockTenantRow(ctx)
	if err != nil {
		return nil, err
	}
	_ = tenant
	d, err := t.loadDomain(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if err := assertVerify(d); err != nil {
		return nil, err
	}
	ok, err := t.domainDao().CompareAndSetStatusWithTx(t.scope(ctx), t.DB, d.ID, model.DomainStatusDraft, model.DomainStatusVerified,
		map[string]any{
			"verified_at":         now,
			"verification_method": method,
			"verification_token":  token,
			"updated_by":          t.Operator,
		})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConflict
	}
	return []string{d.Domain}, nil
}

// TLSReady 证书就绪：verified → tls_ready，写 tls_ready_at。返回 hosts。
func (t *Tx) TLSReady(ctx context.Context, domainID string, now time.Time) ([]string, error) {
	if _, err := t.lockTenantRow(ctx); err != nil {
		return nil, err
	}
	d, err := t.loadDomain(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if err := assertTLSReady(d); err != nil {
		return nil, err
	}
	ok, err := t.domainDao().CompareAndSetStatusWithTx(t.scope(ctx), t.DB, d.ID, model.DomainStatusVerified, model.DomainStatusTLSReady,
		map[string]any{"tls_ready_at": now, "updated_by": t.Operator})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConflict
	}
	return []string{d.Domain}, nil
}

// Activate 激活域名。primary 与 alias 共用入口，按角色/usage 分派：
//   - primary：tls_ready → active；校验 usage=login/both、无其他 active primary；
//     激活同事务把 tenant 置 active、写 issuer=派生自本域、epoch+1。
//   - alias(login/both)：tls_ready → active；target 必须同租户且 active。
//   - alias(email)：verified → active；target 必须同租户且 active（P0 不消费登录）。
//
// 返回需清除的 host 列表。
func (t *Tx) Activate(ctx context.Context, domainID string) ([]string, error) {
	tenant, err := t.lockTenantRow(ctx)
	if err != nil {
		return nil, err
	}
	d, err := t.loadDomain(ctx, domainID)
	if err != nil {
		return nil, err
	}

	switch {
	case d.DomainRole == model.DomainRolePrimary:
		if err := assertPrimaryActivate(d); err != nil {
			return nil, err
		}
		// tenant 必须是 provisioning/active（不能从 suspended 直接激活）。
		if tenant.Status != model.TenantStatusProvisioning && tenant.Status != model.TenantStatusActive {
			return nil, ErrTenantStatusInvalid
		}
		// 无其他 active primary（DB 部分唯一索引也兜底）。
		actives, err := t.domainDao().WithTx(t.DB).GetListByCond(t.scope(ctx), &dao.DomainCond{TenantID: t.TenantID, DomainRole: model.DomainRolePrimary, Status: model.DomainStatusActive})
		if err != nil {
			return nil, err
		}
		if len(actives) > 0 {
			return nil, ErrAlreadyActivePrimary
		}
		ok, err := t.domainDao().CompareAndSetStatusWithTx(t.scope(ctx), t.DB, d.ID, model.DomainStatusTLSReady, model.DomainStatusActive,
			map[string]any{"updated_by": t.Operator})
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrConflict
		}
		// tenant active + issuer=派生自本域 host。
		issuer := DeriveIssuer(d.Domain, t.BasePath)
		if err := t.updateTenantIssuer(ctx, issuer, model.TenantStatusActive); err != nil {
			return nil, err
		}
		return []string{d.Domain}, nil

	case d.DomainRole == model.DomainRoleAlias && d.DomainUsage == model.DomainUsageEmail:
		if err := assertEmailAliasActivate(d); err != nil {
			return nil, err
		}
		if err := t.assertAliasTargetActive(ctx, d); err != nil {
			return nil, err
		}
		ok, err := t.domainDao().CompareAndSetStatusWithTx(t.scope(ctx), t.DB, d.ID, model.DomainStatusVerified, model.DomainStatusActive,
			map[string]any{"updated_by": t.Operator})
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrConflict
		}
		if err := t.bumpEpochOnly(ctx); err != nil {
			return nil, err
		}
		return []string{d.Domain}, nil

	default: // alias login/both
		if err := assertAliasActivate(d); err != nil {
			return nil, err
		}
		if err := t.assertAliasTargetActive(ctx, d); err != nil {
			return nil, err
		}
		ok, err := t.domainDao().CompareAndSetStatusWithTx(t.scope(ctx), t.DB, d.ID, model.DomainStatusTLSReady, model.DomainStatusActive,
			map[string]any{"updated_by": t.Operator})
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrConflict
		}
		if err := t.bumpEpochOnly(ctx); err != nil {
			return nil, err
		}
		return []string{d.Domain}, nil
	}
}

// Disable 停用：仅 alias active → disabled。primary 不允许直接 disable（只能走 offboard）。
func (t *Tx) Disable(ctx context.Context, domainID string) ([]string, error) {
	if _, err := t.lockTenantRow(ctx); err != nil {
		return nil, err
	}
	d, err := t.loadDomain(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if err := assertDisable(d); err != nil {
		return nil, err
	}
	ok, err := t.domainDao().CompareAndSetStatusWithTx(t.scope(ctx), t.DB, d.ID, model.DomainStatusActive, model.DomainStatusDisabled,
		map[string]any{"updated_by": t.Operator})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConflict
	}
	return []string{d.Domain}, nil
}

// Enable 恢复：仅 alias disabled → active（target 必须 active）。
func (t *Tx) Enable(ctx context.Context, domainID string) ([]string, error) {
	if _, err := t.lockTenantRow(ctx); err != nil {
		return nil, err
	}
	d, err := t.loadDomain(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if err := assertEnable(d); err != nil {
		return nil, err
	}
	if err := t.assertAliasTargetActive(ctx, d); err != nil {
		return nil, err
	}
	ok, err := t.domainDao().CompareAndSetStatusWithTx(t.scope(ctx), t.DB, d.ID, model.DomainStatusDisabled, model.DomainStatusActive,
		map[string]any{"updated_by": t.Operator})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConflict
	}
	return []string{d.Domain}, nil
}

// Retire 退役：
//   - primary：disabled → retired（offboard 之后）；清空 issuer 已在 offboard 做过，这里只置墓碑；
//   - alias：active/disabled → retired。
//
// 写 retired_at/retire_reason。primary 退役会级联把其 alias 置 disabled（不变量）。
func (t *Tx) Retire(ctx context.Context, domainID, reason string, now time.Time) ([]string, error) {
	tenant, err := t.lockTenantRow(ctx)
	if err != nil {
		return nil, err
	}
	d, err := t.loadDomain(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if err := assertRetire(d); err != nil {
		return nil, err
	}
	from := d.Status
	ok, err := t.domainDao().CompareAndSetStatusWithTx(t.scope(ctx), t.DB, d.ID, from, model.DomainStatusRetired,
		map[string]any{"retired_at": now, "retire_reason": reason, "updated_by": t.Operator})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConflict
	}
	hosts := []string{d.Domain}
	if d.DomainRole == model.DomainRolePrimary {
		// primary 退役要求 tenant 已 suspended（offboard 之后）。
		if tenant.Status != model.TenantStatusSuspended {
			return nil, ErrTenantStatusInvalid
		}
		aliased, err := t.cascadeAliasesOfPrimary(ctx, d.ID)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, aliased...)
	}
	return hosts, nil
}

// Release 正式释放：retired → released（满 90 天 + 审批单）。
// primary release 要求 tenant=suspended（issuer 已空）；alias release 不动 tenant。
func (t *Tx) Release(ctx context.Context, domainID, approvalTicket string, now time.Time) ([]string, error) {
	tenant, err := t.lockTenantRow(ctx)
	if err != nil {
		return nil, err
	}
	d, err := t.loadDomain(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if err := assertRelease(d, approvalTicket, now); err != nil {
		return nil, err
	}
	if d.DomainRole == model.DomainRolePrimary && tenant.Status != model.TenantStatusSuspended {
		return nil, ErrTenantStatusInvalid
	}
	ok, err := t.domainDao().CompareAndSetStatusWithTx(t.scope(ctx), t.DB, d.ID, model.DomainStatusRetired, model.DomainStatusReleased,
		map[string]any{"released_at": now, "release_approval": approvalTicket, "updated_by": t.Operator})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConflict
	}
	return []string{d.Domain}, nil
}

// CancelDelete 取消/删除未上线域名：draft/verified/tls_ready → released（不发审批单）。
// 有未 released 的 alias 引用时拒绝（先处理 alias）。
func (t *Tx) CancelDelete(ctx context.Context, domainID string, now time.Time) ([]string, error) {
	if _, err := t.lockTenantRow(ctx); err != nil {
		return nil, err
	}
	d, err := t.loadDomain(ctx, domainID)
	if err != nil {
		return nil, err
	}
	if err := assertCancelDelete(d); err != nil {
		return nil, err
	}
	// 有未 released 的 alias 引用则拒绝。
	refs, err := t.domainDao().WithTx(t.DB).GetListByCond(t.scope(ctx), &dao.DomainCond{TenantID: t.TenantID, TargetDomainID: d.ID})
	if err != nil {
		return nil, err
	}
	for i := range refs {
		if refs[i].Status != model.DomainStatusReleased {
			return nil, ErrReferencedByAlias
		}
	}
	from := d.Status
	ok, err := t.domainDao().CompareAndSetStatusWithTx(t.scope(ctx), t.DB, d.ID, from, model.DomainStatusReleased,
		map[string]any{"released_at": now, "updated_by": t.Operator})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrConflict
	}
	return []string{d.Domain}, nil
}

// assertAliasTargetActive 校验 alias 的 target 是同租户、active 的 primary。
func (t *Tx) assertAliasTargetActive(ctx context.Context, d *model.DomainEntity) error {
	if d.TargetDomainID == "" || d.TargetDomainID == d.ID {
		return ErrTargetMismatch
	}
	target, err := t.domainDao().WithTx(t.DB).GetByID(t.scope(ctx), d.TargetDomainID)
	if err != nil {
		return err
	}
	if target == nil || target.TenantID != t.TenantID || target.DomainRole != model.DomainRolePrimary {
		return ErrTargetMismatch
	}
	if target.Status != model.DomainStatusActive {
		return ErrTargetNotActive
	}
	return nil
}

// DeriveIssuer 由 primary host + basePath 派生 issuer：https://<host><basePath>/oidc。
// basePath 为空或 "/iam"（不含 /oidc），示例用占位（部署值由运维配置）。
func DeriveIssuer(host, basePath string) string {
	return "https://" + host + basePath + "/oidc"
}

// Offboard 是 P0 primary 的**唯一**停用入口（租户 offboard）：在调用方事务内
//   - tenant → suspended，清空 issuer，domain_epoch + 1；
//   - active primary → disabled；
//   - 指向 primary 的 active/disabled alias → disabled（不变量级联）；
//   - 所有未上线域名（draft/verified/tls_ready）→ released（取消，不填 release_approval）。
//
// 返回需要清缓存的 host 列表。session/refresh 撤销由应用层在本事务提交后异步执行（可重试）。
// 依据 p0-tenant-lifecycle-contract §3「offboard」。
func (t *Tx) Offboard(ctx context.Context, now time.Time) ([]string, error) {
	tenant, err := t.lockTenantRow(ctx)
	if err != nil {
		return nil, err
	}
	if tenant.IsolationMode != model.TenantIsolationDedicated {
		return nil, ErrTenantStatusInvalid
	}
	// 只允许 provisioning / active → suspended。
	if tenant.Status != model.TenantStatusActive && tenant.Status != model.TenantStatusProvisioning {
		return nil, ErrTenantStatusInvalid
	}

	scope := t.scope(ctx)
	repo := t.domainDao()
	all, err := repo.WithTx(t.DB).GetListByCond(scope, &dao.DomainCond{TenantID: t.TenantID})
	if err != nil {
		return nil, err
	}
	var hosts []string
	// 先处理未上线域名 → released（取消）。
	for i := range all {
		d := all[i]
		switch d.Status {
		case model.DomainStatusDraft, model.DomainStatusVerified, model.DomainStatusTLSReady:
			ok, e := repo.CompareAndSetStatusWithTx(scope, t.DB, d.ID, d.Status, model.DomainStatusReleased,
				map[string]any{"released_at": now, "updated_by": t.Operator})
			if e != nil {
				return nil, e
			}
			if !ok {
				return nil, ErrConflict
			}
			hosts = append(hosts, d.Domain)
		case model.DomainStatusActive:
			if d.DomainRole == model.DomainRolePrimary {
				ok, e := repo.CompareAndSetStatusWithTx(scope, t.DB, d.ID, model.DomainStatusActive, model.DomainStatusDisabled,
					map[string]any{"updated_by": t.Operator})
				if e != nil {
					return nil, e
				}
				if !ok {
					return nil, ErrConflict
				}
				hosts = append(hosts, d.Domain)
				// 级联其 alias
				aliased, e := t.cascadeAliasesOfPrimary(ctx, d.ID)
				if e != nil {
					return nil, e
				}
				hosts = append(hosts, aliased...)
			} else {
				// active alias → disabled
				ok, e := repo.CompareAndSetStatusWithTx(scope, t.DB, d.ID, model.DomainStatusActive, model.DomainStatusDisabled,
					map[string]any{"updated_by": t.Operator})
				if e != nil {
					return nil, e
				}
				if !ok {
					return nil, ErrConflict
				}
				hosts = append(hosts, d.Domain)
			}
		}
	}
	// tenant → suspended，清 issuer，epoch+1。
	if err := t.updateTenantIssuer(ctx, "", model.TenantStatusSuspended); err != nil {
		return nil, err
	}
	return hosts, nil
}
