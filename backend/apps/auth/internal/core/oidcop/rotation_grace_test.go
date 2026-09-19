package oidcop

import (
	"context"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// rotationGraceTestClientID 是宽限窗口用例里使用的 client_id。
const rotationGraceTestClientID = "client-grace"

// TestRefreshRotationGraceWindowReplaysSameRefreshToken 覆盖 P9 的核心语义：
// 同一把旧 refresh token 在宽限窗口内第二次刷新，**不**判为复用（不撤销 token 家族），
// 而是原样返回第一次轮换签发的那把新 refresh token。
//
// 这对应真实世界里的「响应丢失后客户端重试」与「两个标签页同时刷新」。
func TestRefreshRotationGraceWindowReplaysSameRefreshToken(t *testing.T) {
	ctx := context.Background()
	users := []model.UserEntity{
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "30"}}, TenantID: "1", PersonID: "90"},
	}
	storage, db := newTenantClaimTestStore(t, users)
	if err := db.Create(&model.ApplicationClientEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "acl-grace"}},
		Code:       rotationGraceTestClientID,
		TenantID:   "1",
	}).Error; err != nil {
		t.Fatalf("insert application client: %v", err)
	}
	persistentStore := storage.persistentStore

	// 首次授权：拿到第一把 refresh token。
	authReq := &AuthRequest{Subject: BuildSubject("90"), ClientID: rotationGraceTestClientID, TenantID: "1"}
	_, firstRefresh, _, err := storage.CreateAccessAndRefreshTokens(ctx, authReq, "")
	if err != nil {
		t.Fatalf("initial CreateAccessAndRefreshTokens failed: %v", err)
	}

	req1, err := storage.TokenRequestByRefreshToken(ctx, firstRefresh)
	if err != nil {
		t.Fatalf("first TokenRequestByRefreshToken failed: %v", err)
	}
	_, rotated, _, err := storage.CreateAccessAndRefreshTokens(ctx, req1, firstRefresh)
	if err != nil {
		t.Fatalf("first rotation failed: %v", err)
	}
	if rotated == "" || rotated == firstRefresh {
		t.Fatalf("rotation must issue a new refresh token, got %q", rotated)
	}

	// 旧 token 已被撤销：常规路径应当拒绝。
	if _, err := storage.TokenRequestByRefreshToken(ctx, firstRefresh); err != nil {
		t.Fatalf("revoked token must still be readable inside the grace window, got %v", err)
	}

	wantKey := rotationKey(firstRefresh)
	if entry, ok := persistentStore.rotationGrace[wantKey]; !ok {
		t.Fatalf("rotation grace entry missing for the consumed refresh token key %q", wantKey)
	} else if entry.newRefreshToken != rotated {
		t.Fatalf("grace entry must point at the rotated refresh token: want %q, got %q", rotated, entry.newRefreshToken)
	}

	// 宽限窗口内的第二次刷新：必须返回与第一次相同的新 token。
	req2, err := storage.TokenRequestByRefreshToken(ctx, firstRefresh)
	if err != nil {
		t.Fatalf("second TokenRequestByRefreshToken failed: %v", err)
	}
	_, replayed, _, err := storage.CreateAccessAndRefreshTokens(ctx, req2, firstRefresh)
	if err != nil {
		t.Fatalf("grace-window replay must not be treated as reuse, got %v", err)
	}
	if replayed != rotated {
		t.Fatalf("grace-window replay must return the same refresh token: want %q, got %q", rotated, replayed)
	}

	// token 家族必须仍然有效：新 token 可继续正常使用（未被误判复用而全撤销）。
	if _, err := storage.TokenRequestByRefreshToken(ctx, rotated); err != nil {
		t.Fatalf("token family must stay alive after a grace-window replay, got %v", err)
	}
	var active int64
	if err := db.Table(model.TableNameRefreshToken).
		Where("person_id = ? AND revoked_at IS NULL", "90").Count(&active).Error; err != nil {
		t.Fatalf("count active refresh tokens failed: %v", err)
	}
	if active != 1 {
		t.Fatalf("expected exactly the latest refresh token active, got %d", active)
	}
}

// TestRefreshRotationGraceWindowExpires 确认窗口外仍然 fail-closed：
// 撤销时间早于窗口的旧 token 读不回来，也不会被重放。
func TestRefreshRotationGraceWindowExpires(t *testing.T) {
	ctx := context.Background()
	users := []model.UserEntity{
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "31"}}, TenantID: "1", PersonID: "91"},
	}
	storage, db := newTenantClaimTestStore(t, users)
	if err := db.Create(&model.ApplicationClientEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "acl-grace"}},
		Code:       rotationGraceTestClientID,
		TenantID:   "1",
	}).Error; err != nil {
		t.Fatalf("insert application client: %v", err)
	}

	authReq := &AuthRequest{Subject: BuildSubject("91"), ClientID: rotationGraceTestClientID, TenantID: "1"}
	_, firstRefresh, _, err := storage.CreateAccessAndRefreshTokens(ctx, authReq, "")
	if err != nil {
		t.Fatalf("initial CreateAccessAndRefreshTokens failed: %v", err)
	}

	// 模拟「窗口早已过去」：把撤销时间与宽限登记都推回窗口之外。
	stale := time.Now().Add(-2 * refreshRotationGraceWindow)
	if err := db.Table(model.TableNameRefreshToken).
		Where("token = ?", credential.HashSecret(firstRefresh)).
		Update("revoked_at", &stale).Error; err != nil {
		t.Fatalf("mark token revoked in the past failed: %v", err)
	}
	key := rotationKey(firstRefresh)
	storage.persistentStore.rotationGrace[key] = refreshGraceEntry{
		newRefreshToken: "should-not-be-used",
		expiration:      time.Now().Add(time.Minute),
		rotatedAt:       stale,
	}

	if _, err := storage.TokenRequestByRefreshToken(ctx, firstRefresh); err == nil {
		t.Fatal("token revoked outside the grace window must be rejected")
	}
	// 过期条目应按需清理。
	if _, ok := storage.persistentStore.lookupRotationGrace(key); ok {
		t.Fatal("stale grace entry must not be replayed")
	}
	if _, exists := storage.persistentStore.rotationGrace[key]; exists {
		t.Fatal("stale grace entry must be evicted on lookup")
	}
}

// TestRefreshRotationGraceWindowBounded 确认宽限缓存有界：
// 条目数达到上界后不会被无界撑大（用唯一旧 token 模拟窗口内大量并发）。
func TestRefreshRotationGraceWindowBounded(t *testing.T) {
	store := &PersistentStore{
		rotationGrace:    make(map[string]refreshGraceEntry),
		rotationInflight: make(map[string]chan struct{}),
	}
	now := time.Now()
	for i := 0; i < maxRefreshGraceEntries+10; i++ {
		store.rotationGrace[rotationKey(string(rune(i)))] = refreshGraceEntry{
			newRefreshToken: "rt",
			expiration:      now.Add(time.Minute),
			rotatedAt:       now,
		}
	}
	// 超上界后再写入：必须淘汰最旧的一条，而不是无界增长、也不是整体清空。
	store.rememberRotationGrace(rotationKey("newest"), refreshGraceEntry{
		newRefreshToken: "rt-new",
		expiration:      now.Add(time.Minute),
		rotatedAt:       now.Add(time.Minute),
	})
	if len(store.rotationGrace) != maxRefreshGraceEntries {
		t.Fatalf("grace cache must stay bounded at %d entries, got %d", maxRefreshGraceEntries, len(store.rotationGrace))
	}
	if _, ok := store.rotationGrace[rotationKey("newest")]; !ok {
		t.Fatal("the newest entry must be kept when evicting")
	}
}

// TestWithinRotationGrace 是窗口判定的边界表。
func TestWithinRotationGrace(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name      string
		revokedAt time.Time
		want      bool
	}{
		{name: "just revoked", revokedAt: now, want: true},
		{name: "inside window", revokedAt: now.Add(-refreshRotationGraceWindow / 2), want: true},
		{name: "at boundary", revokedAt: now.Add(-refreshRotationGraceWindow), want: true},
		{name: "outside window", revokedAt: now.Add(-refreshRotationGraceWindow - time.Second), want: false},
		{name: "zero value", revokedAt: time.Time{}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := withinRotationGrace(tc.revokedAt, now); got != tc.want {
				t.Fatalf("withinRotationGrace = %v, want %v", got, tc.want)
			}
		})
	}
}
