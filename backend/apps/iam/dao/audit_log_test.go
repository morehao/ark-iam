package dao

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/morehao/ark-iam/iam/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuditLogDao_InsertAndGetByCond(t *testing.T) {
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeAuditTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}()

	_ = db.AutoMigrate(&model.AuditLogEntity{})

	auditDao := NewAuditLogDaoWithDB(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	})

	entity := &model.AuditLogEntity{
		ActorPersonID: 11,
		ActorUserID:   22,
		TenantID:      1,
		ClientID:      "test-client",
		Action:        "application.create",
		TargetType:    "application",
		TargetID:      99,
		Result:        "success",
		IP:            "127.0.0.1",
		UserAgent:     "go-test",
		Detail:        "created app",
		CreatedBy:     22,
	}
	if err := auditDao.Insert(context.Background(), entity); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if entity.ID == 0 {
		t.Fatal("expected non-zero ID after insert")
	}

	got, err := auditDao.GetByCond(context.Background(), &AuditLogCond{
		PersonID: 11,
		Action:   "application.create",
		Result:   "success",
	})
	if err != nil {
		t.Fatalf("GetByCond failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected one audit log row")
	}
	if got.ID != entity.ID {
		t.Fatalf("expected id %d, got %d", entity.ID, got.ID)
	}
	if got.ActorUserID != 22 {
		t.Fatalf("expected actor_user_id 22, got %d", got.ActorUserID)
	}
	if got.TenantID != 1 {
		t.Fatalf("expected tenant_id 1, got %d", got.TenantID)
	}
	if got.TargetID != 99 {
		t.Fatalf("expected target_id 99, got %d", got.TargetID)
	}
	if got.Detail != "created app" {
		t.Fatalf("expected detail 'created app', got '%s'", got.Detail)
	}
}

func sanitizeAuditTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(name)
}
