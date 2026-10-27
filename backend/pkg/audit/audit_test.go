package audit

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWriteAudit(t *testing.T) {
	ctx, db, restore := newAuditTestEnv(t)
	defer restore()

	WriteAudit(ctx, AuditEntry{
		Action:     ActionApplicationCreate,
		TenantID:   "1",
		TargetType: "application",
		TargetID:   "7",
		Result:     "success",
		Detail:     "created app",
		ClientID:   "web-portal",
		// 身份是业务输入：由调用方从 DTO/显式参数填进 Operator，审计不再读 gin 上下文。
		Operator: newAuditTestOperator(),
	})

	var entity model.AuditLogEntity
	if err := db.First(&entity).Error; err != nil {
		t.Fatalf("query persisted audit log failed: %v", err)
	}

	if entity.ActorPersonID != "101" {
		t.Fatalf("expected actor_person_id 101, got %s", entity.ActorPersonID)
	}
	if entity.ActorUserID != "202" {
		t.Fatalf("expected actor_user_id 202, got %s", entity.ActorUserID)
	}
	if entity.TenantID != "1" {
		t.Fatalf("expected tenant_id 1, got %s", entity.TenantID)
	}
	if entity.ClientID != "web-portal" {
		t.Fatalf("expected client_id 'web-portal', got '%s'", entity.ClientID)
	}
	if entity.Action != ActionApplicationCreate {
		t.Fatalf("expected action '%s', got '%s'", ActionApplicationCreate, entity.Action)
	}
	if entity.TargetType != "application" {
		t.Fatalf("expected target_type 'application', got '%s'", entity.TargetType)
	}
	if entity.TargetID != "7" {
		t.Fatalf("expected target_id 7, got %s", entity.TargetID)
	}
	if entity.Result != "success" {
		t.Fatalf("expected result 'success', got '%s'", entity.Result)
	}
	if entity.IP != "10.0.0.1" {
		t.Fatalf("expected ip '10.0.0.1', got '%s'", entity.IP)
	}
	if entity.UserAgent != "test-agent" {
		t.Fatalf("expected user_agent 'test-agent', got '%s'", entity.UserAgent)
	}
	if entity.CreatedBy != "202" {
		t.Fatalf("expected created_by 202, got %s", entity.CreatedBy)
	}
}

// newAuditTestOperator 构造审计入参里的操作人（与 controller 的
// identity.OperatorFromGin 产出同形）。
func newAuditTestOperator() objidentity.Operator {
	return objidentity.Operator{
		UserID:    "202",
		PersonID:  "101",
		ClientIP:  "10.0.0.1",
		UserAgent: "test-agent",
	}
}

func newAuditTestEnv(t *testing.T) (context.Context, *gorm.DB, func()) {
	t.Helper()

	// 审计只需要一个非 nil 的纯 context.Context；身份与 IP/UA 全部来自 AuditEntry.Operator。
	ctx := context.Background()

	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeAuditTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	_ = db.AutoMigrate(&model.AuditLogEntity{})

	auditDao := dao.NewAuditLogDao(dao.WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))

	orig := newAuditLogDao
	newAuditLogDao = func() *dao.AuditLogDao { return auditDao }

	restore := func() {
		newAuditLogDao = orig
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}

	return ctx, db, restore
}

func sanitizeAuditTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(name)
}

// TestWriteAudit_EmptyOperatorLeavesActorAndCreatedByEmpty 边界：未携带操作人时
// actor_*/created_by 一律落空串（既有规则），不得从任何隐式上下文"捞"出身份。
func TestWriteAudit_EmptyOperatorLeavesActorAndCreatedByEmpty(t *testing.T) {
	ctx, db, restore := newAuditTestEnv(t)
	defer restore()

	WriteAudit(ctx, AuditEntry{
		Action:     ActionInstallationInitialize,
		TenantID:   "1",
		TargetType: "tenant",
		TargetID:   "7",
		Result:     "success",
	})

	var entity model.AuditLogEntity
	if err := db.First(&entity).Error; err != nil {
		t.Fatalf("query persisted audit log failed: %v", err)
	}
	if entity.ActorPersonID != "" || entity.ActorUserID != "" {
		t.Fatalf("空 operator 必须落空串, got actor_person_id=%q actor_user_id=%q", entity.ActorPersonID, entity.ActorUserID)
	}
	if entity.CreatedBy != "" {
		t.Fatalf("空 operator 时 created_by 必须为空串, got %q", entity.CreatedBy)
	}
	if entity.IP != "" || entity.UserAgent != "" {
		t.Fatalf("空 operator 时 IP/UA 必须为空, got ip=%q ua=%q", entity.IP, entity.UserAgent)
	}
}

// TestWriteAudit_ExplicitIPOverridesOperator 非 HTTP 入口：显式 IP/UA 覆盖 Operator 里的值。
func TestWriteAudit_ExplicitIPOverridesOperator(t *testing.T) {
	ctx, db, restore := newAuditTestEnv(t)
	defer restore()

	WriteAudit(ctx, AuditEntry{
		Action:    ActionApiKeyCreate,
		TenantID:  "1",
		TargetID:  "7",
		Result:    "success",
		Operator:  newAuditTestOperator(),
		IP:        "192.0.2.9",
		UserAgent: "svc-worker/1.0",
	})

	var entity model.AuditLogEntity
	if err := db.First(&entity).Error; err != nil {
		t.Fatalf("query persisted audit log failed: %v", err)
	}
	if entity.IP != "192.0.2.9" || entity.UserAgent != "svc-worker/1.0" {
		t.Fatalf("显式 IP/UA 应覆盖 operator, got ip=%q ua=%q", entity.IP, entity.UserAgent)
	}
	if entity.ActorUserID != "202" {
		t.Fatalf("身份仍应取自 operator, got actor_user_id=%q", entity.ActorUserID)
	}
}
