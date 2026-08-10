package svcaudit

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWriteAudit(t *testing.T) {
	ctx, db, restore := newAuditTestEnv(t)
	defer restore()

	WriteAudit(ctx, AuditEntry{
		Action:     ActionApplicationCreate,
		TenantID:   1,
		TargetType: "application",
		TargetID:   7,
		Result:     "success",
		Detail:     "created app",
		ClientID:   "web-portal",
	})

	var entity model.AuditLogEntity
	if err := db.First(&entity).Error; err != nil {
		t.Fatalf("query persisted audit log failed: %v", err)
	}

	if entity.ActorPersonID != 101 {
		t.Fatalf("expected actor_person_id 101, got %d", entity.ActorPersonID)
	}
	if entity.ActorUserID != 202 {
		t.Fatalf("expected actor_user_id 202, got %d", entity.ActorUserID)
	}
	if entity.TenantID != 1 {
		t.Fatalf("expected tenant_id 1, got %d", entity.TenantID)
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
	if entity.TargetID != 7 {
		t.Fatalf("expected target_id 7, got %d", entity.TargetID)
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
	if entity.CreatedBy != 202 {
		t.Fatalf("expected created_by 202, got %d", entity.CreatedBy)
	}
}

func newAuditTestEnv(t *testing.T) (*gin.Context, *gorm.DB, func()) {
	t.Helper()

	req, rerr := http.NewRequest(http.MethodPost, "http://localhost/v1/iam/application/create", nil)
	if rerr != nil {
		t.Fatalf("build request: %v", rerr)
	}
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "10.0.0.1:61234"

	ctx, _ := gin.CreateTestContext(nil)
	ctx.Request = req.WithContext(context.Background())
	ctx.Set(gcontext.KeyPersonID, uint(101))
	ctx.Set(gcontext.KeyUserID, uint(202))
	ctx.Set(gcontext.KeyTenantID, uint(1))

	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeAuditTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	_ = db.AutoMigrate(&model.AuditLogEntity{})

	auditDao := dao.NewAuditLogDaoWithDB(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	})

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
