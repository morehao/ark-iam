package dao

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSessionAuditDao_InsertAndGetByCond(t *testing.T) {
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeSessionAuditTestName(t.Name()), time.Now().UnixNano())
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

	_ = db.AutoMigrate(&model.SessionAuditEntity{})

	sessionAuditDao := NewSessionAuditDaoWithDB(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	})

	now := time.Now()
	entity := &model.SessionAuditEntity{
		PersonID:  55,
		SessionID: "sess-abc-123",
		TenantID:  1,
		ClientIP:  "10.0.0.1",
		UserAgent: "go-test-ua",
		LoginTime: now,
		Status:    "active",
		CreatedBy: 55,
		UpdatedBy: 55,
	}
	if err := sessionAuditDao.Insert(context.Background(), entity); err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if entity.ID == 0 {
		t.Fatal("expected non-zero ID after insert")
	}

	got, err := sessionAuditDao.GetByCond(context.Background(), &SessionAuditCond{
		PersonID:  55,
		SessionID: "sess-abc-123",
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("GetByCond failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected one session audit row")
	}
	if got.ID != entity.ID {
		t.Fatalf("expected id %d, got %d", entity.ID, got.ID)
	}
	if got.PersonID != 55 {
		t.Fatalf("expected person_id 55, got %d", got.PersonID)
	}
	if got.SessionID != "sess-abc-123" {
		t.Fatalf("expected session_id 'sess-abc-123', got '%s'", got.SessionID)
	}
	if got.TenantID != 1 {
		t.Fatalf("expected tenant_id 1, got %d", got.TenantID)
	}
	if got.ClientIP != "10.0.0.1" {
		t.Fatalf("expected client_ip '10.0.0.1', got '%s'", got.ClientIP)
	}
	if got.Status != "active" {
		t.Fatalf("expected status 'active', got '%s'", got.Status)
	}
}

func sanitizeSessionAuditTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(name)
}
