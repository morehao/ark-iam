package person

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	dsn := fmt.Sprintf("file:%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.PersonEntity{}); err != nil {
		t.Fatalf("migrate person: %v", err)
	}
	return db, func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
}

func seedPerson(t *testing.T, db *gorm.DB, email string) {
	t.Helper()
	if err := db.Create(&model.PersonEntity{
		Username:          model.StrPtr(""),
		PrimaryEmail:      model.StrPtr(email),
		PrimaryPhone:      model.StrPtr(""),
		Name:              "seed",
		Profile:           []byte(`{}`),
		CustomData:        []byte(`{}`),
		PasswordEncrypted: "hash",
		PasswordMethod:    "bcrypt",
	}).Error; err != nil {
		t.Fatalf("seed person: %v", err)
	}
}

func TestFindOrCreate_MatchExistingByEmail(t *testing.T) {
	db, restore := newTestDB(t)
	defer restore()

	tx := db.Begin()
	seedPerson(t, tx, "exist@example.com")
	tx.Commit()

	req := &FindOrCreateReq{
		PrimaryEmail: "exist@example.com",
		Name:         "new",
	}
	p, created, err := FindOrCreate(context.Background(), db, req)
	if err != nil {
		t.Fatalf("FindOrCreate fail: %v", err)
	}
	if created {
		t.Fatalf("expected reuse (created=false) for existing email")
	}
	if p.PrimaryEmail == nil || model.DerefStr(p.PrimaryEmail) != "exist@example.com" {
		t.Fatalf("expected matched person, got %v", p)
	}
}

func TestFindOrCreate_CreateWhenNoMatch(t *testing.T) {
	db, restore := newTestDB(t)
	defer restore()

	req := &FindOrCreateReq{
		PrimaryEmail:      "new@example.com",
		Name:              "Alice",
		PasswordEncrypted: "hash",
		PasswordMethod:    "bcrypt",
	}
	p, created, err := FindOrCreate(context.Background(), db, req)
	if err != nil {
		t.Fatalf("FindOrCreate fail: %v", err)
	}
	if !created {
		t.Fatalf("expected creation (created=true) when no match")
	}
	if p.ID == "" {
		t.Fatalf("expected created person id, got empty")
	}
	if model.DerefStr(p.PrimaryEmail) != "new@example.com" {
		t.Fatalf("expected new email, got %q", model.DerefStr(p.PrimaryEmail))
	}
}

func TestFindOrCreate_CreateInCallerTx(t *testing.T) {
	db, restore := newTestDB(t)
	defer restore()

	tx := db.Begin()

	req := &FindOrCreateReq{PrimaryPhone: "13800000000", Name: "Bob"}
	p, created, err := FindOrCreate(context.Background(), tx, req)
	if err != nil {
		t.Fatalf("FindOrCreate fail: %v", err)
	}
	if !created {
		t.Fatalf("expected creation in tx")
	}
	if p.ID == "" {
		t.Fatalf("expected created id")
	}

	tx.Rollback()

	// 回滚后不应残留
	var count int64
	if err := db.Model(&model.PersonEntity{}).Count(&count).Error; err != nil {
		t.Fatalf("count fail: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to discard person, got %d", count)
	}
}
