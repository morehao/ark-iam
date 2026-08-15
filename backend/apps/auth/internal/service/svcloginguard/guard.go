package svcloginguard

import (
	"context"
	"fmt"
	"time"

	"github.com/morehao/ark-iam/auth/config"
	"github.com/morehao/ark-iam/pkg/dbclient"
)

const (
	keyFailIP     = "iam:login_fail:ip:%s"
	keyFailPerson = "iam:login_fail:person:%s"
	keyLockIP     = "iam:login_lock:ip:%s"
	keyLockPerson = "iam:login_lock:person:%s"
)

func cfg() (maxFailures, windowSec, lockSec int) {
	maxFailures, windowSec, lockSec = 5, 300, 900
	if config.Conf != nil {
		if config.Conf.Security.Login.MaxFailures > 0 {
			maxFailures = config.Conf.Security.Login.MaxFailures
		}
		if config.Conf.Security.Login.WindowSec > 0 {
			windowSec = config.Conf.Security.Login.WindowSec
		}
		if config.Conf.Security.Login.LockSec > 0 {
			lockSec = config.Conf.Security.Login.LockSec
		}
	}
	return
}

// Check 是否被锁定，true=锁定。Redis 不可用时 fail-open 返回 false。
func Check(ctx context.Context, ip string, personID string) bool {
	if dbclient.RedisCli == nil {
		return false
	}
	lockIP := fmt.Sprintf(keyLockIP, ip)
	if v, _ := dbclient.RedisCli.Exists(ctx, lockIP).Result(); v > 0 {
		return true
	}
	if personID != "" {
		lockPerson := fmt.Sprintf(keyLockPerson, personID)
		if v, _ := dbclient.RedisCli.Exists(ctx, lockPerson).Result(); v > 0 {
			return true
		}
	}
	return false
}

// RecordFailure 记录失败次数，达到阈值则锁定 IP 与 person。
func RecordFailure(ctx context.Context, ip string, personID string) {
	if dbclient.RedisCli == nil {
		return
	}
	maxFailures, windowSec, lockSec := cfg()
	w := windowSeconds(windowSec)
	cli := dbclient.RedisCli
	lock := lockDuration(lockSec)

	ipKey := fmt.Sprintf(keyFailIP, ip)
	if n, _ := cli.Incr(ctx, ipKey).Result(); n >= int64(maxFailures) {
		cli.Set(ctx, fmt.Sprintf(keyLockIP, ip), "1", lock)
	}
	cli.Expire(ctx, ipKey, w)

	if personID != "" {
		personKey := fmt.Sprintf(keyFailPerson, personID)
		if n, _ := cli.Incr(ctx, personKey).Result(); n >= int64(maxFailures) {
			cli.Set(ctx, fmt.Sprintf(keyLockPerson, personID), "1", lock)
		}
		cli.Expire(ctx, personKey, w)
	}
}

// RecordSuccess 登录成功清除失败与锁定计数。
func RecordSuccess(ctx context.Context, personID string) {
	if dbclient.RedisCli == nil {
		return
	}
	dbclient.RedisCli.Del(ctx, personKeys(personID)...)
}

func windowSeconds(s int) time.Duration {
	if s <= 0 {
		s = 300
	}
	return time.Duration(s) * time.Second
}

func lockDuration(s int) time.Duration {
	if s <= 0 {
		s = 900
	}
	return time.Duration(s) * time.Second
}

func personKeys(pid string) []string {
	return []string{fmt.Sprintf(keyFailPerson, pid), fmt.Sprintf(keyLockPerson, pid)}
}
