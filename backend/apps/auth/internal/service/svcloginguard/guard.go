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
// person 维度锁记录触发锁定的 IP：仅当当前请求 IP 与触发 IP 一致时拒绝，
// 防止攻击者用受害者的用户名触发锁定后，受害者从其它 IP 也无法登录（DoS）。
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
		if v, _ := dbclient.RedisCli.Get(ctx, lockPerson).Result(); v == ip {
			return true
		}
	}
	return false
}

// RecordFailure 记录失败次数，达到阈值则锁定 IP，并对 person 记录触发锁定的 IP。
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
			// 记录触发锁定的 IP：Check 时仅同 IP 拒绝，避免账号锁定被用作 DoS
			cli.Set(ctx, fmt.Sprintf(keyLockPerson, personID), ip, lock)
		}
		cli.Expire(ctx, personKey, w)
	}
}

// RecordSuccess 登录成功清除该 person 与来源 IP 的失败计数与锁定状态。
// 此前仅清 person 维度：共享出口 IP（NAT/办公网）下某用户触发 IP 锁后，
// 其它用户即使成功登录也无法解锁，导致整个出口 IP 被误锁（H9）。
func RecordSuccess(ctx context.Context, ip string, personID string) {
	if dbclient.RedisCli == nil {
		return
	}
	keys := personKeys(personID)
	if ip != "" {
		keys = append(keys, fmt.Sprintf(keyFailIP, ip), fmt.Sprintf(keyLockIP, ip))
	}
	dbclient.RedisCli.Del(ctx, keys...)
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
