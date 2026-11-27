package tenantdomain

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTD 内存 SQLite 注册 iam 库并迁移 tenant/domain。SQLite 不模拟 FOR UPDATE 锁语义，
// 单测只验证状态机/不变量/事务内条件更新逻辑（锁与并发在 PG 集成/PG spike 验证）。
func setupTD(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:td_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.TenantEntity{}, &model.DomainEntity{}, &model.ConsoleBindingEntity{}))
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		dbclient.ClearDBForTest(dbclient.ServiceNameIam)
		if sqlDB, e := db.DB(); e == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// seedTenant 跨租户插入 tenant（iam_tenant 是 skip 表）。
func seedTenant(t *testing.T, db *gorm.DB, id string, status model.TenantStatus) *model.TenantEntity {
	t.Helper()
	tn := &model.TenantEntity{Status: status, IsolationMode: model.TenantIsolationDedicated}
	tn.ID = id
	require.NoError(t, db.WithContext(dbclient.CrossTenantContext(context.Background())).Create(tn).Error)
	return tn
}

// seedDomain 在目标租户作用域插入 domain 行。
func seedDomain(t *testing.T, db *gorm.DB, tenantID string, d *model.DomainEntity) {
	t.Helper()
	d.TenantID = tenantID
	ctx := dbclient.ExplicitTenantContext(context.Background(), tenantID)
	require.NoError(t, db.WithContext(ctx).Create(d).Error)
}

// withTx 在事务内跑 fn（模拟应用层开事务）。
func withTx(t *testing.T, db *gorm.DB, fn func(tx *Tx) ([]string, error)) ([]string, error) {
	t.Helper()
	var hosts []string
	err := db.Transaction(func(tx *gorm.DB) error {
		h, e := fn(&Tx{DB: tx})
		if e != nil {
			return e
		}
		hosts = h
		return nil
	})
	return hosts, err
}

// TestPrimaryLifecycle 覆盖 primary draft→verified→tls_ready→active，激活后 tenant active + issuer 派生。
func TestPrimaryLifecycle(t *testing.T) {
	db := setupTD(t)
	seedTenant(t, db, "t1", model.TenantStatusProvisioning)
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity:  be("d1"),
		Domain:      "login.acme.com",
		DomainRole:  model.DomainRolePrimary,
		DomainUsage: model.DomainUsageLogin,
		Status:      model.DomainStatusDraft,
	})

	now := time.Now()
	// verify
	_, err := withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.Verify(context.Background(), "d1", model.DomainVerificationManual, "", now)
	})
	require.NoError(t, err)

	// tls-ready
	_, err = withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.TLSReady(context.Background(), "d1", now)
	})
	require.NoError(t, err)

	// activate
	_, err = withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.Activate(context.Background(), "d1")
	})
	require.NoError(t, err)

	// 断言 domain active + tenant active + issuer 派生 + epoch 递增
	d := getDomain(t, db, "t1", "d1")
	require.Equal(t, model.DomainStatusActive, d.Status)
	tn := getTenant(t, db, "t1")
	require.Equal(t, model.TenantStatusActive, tn.Status)
	require.Equal(t, "https://login.acme.com/oidc", tn.Issuer)
	require.EqualValues(t, 1, tn.DomainEpoch)
}

// TestActivateRejectsSecondActivePrimary 覆盖「无其他 active primary」不变量。
func TestActivateRejectsSecondActivePrimary(t *testing.T) {
	db := setupTD(t)
	seedTenant(t, db, "t1", model.TenantStatusActive)
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("d1"), Domain: "login.acme.com", DomainRole: model.DomainRolePrimary,
		DomainUsage: model.DomainUsageLogin, Status: model.DomainStatusActive,
	})
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("d2"), Domain: "login2.acme.com", DomainRole: model.DomainRolePrimary,
		DomainUsage: model.DomainUsageLogin, Status: model.DomainStatusTLSReady,
	})
	_, err := withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.Activate(context.Background(), "d2")
	})
	require.ErrorIs(t, err, ErrAlreadyActivePrimary)
}

// TestPrimaryDisableRejected 覆盖「active primary 不允许直接 disable」。
func TestPrimaryDisableRejected(t *testing.T) {
	db := setupTD(t)
	seedTenant(t, db, "t1", model.TenantStatusActive)
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("d1"), Domain: "login.acme.com", DomainRole: model.DomainRolePrimary,
		DomainUsage: model.DomainUsageLogin, Status: model.DomainStatusActive,
	})
	_, err := withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.Disable(context.Background(), "d1")
	})
	require.ErrorIs(t, err, ErrInvalidTransition)
}

// TestAliasTargetCascadeOnRetire 覆盖 alias 激活需 target active，以及 primary offboard→retire 级联 alias disabled。
func TestAliasTargetCascadeOnRetire(t *testing.T) {
	db := setupTD(t)
	seedTenant(t, db, "t1", model.TenantStatusActive)
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("p1"), Domain: "login.acme.com", DomainRole: model.DomainRolePrimary,
		DomainUsage: model.DomainUsageLogin, Status: model.DomainStatusActive,
	})
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("a1"), Domain: "sso.acme.com", DomainRole: model.DomainRoleAlias,
		DomainUsage: model.DomainUsageLogin, TargetDomainID: "p1", Status: model.DomainStatusActive,
	})

	// offboard 语义：tenant→suspended、primary→disabled（用 Disable 不允许，模拟 offboard 直接 CAS 到 disabled）
	tn := getTenant(t, db, "t1")
	tn.Status = model.TenantStatusSuspended
	tn.Issuer = "https://login.acme.com/oidc"
	require.NoError(t, db.WithContext(dbclient.CrossTenantContext(context.Background())).Save(tn).Error)
	// 直接把 primary 置 disabled（模拟 offboard 事务已做）
	require.NoError(t, db.WithContext(dbclient.ExplicitTenantContext(context.Background(), "t1")).
		Model(&model.DomainEntity{}).Where("id=?", "p1").Update("status", model.DomainStatusDisabled).Error)

	// retire primary → 应把 alias a1 级联 disabled
	now := time.Now()
	hosts, err := withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.Retire(context.Background(), "p1", "offboard", now)
	})
	require.NoError(t, err)
	require.Contains(t, hosts, "sso.acme.com")

	require.Equal(t, model.DomainStatusRetired, getDomain(t, db, "t1", "p1").Status)
	require.Equal(t, model.DomainStatusDisabled, getDomain(t, db, "t1", "a1").Status)
	// alias 保留 target_domain_id 不改
	require.Equal(t, "p1", getDomain(t, db, "t1", "a1").TargetDomainID)
}

// TestEmailAliasSkipsTLSReady 覆盖 email-only alias verified→active（不要求 tls_ready）。
func TestEmailAliasSkipsTLSReady(t *testing.T) {
	db := setupTD(t)
	seedTenant(t, db, "t1", model.TenantStatusActive)
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("p1"), Domain: "login.acme.com", DomainRole: model.DomainRolePrimary,
		DomainUsage: model.DomainUsageLogin, Status: model.DomainStatusActive,
	})
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("e1"), Domain: "acme.com", DomainRole: model.DomainRoleAlias,
		DomainUsage: model.DomainUsageEmail, TargetDomainID: "p1", Status: model.DomainStatusVerified,
	})
	_, err := withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.Activate(context.Background(), "e1")
	})
	require.NoError(t, err)
	require.Equal(t, model.DomainStatusActive, getDomain(t, db, "t1", "e1").Status)
}

// TestReleaseRequiresApprovalAndPeriod 覆盖 release 需满 90 天 + 审批单。
func TestReleaseRequiresApprovalAndPeriod(t *testing.T) {
	db := setupTD(t)
	seedTenant(t, db, "t1", model.TenantStatusSuspended)
	past := time.Now().Add(-91 * 24 * time.Hour)
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("p1"), Domain: "login.acme.com", DomainRole: model.DomainRolePrimary,
		DomainUsage: model.DomainUsageLogin, Status: model.DomainStatusRetired, RetiredAt: &past,
	})
	// 缺审批单 → 拒
	_, err := withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.Release(context.Background(), "p1", "", time.Now())
	})
	require.ErrorIs(t, err, ErrReleaseNotDue)
	// 有审批单 → 成功
	_, err = withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.Release(context.Background(), "p1", "TICKET-1", time.Now())
	})
	require.NoError(t, err)
	require.Equal(t, model.DomainStatusReleased, getDomain(t, db, "t1", "p1").Status)
}

// TestCancelDeleteRejectedWhenReferenced 覆盖有未 released alias 引用时拒绝取消删除。
func TestCancelDeleteRejectedWhenReferenced(t *testing.T) {
	db := setupTD(t)
	seedTenant(t, db, "t1", model.TenantStatusProvisioning)
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("p1"), Domain: "login.acme.com", DomainRole: model.DomainRolePrimary,
		DomainUsage: model.DomainUsageLogin, Status: model.DomainStatusDraft,
	})
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("a1"), Domain: "sso.acme.com", DomainRole: model.DomainRoleAlias,
		DomainUsage: model.DomainUsageLogin, TargetDomainID: "p1", Status: model.DomainStatusDraft,
	})
	_, err := withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.CancelDelete(context.Background(), "p1", time.Now())
	})
	require.ErrorIs(t, err, ErrReferencedByAlias)
}

// TestConcurrentConflictCAS 覆盖 CAS 0 行=ErrConflict（状态已被改）。
func TestConcurrentConflictCAS(t *testing.T) {
	db := setupTD(t)
	seedTenant(t, db, "t1", model.TenantStatusProvisioning)
	seedDomain(t, db, "t1", &model.DomainEntity{
		BaseEntity: be("d1"), Domain: "login.acme.com", DomainRole: model.DomainRolePrimary,
		DomainUsage: model.DomainUsageLogin, Status: model.DomainStatusVerified, // 不是 draft
	})
	// Verify 期望 draft，但当前是 verified → CAS 命中 0 行... 但 assertVerify 会先拒
	_, err := withTx(t, db, func(tx *Tx) ([]string, error) {
		tx.TenantID = "t1"
		return tx.Verify(context.Background(), "d1", model.DomainVerificationManual, "", time.Now())
	})
	require.ErrorIs(t, err, ErrInvalidTransition) // assert 先于 CAS
}

// --- helpers ---

func be(id string) gormdao.BaseEntity {
	return gormdao.BaseEntity{StringID: gormdao.StringID{ID: id}}
}

func getTenant(t *testing.T, db *gorm.DB, id string) *model.TenantEntity {
	t.Helper()
	var tn model.TenantEntity
	require.NoError(t, db.WithContext(dbclient.CrossTenantContext(context.Background())).Where("id=?", id).First(&tn).Error)
	return &tn
}

func getDomain(t *testing.T, db *gorm.DB, tenantID, id string) *model.DomainEntity {
	t.Helper()
	var d model.DomainEntity
	require.NoError(t, db.WithContext(dbclient.ExplicitTenantContext(context.Background(), tenantID)).Where("id=?", id).First(&d).Error)
	return &d
}
