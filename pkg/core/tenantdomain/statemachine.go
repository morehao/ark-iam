package tenantdomain

import (
	"time"

	"github.com/morehao/ark-iam/pkg/model"
)

// 本文件实现域名状态机的**纯判定**（给定 domain 行 + 动作 → 校验前置，不改库）。
// 实际写库在 tx_ops.go 的 Tx 方法里，先调这里校验、再 CompareAndSetStatusWithTx 落库。
// 状态机见 p0-tenant-lifecycle-contract.md §4 / redesign §3.2。

// assertPrimaryActivate 校验 primary activate 前置：
//   - from 必须是 tls_ready（email-only 例外见 Activate，仍走 tls_ready 之外的分支）；
//   - usage 必须 login/both（primary 不能用 email）；
//   - target_domain_id 必须为空（primary 无 target）。
func assertPrimaryActivate(d *model.DomainEntity) error {
	if d.DomainRole != model.DomainRolePrimary {
		return ErrInvalidTransition
	}
	if d.TargetDomainID != "" {
		return ErrTargetMismatch
	}
	if !serviceLoginUsage(d.DomainUsage) {
		return ErrPrimaryUsageInvalid
	}
	if d.Status != model.DomainStatusTLSReady {
		return ErrInvalidTransition
	}
	return nil
}

// assertAliasActivate 校验 alias activate（login/both）前置：
//   - from 必须是 tls_ready；
//   - target_domain_id 必填（D alias 必须指向同租户 primary）；
//   - target 的实际 active/同租户校验需查库，由 Tx.Activate 在锁内完成（这里只做静态校验）。
func assertAliasActivate(d *model.DomainEntity) error {
	if d.DomainRole != model.DomainRoleAlias {
		return ErrInvalidTransition
	}
	if d.TargetDomainID == "" {
		return ErrTargetMismatch
	}
	if d.Status != model.DomainStatusTLSReady {
		return ErrInvalidTransition
	}
	return nil
}

// assertEmailAliasActivate 校验 email-only alias activate 前置：
//   - usage 必须是 email；
//   - from 必须是 verified（email-only 可跳过 tls_ready，见 AC-12 / §4）；
//   - D 的 email alias 必须有 target（同租户 primary）。
func assertEmailAliasActivate(d *model.DomainEntity) error {
	if d.DomainRole != model.DomainRoleAlias || d.DomainUsage != model.DomainUsageEmail {
		return ErrInvalidTransition
	}
	if d.TargetDomainID == "" {
		return ErrTargetMismatch
	}
	if d.Status != model.DomainStatusVerified {
		return ErrInvalidTransition
	}
	return nil
}

// assertVerify 校验 verify：仅 draft → verified。
func assertVerify(d *model.DomainEntity) error {
	if d.Status != model.DomainStatusDraft {
		return ErrInvalidTransition
	}
	return nil
}

// assertTLSReady 校验 tls-ready：仅 verified → tls_ready（email-only 不要求 TLS，见 §4）。
func assertTLSReady(d *model.DomainEntity) error {
	if d.Status != model.DomainStatusVerified {
		return ErrInvalidTransition
	}
	return nil
}

// assertDisable 校验 disable：alias active→disabled；primary **不允许**直接 disable（只能走 offboard）。
func assertDisable(d *model.DomainEntity) error {
	if d.DomainRole == model.DomainRolePrimary {
		// active primary 不允许直接 disable，只能经 tenant offboard（§5.1 / 沉默清单）。
		return ErrInvalidTransition
	}
	if d.Status != model.DomainStatusActive {
		return ErrInvalidTransition
	}
	return nil
}

// assertEnable 校验 alias enable：disabled→active（target 校验在 Tx 内）。
func assertEnable(d *model.DomainEntity) error {
	if d.DomainRole != model.DomainRoleAlias {
		return ErrInvalidTransition
	}
	if d.Status != model.DomainStatusDisabled {
		return ErrInvalidTransition
	}
	return nil
}

// assertRetire 校验 retire：
//   - primary：只能 disabled→retired（offboard 之后）；
//   - alias：active/disabled→retired。
func assertRetire(d *model.DomainEntity) error {
	switch d.DomainRole {
	case model.DomainRolePrimary:
		if d.Status != model.DomainStatusDisabled {
			return ErrInvalidTransition
		}
	case model.DomainRoleAlias:
		if d.Status != model.DomainStatusActive && d.Status != model.DomainStatusDisabled {
			return ErrInvalidTransition
		}
	default:
		return ErrInvalidTransition
	}
	return nil
}

// assertRelease 校验 release（正式释放）：
//   - 仅 retired→released；
//   - 必须满隔离期（从 retired_at 起 90 天）且有审批单。
func assertRelease(d *model.DomainEntity, approvalTicket string, now time.Time) error {
	if d.Status != model.DomainStatusRetired {
		return ErrInvalidTransition
	}
	if approvalTicket == "" {
		return ErrReleaseNotDue
	}
	if d.RetiredAt == nil || now.Sub(*d.RetiredAt) < ReleaseIsolationPeriod {
		return ErrReleaseNotDue
	}
	return nil
}

// assertCancelDelete 校验取消/删除（未上线域名 → released）：
//   - 仅 draft/verified/tls_ready 可取消；
//   - active/disabled/retired 不可走此路径（active 必须 disable/offboard）。
func assertCancelDelete(d *model.DomainEntity) error {
	switch d.Status {
	case model.DomainStatusDraft, model.DomainStatusVerified, model.DomainStatusTLSReady:
		return nil
	default:
		return ErrInvalidTransition
	}
}
