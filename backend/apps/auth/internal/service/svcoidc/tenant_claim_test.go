package svcoidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/token"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTenantClaimTestStore(t *testing.T, users []model.UserEntity) (storage *OIDCStorage, db *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:tenant_claim_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.UserEntity{}, &model.RefreshTokenEntity{}, &model.ApplicationClientEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	for i := range users {
		now := time.Now()
		users[i].Profile = []byte("{}")
		users[i].CustomData = []byte("{}")
		users[i].JoinedAt = &now
		users[i].LastSignInAt = &now
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}

	persistentStore := NewPersistentStore()
	persistentStore.userDao = func() *dao.UserDao {
		return &dao.UserDao{Dao: gormdao.NewDao[model.UserEntity, model.UserEntityList](
			model.TableNameUser, "UserDao",
			func(c context.Context) *gorm.DB { return db.WithContext(c) },
		)}
	}
	persistentStore.applicationClientDao = func() *dao.ApplicationClientDao {
		return dao.NewApplicationClientDaoWithDB(func(c context.Context) *gorm.DB { return db.WithContext(c) })
	}
	persistentStore.refreshTokenDao = func() *dao.RefreshTokenDao {
		return &dao.RefreshTokenDao{Dao: gormdao.NewDao[model.RefreshTokenEntity, model.RefreshTokenEntityList](
			model.TableNameRefreshToken, "RefreshTokenDao",
			func(c context.Context) *gorm.DB { return db.WithContext(c) },
		)}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	storage = NewOIDCStorage(nil, persistentStore, privateKey, "test-key")
	return storage, db
}

func TestCreateAccessAndRefreshTokensSelectsTenantFromAuthRequest(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{Model: gorm.Model{ID: 10}, TenantID: 1, PersonID: 88},
		{Model: gorm.Model{ID: 11}, TenantID: 2, PersonID: 88},
		{Model: gorm.Model{ID: 12}, TenantID: 9, PersonID: 88},
	}
	storage, db := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{
		Subject:  buildOIDCSubject(88),
		ClientID: "client-1",
		TenantID: 9,
	}

	_, refreshToken, _, err := storage.CreateAccessAndRefreshTokens(ctx, authReq, "")
	if err != nil {
		t.Fatalf("CreateAccessAndRefreshTokens failed: %v", err)
	}

	var stored model.RefreshTokenEntity
	if err := db.Table(model.TableNameRefreshToken).Where("token = ?", token.HashToken(refreshToken)).First(&stored).Error; err != nil {
		t.Fatalf("refresh token not stored: %v", err)
	}
	if stored.TenantID != 9 {
		t.Fatalf("expected refresh token tenant 9 (from AuthRequest), got %d", stored.TenantID)
	}
	if stored.UserID != 12 {
		t.Fatalf("expected refresh token user 12 (tenant 9), got %d", stored.UserID)
	}
}

func TestCreateAccessAndRefreshTokensFallsBackToFirstUserWhenNoTenant(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{Model: gorm.Model{ID: 20}, TenantID: 1, PersonID: 88},
		{Model: gorm.Model{ID: 21}, TenantID: 2, PersonID: 88},
	}
	storage, db := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{Subject: buildOIDCSubject(88), ClientID: "client-1"}

	_, refreshToken, _, err := storage.CreateAccessAndRefreshTokens(ctx, authReq, "")
	if err != nil {
		t.Fatalf("CreateAccessAndRefreshTokens failed: %v", err)
	}

	var stored model.RefreshTokenEntity
	if err := db.Table(model.TableNameRefreshToken).Where("token = ?", token.HashToken(refreshToken)).First(&stored).Error; err != nil {
		t.Fatalf("refresh token not stored: %v", err)
	}
	if stored.TenantID != 1 {
		t.Fatalf("expected refresh token tenant 1 (users[0] fallback), got %d", stored.TenantID)
	}
	if stored.UserID != 20 {
		t.Fatalf("expected refresh token user 20 (users[0] fallback), got %d", stored.UserID)
	}
}

func TestGetPrivateClaimsFromRequestSelectsTenantFromAuthRequest(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{Model: gorm.Model{ID: 30}, TenantID: 1, PersonID: 88},
		{Model: gorm.Model{ID: 31}, TenantID: 7, PersonID: 88},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{Subject: buildOIDCSubject(88), ClientID: "client-1", TenantID: 7}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, authReq, []string{"openid"})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if got, ok := claims["tenant_id"].(uint); !ok || got != 7 {
		t.Fatalf("expected tenant_id claim 7 (from AuthRequest), got %v (%T)", claims["tenant_id"], claims["tenant_id"])
	}
}

func TestGetPrivateClaimsFromRequestFallsBackToFirstUserWhenNoTenant(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{Model: gorm.Model{ID: 40}, TenantID: 1, PersonID: 88},
		{Model: gorm.Model{ID: 41}, TenantID: 5, PersonID: 88},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{Subject: buildOIDCSubject(88), ClientID: "client-1"}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, authReq, []string{"openid"})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if got := claims["tenant_id"]; got != uint(1) {
		t.Fatalf("expected tenant_id claim 1 (users[0] fallback), got %v (%T)", got, got)
	}
}

func TestCreateAccessAndRefreshTokensSelectsTenantFromRefreshTokenRequest(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{Model: gorm.Model{ID: 50}, TenantID: 1, PersonID: 88},
		{Model: gorm.Model{ID: 51}, TenantID: 6, PersonID: 88},
	}
	storage, db := newTenantClaimTestStore(t, users)

	// 模拟 refresh token 轮换：请求携带存储的租户 6
	refreshReq := &refreshTokenRequest{
		subject:  buildOIDCSubject(88),
		clientID: "client-1",
		tenantID: 6,
	}

	_, refreshToken, _, err := storage.CreateAccessAndRefreshTokens(ctx, refreshReq, "")
	if err != nil {
		t.Fatalf("CreateAccessAndRefreshTokens failed: %v", err)
	}

	var stored model.RefreshTokenEntity
	if err := db.Table(model.TableNameRefreshToken).Where("token = ?", token.HashToken(refreshToken)).First(&stored).Error; err != nil {
		t.Fatalf("refresh token not stored: %v", err)
	}
	if stored.TenantID != 6 {
		t.Fatalf("expected refresh token tenant 6 (from refreshTokenRequest), got %d", stored.TenantID)
	}
	if stored.UserID != 51 {
		t.Fatalf("expected refresh token user 51 (tenant 6), got %d", stored.UserID)
	}
}

func TestGetPrivateClaimsFromRequestSelectsTenantFromRefreshTokenRequest(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{Model: gorm.Model{ID: 60}, TenantID: 1, PersonID: 88},
		{Model: gorm.Model{ID: 61}, TenantID: 8, PersonID: 88},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	refreshReq := &refreshTokenRequest{
		subject:  buildOIDCSubject(88),
		clientID: "client-1",
		tenantID: 8,
	}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, refreshReq, []string{"openid"})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if got, ok := claims["tenant_id"].(uint); !ok || got != 8 {
		t.Fatalf("expected tenant_id claim 8 (from refreshTokenRequest), got %v (%T)", claims["tenant_id"], claims["tenant_id"])
	}
}

func TestGetPrivateClaimsFromRequestInjectsSidFromAuthRequest(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{Model: gorm.Model{ID: 70}, TenantID: 1, PersonID: 88},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{
		Subject:   buildOIDCSubject(88),
		ClientID:  "client-1",
		TenantID:  1,
		SessionID: "sid-xyz",
	}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, authReq, []string{"openid"})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if got, ok := claims["sid"].(string); !ok || got != "sid-xyz" {
		t.Fatalf("expected sid claim, got %v (%T)", claims["sid"], claims["sid"])
	}
}

func TestGetPrivateClaimsFromAuthRequestOmitsSidWhenEmpty(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{Model: gorm.Model{ID: 71}, TenantID: 1, PersonID: 88},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{Subject: buildOIDCSubject(88), ClientID: "client-1", TenantID: 1}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, authReq, []string{"openid"})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if _, exists := claims["sid"]; exists {
		t.Fatalf("expected no sid claim when SessionID empty, got %v", claims["sid"])
	}
}
