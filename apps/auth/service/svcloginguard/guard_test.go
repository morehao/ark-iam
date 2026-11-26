package svcloginguard

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/morehao/ark-iam/apps/auth/testutil"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/loginsetting"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gsetting"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// setupGuardDB 建内存库并返回 DBGetter（loginsetting 读它）。
func setupGuardDB(t *testing.T) gormdao.DBGetter {
	t.Helper()
	db := testutil.SetupSQLite(t, gsetting.SchemaModels()...)
	return func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }
}

func TestCfgDefaultsWhenNoConfig(t *testing.T) {
	db := setupGuardDB(t) // 空库：分组无行 -> 默认值
	mf, ws, ls := cfg(context.Background(), db)
	assert.Equal(t, 5, mf)
	assert.Equal(t, 300, ws)
	assert.Equal(t, 900, ls)
}

func TestCfgFromSetting(t *testing.T) {
	db := setupGuardDB(t)
	ctx := context.Background()
	if err := loginsetting.Save(ctx, db, loginsetting.Config{
		MaxFailures: 7, WindowSec: 120, LockSec: 600, RatePerMinute: 30, Burst: 10,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	mf, ws, ls := cfg(ctx, db)
	assert.Equal(t, 7, mf)
	assert.Equal(t, 120, ws)
	assert.Equal(t, 600, ls)
}

func TestCfgIgnoresZeroFallsBackToDefault(t *testing.T) {
	db := setupGuardDB(t)
	ctx := context.Background()
	if err := loginsetting.Save(ctx, db, loginsetting.Config{
		MaxFailures: 3, WindowSec: 0, LockSec: 0, RatePerMinute: 30, Burst: 10,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	mf, ws, ls := cfg(ctx, db)
	assert.Equal(t, 3, mf)
	assert.Equal(t, 300, ws)
	assert.Equal(t, 900, ls)
}

func TestFailOpenWhenRedisNil(t *testing.T) {
	oldCli := dbclient.RedisCli
	dbclient.RedisCli = nil
	defer func() { dbclient.RedisCli = oldCli }()

	db := setupGuardDB(t)
	ctx := context.Background()
	assert.False(t, Check(ctx, "1.2.3.4", "100"))
	assert.NotPanics(t, func() { RecordFailure(ctx, db, "1.2.3.4", "100") })
	assert.NotPanics(t, func() { RecordSuccess(ctx, "", "100") })
}

func TestGuardCounterAndLock(t *testing.T) {
	s := miniredis.RunT(t)
	oldCli := dbclient.RedisCli
	dbclient.RedisCli = redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer func() {
		_ = dbclient.RedisCli.Close()
		dbclient.RedisCli = oldCli
	}()

	db := setupGuardDB(t)
	ctx := context.Background()
	// 阈值 3 写进配置中心
	if err := loginsetting.Save(ctx, db, loginsetting.Config{
		MaxFailures: 3, WindowSec: 300, LockSec: 900, RatePerMinute: 30, Burst: 10,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	const ip = "10.0.0.1"
	const pid = "42"

	assert.False(t, Check(ctx, ip, pid), "not locked before failures")

	RecordFailure(ctx, db, ip, pid)
	RecordFailure(ctx, db, ip, pid)
	assert.False(t, Check(ctx, ip, pid), "not locked below threshold")

	RecordFailure(ctx, db, ip, pid)
	assert.True(t, Check(ctx, ip, pid), "locked after threshold")

	RecordSuccess(ctx, ip, pid)
	assert.False(t, Check(ctx, "other-ip", pid), "person unlocked after success")
	assert.False(t, Check(ctx, ip, pid), "ip lock cleared after success")
}
