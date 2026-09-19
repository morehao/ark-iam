package oidcop

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
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
	if err := db.AutoMigrate(&model.UserEntity{}, &model.RefreshTokenEntity{}, &model.ApplicationClientEntity{}, &model.TenantEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	for i := range users {
		now := time.Now()
		users[i].JoinedAt = &now
		users[i].LastSignInAt = &now
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatalf("insert user: %v", err)
		}
	}
	// 每个用户所属租户建为 active：令牌签发前的租户准入门禁（tenantTokenGate）
	// 会按 tenant_id 回查租户状态，非 active 一律拒绝签发。
	seededTenants := map[string]bool{}
	for i := range users {
		tenantID := users[i].TenantID
		if tenantID == "" || seededTenants[tenantID] {
			continue
		}
		seededTenants[tenantID] = true
		if err := db.Create(&model.TenantEntity{
			BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: tenantID}},
			Code:       "t_" + tenantID,
			Name:       "tenant-" + tenantID,
			Status:     model.TenantStatusActive,
			Type:       model.TenantTypeCustomer,
		}).Error; err != nil {
			t.Fatalf("insert tenant: %v", err)
		}
	}

	persistentStore := NewPersistentStore()
	persistentStore.tenantDao = func(opts ...dao.DaoOption) *dao.TenantDao {
		return &dao.TenantDao{Dao: gormdao.NewDao[model.TenantEntity, model.TenantEntityList, string](
			model.TableNameTenant, "TenantDao",
			func(c context.Context) *gorm.DB { return db.WithContext(c) },
		)}
	}
	persistentStore.userDao = func(opts ...dao.DaoOption) *dao.UserDao {
		return &dao.UserDao{Dao: gormdao.NewDao[model.UserEntity, model.UserEntityList, string](
			model.TableNameUser, "UserDao",
			func(c context.Context) *gorm.DB { return db.WithContext(c) },
		)}
	}
	persistentStore.applicationClientDao = func(opts ...dao.DaoOption) *dao.ApplicationClientDao {
		return dao.NewApplicationClientDao(dao.WithDBGetter(func(c context.Context) *gorm.DB { return db.WithContext(c) }))
	}
	persistentStore.refreshTokenDao = func(opts ...dao.DaoOption) *dao.RefreshTokenDao {
		return &dao.RefreshTokenDao{Dao: gormdao.NewDao[model.RefreshTokenEntity, model.RefreshTokenEntityList, string](
			model.TableNameRefreshToken, "RefreshTokenDao",
			func(c context.Context) *gorm.DB { return db.WithContext(c) },
		)}
	}
	persistentStore.db = func(c context.Context) *gorm.DB { return db.WithContext(c) }

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
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "10"}}, TenantID: "1", PersonID: "88"},
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "11"}}, TenantID: "2", PersonID: "88"},
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "12"}}, TenantID: "9", PersonID: "88"},
	}
	storage, db := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{
		Subject:  BuildSubject("88"),
		ClientID: "client-1",
		TenantID: "9",
	}

	_, refreshToken, _, err := storage.CreateAccessAndRefreshTokens(ctx, authReq, "")
	if err != nil {
		t.Fatalf("CreateAccessAndRefreshTokens failed: %v", err)
	}

	var stored model.RefreshTokenEntity
	if err := db.Table(model.TableNameRefreshToken).Where("token = ?", credential.HashSecret(refreshToken)).First(&stored).Error; err != nil {
		t.Fatalf("refresh token not stored: %v", err)
	}
	if stored.TenantID != "9" {
		t.Fatalf("expected refresh token tenant 9 (from AuthRequest), got %s", stored.TenantID)
	}
	if stored.UserID != "12" {
		t.Fatalf("expected refresh token user 12 (tenant 9), got %s", stored.UserID)
	}
}

func TestCreateAccessAndRefreshTokensFallsBackToFirstUserWhenNoTenant(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "20"}}, TenantID: "1", PersonID: "88"},
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "21"}}, TenantID: "2", PersonID: "88"},
	}
	storage, db := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{Subject: BuildSubject("88"), ClientID: "client-1"}

	_, refreshToken, _, err := storage.CreateAccessAndRefreshTokens(ctx, authReq, "")
	if err != nil {
		t.Fatalf("CreateAccessAndRefreshTokens failed: %v", err)
	}

	var stored model.RefreshTokenEntity
	if err := db.Table(model.TableNameRefreshToken).Where("token = ?", credential.HashSecret(refreshToken)).First(&stored).Error; err != nil {
		t.Fatalf("refresh token not stored: %v", err)
	}
	if stored.TenantID != "1" {
		t.Fatalf("expected refresh token tenant 1 (users[0] fallback), got %s", stored.TenantID)
	}
	if stored.UserID != "20" {
		t.Fatalf("expected refresh token user 20 (users[0] fallback), got %s", stored.UserID)
	}
}

func TestGetPrivateClaimsFromRequestSelectsTenantFromAuthRequest(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "30"}}, TenantID: "1", PersonID: "88"},
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "31"}}, TenantID: "7", PersonID: "88"},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{Subject: BuildSubject("88"), ClientID: "client-1", TenantID: "7"}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, authReq, []string{model.ScopeOpenID})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if got, ok := claims["tenant_id"].(string); !ok || got != "7" {
		t.Fatalf("expected tenant_id claim 7 (from AuthRequest), got %v (%T)", claims["tenant_id"], claims["tenant_id"])
	}
}

func TestGetPrivateClaimsFromRequestOmitsTenantWhenAmbiguous(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "40"}}, TenantID: "1", PersonID: "88"},
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "41"}}, TenantID: "5", PersonID: "88"},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	// 多租户 person 且请求未携带明确租户（TenantID == ""）时，宁可不产出 tenant_id，
	// 也不静默取 users[0]（L4）。
	authReq := &AuthRequest{Subject: BuildSubject("88"), ClientID: "client-1"}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, authReq, []string{model.ScopeOpenID})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if _, exists := claims["tenant_id"]; exists {
		t.Fatalf("expected no tenant_id claim for ambiguous tenant, got %v", claims["tenant_id"])
	}
}

func TestCreateAccessAndRefreshTokensSelectsTenantFromRefreshTokenRequest(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "50"}}, TenantID: "1", PersonID: "88"},
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "51"}}, TenantID: "6", PersonID: "88"},
	}
	storage, db := newTenantClaimTestStore(t, users)

	// 模拟 refresh token 轮换：请求携带存储的租户 6
	refreshReq := &refreshTokenRequest{
		subject:  BuildSubject("88"),
		clientID: "client-1",
		tenantID: "6",
	}

	_, refreshToken, _, err := storage.CreateAccessAndRefreshTokens(ctx, refreshReq, "")
	if err != nil {
		t.Fatalf("CreateAccessAndRefreshTokens failed: %v", err)
	}

	var stored model.RefreshTokenEntity
	if err := db.Table(model.TableNameRefreshToken).Where("token = ?", credential.HashSecret(refreshToken)).First(&stored).Error; err != nil {
		t.Fatalf("refresh token not stored: %v", err)
	}
	if stored.TenantID != "6" {
		t.Fatalf("expected refresh token tenant 6 (from refreshTokenRequest), got %s", stored.TenantID)
	}
	if stored.UserID != "51" {
		t.Fatalf("expected refresh token user 51 (tenant 6), got %s", stored.UserID)
	}
}

func TestGetPrivateClaimsFromRequestSelectsTenantFromRefreshTokenRequest(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "60"}}, TenantID: "1", PersonID: "88"},
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "61"}}, TenantID: "8", PersonID: "88"},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	refreshReq := &refreshTokenRequest{
		subject:  BuildSubject("88"),
		clientID: "client-1",
		tenantID: "8",
	}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, refreshReq, []string{model.ScopeOpenID})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if got, ok := claims["tenant_id"].(string); !ok || got != "8" {
		t.Fatalf("expected tenant_id claim 8 (from refreshTokenRequest), got %v (%T)", claims["tenant_id"], claims["tenant_id"])
	}
}

func TestGetPrivateClaimsFromRequestInjectsSidFromAuthRequest(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "70"}}, TenantID: "1", PersonID: "88"},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{
		Subject:   BuildSubject("88"),
		ClientID:  "client-1",
		TenantID:  "1",
		SessionID: "sid-xyz",
	}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, authReq, []string{model.ScopeOpenID})
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
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "71"}}, TenantID: "1", PersonID: "88"},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{Subject: BuildSubject("88"), ClientID: "client-1", TenantID: "1"}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, authReq, []string{model.ScopeOpenID})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if _, exists := claims["sid"]; exists {
		t.Fatalf("expected no sid claim when SessionID empty, got %v", claims["sid"])
	}
}

// TestGetPrivateClaimsFromRequestIncludesUserID 覆盖「令牌新增 user_id 声明」：
// 人令牌的 user_id 必须等于该 (tenant_id, person_id) 对应的 tenant_user.id，
// 且与刷新令牌行上的 UserID 一致（下游据此写审计操作者列）。
func TestGetPrivateClaimsFromRequestIncludesUserID(t *testing.T) {
	ctx := context.Background()

	users := []model.UserEntity{
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "50"}}, TenantID: "1", PersonID: "88"},
		{BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "51"}}, TenantID: "7", PersonID: "88"},
	}
	storage, _ := newTenantClaimTestStore(t, users)

	authReq := &AuthRequest{Subject: BuildSubject("88"), ClientID: "client-1", TenantID: "7", SessionID: "sid-1"}

	claims, err := storage.GetPrivateClaimsFromRequest(ctx, authReq, []string{model.ScopeOpenID})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest failed: %v", err)
	}
	if got := claims["user_id"]; got != "51" {
		t.Fatalf("expected user_id claim 51 (tenant 7 的 tenant_user.id), got %v", got)
	}
	if got := claims["person_id"]; got != "88" {
		t.Fatalf("expected person_id claim 88, got %v", got)
	}
	if got := claims["sid"]; got != "sid-1" {
		t.Fatalf("expected sid claim sid-1, got %v", got)
	}

	// 刷新轮换必须保持同一 user_id（验收标准 4：刷新后不变）。
	rr := &refreshTokenRequest{subject: BuildSubject("88"), tenantID: "7", sessionID: "sid-1"}
	refreshClaims, err := storage.GetPrivateClaimsFromRequest(ctx, rr, []string{model.ScopeOpenID})
	if err != nil {
		t.Fatalf("GetPrivateClaimsFromRequest(refresh) failed: %v", err)
	}
	if got := refreshClaims["user_id"]; got != "51" {
		t.Fatalf("refresh must keep user_id 51, got %v", got)
	}
}
