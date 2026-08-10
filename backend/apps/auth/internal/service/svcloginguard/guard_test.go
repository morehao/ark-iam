package svcloginguard

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/morehao/ark-iam/auth/config"
	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestCfgDefaults(t *testing.T) {
	oldConf := config.Conf
	config.Conf = nil
	defer func() { config.Conf = oldConf }()

	mf, ws, ls := cfg()
	assert.Equal(t, 5, mf)
	assert.Equal(t, 300, ws)
	assert.Equal(t, 900, ls)
}

func TestCfgFromConfig(t *testing.T) {
	oldConf := config.Conf
	config.Conf = &pkgconfig.Config{Security: pkgconfig.SecurityConfig{Login: pkgconfig.LoginGuardConfig{
		MaxFailures: 7, WindowSec: 120, LockSec: 600,
	}}}
	defer func() { config.Conf = oldConf }()

	mf, ws, ls := cfg()
	assert.Equal(t, 7, mf)
	assert.Equal(t, 120, ws)
	assert.Equal(t, 600, ls)
}

func TestCfgIgnoresZero(t *testing.T) {
	oldConf := config.Conf
	config.Conf = &pkgconfig.Config{Security: pkgconfig.SecurityConfig{Login: pkgconfig.LoginGuardConfig{
		MaxFailures: 3, WindowSec: 0, LockSec: 0,
	}}}
	defer func() { config.Conf = oldConf }()

	mf, ws, ls := cfg()
	assert.Equal(t, 3, mf)
	assert.Equal(t, 300, ws)
	assert.Equal(t, 900, ls)
}

func TestFailOpenWhenRedisNil(t *testing.T) {
	oldCli := dbclient.RedisCli
	dbclient.RedisCli = nil
	defer func() { dbclient.RedisCli = oldCli }()

	ctx := context.Background()
	assert.False(t, Check(ctx, "1.2.3.4", 100))
	assert.NotPanics(t, func() { RecordFailure(ctx, "1.2.3.4", 100) })
	assert.NotPanics(t, func() { RecordSuccess(ctx, 100) })
}

func TestGuardCounterAndLock(t *testing.T) {
	s := miniredis.RunT(t)
	oldCli := dbclient.RedisCli
	oldConf := config.Conf
	dbclient.RedisCli = redis.NewClient(&redis.Options{Addr: s.Addr()})
	config.Conf = &pkgconfig.Config{Security: pkgconfig.SecurityConfig{Login: pkgconfig.LoginGuardConfig{
		MaxFailures: 3, WindowSec: 300, LockSec: 900,
	}}}
	defer func() {
		dbclient.RedisCli.Close()
		dbclient.RedisCli = oldCli
		config.Conf = oldConf
	}()

	ctx := context.Background()
	const ip = "10.0.0.1"
	const pid uint = 42

	assert.False(t, Check(ctx, ip, pid), "not locked before failures")

	RecordFailure(ctx, ip, pid)
	RecordFailure(ctx, ip, pid)
	assert.False(t, Check(ctx, ip, pid), "not locked below threshold")

	RecordFailure(ctx, ip, pid)
	assert.True(t, Check(ctx, ip, pid), "locked after threshold")

	// success clears the person-side lock; a second-chance login still sees
	// the person cleared but IP remains locked, so verify via a fresh person
	RecordSuccess(ctx, pid)
	assert.False(t, Check(ctx, "other-ip", pid), "person unlocked after success")
	assert.True(t, Check(ctx, ip, pid), "ip lock persists after success (person only cleared)")
}
