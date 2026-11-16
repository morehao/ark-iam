package loginsetting

import (
	"context"
	"testing"

	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) gormdao.DBGetter {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:loginsetting_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }
}

func TestLoadDefaultsWhenEmpty(t *testing.T) {
	db := setupDB(t)
	c := Load(context.Background(), db)
	if c.MaxFailures != DefaultMaxFailures || c.WindowSec != DefaultWindowSec ||
		c.LockSec != DefaultLockSec || c.RatePerMinute != DefaultRatePerMinute || c.Burst != DefaultBurst {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	want := Config{MaxFailures: 7, WindowSec: 120, LockSec: 600, RatePerMinute: 60, Burst: 20}
	if err := Save(ctx, db, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := Load(ctx, db)
	if got != want {
		t.Fatalf("roundtrip mismatch: got=%+v want=%+v", got, want)
	}
}

func TestIsEmpty(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if empty, _ := IsEmpty(ctx, db); !empty {
		t.Fatal("expected empty before save")
	}
	if err := Save(ctx, db, Config{MaxFailures: 3, WindowSec: 300, LockSec: 900, RatePerMinute: 30, Burst: 10}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if empty, _ := IsEmpty(ctx, db); empty {
		t.Fatal("expected non-empty after save")
	}
}
