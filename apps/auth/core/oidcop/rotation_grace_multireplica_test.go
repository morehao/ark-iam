package oidcop

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// TestRefreshRotationGraceSharedAcrossReplicas 回归 P0-2：
// auth 以多副本部署（共享同一 Redis + PostgreSQL）时，同一把旧 refresh token 的重试可能落到
// 不同副本。宽限窗口若只在进程内，B 副本读不到 A 副本的轮换结果，就会把重试误判为复用、
// 撤销整个 token 家族，把用户踢下线。本用例用两个独立的 PersistentStore（模拟两个副本）共享
// 同一 miniredis 与同一 SQLite 库，验证第二次（落到另一副本的）刷新被幂等重放、家族不撤销。
func TestRefreshRotationGraceSharedAcrossReplicas(t *testing.T) {
	// 共享 Redis（模拟认证集群共享的那一个）。
	mr := miniredis.RunT(t)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = cli.Close() })
	oldCli := dbclient.RedisCli
	dbclient.RedisCli = cli
	t.Cleanup(func() { dbclient.RedisCli = oldCli })

	// 共享数据库：两个副本连同一个库。
	dsn := fmt.Sprintf("file:rt_multireplica_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.UserEntity{}, &model.RefreshTokenEntity{}, &model.ApplicationClientEntity{}, &model.TenantEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	now := time.Now()
	if err := db.Create(&model.UserEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "40"}},
		TenantID:   "1", PersonID: "95", JoinedAt: &now,
	}).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if err := db.Create(&model.TenantEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "1"}},
		Code:       "t_1", Name: "tenant-1",
		Status: model.TenantStatusActive, Type: model.TenantTypeCustomer,
	}).Error; err != nil {
		t.Fatalf("insert tenant: %v", err)
	}
	if err := db.Create(&model.ApplicationClientEntity{
		BaseEntity: gormdao.BaseEntity{StringID: gormdao.StringID{ID: "acl-mr"}},
		Code:       "client-mr", TenantID: "1",
	}).Error; err != nil {
		t.Fatalf("insert application client: %v", err)
	}

	// 两个副本：各自的 PersistentStore（各自的进程内 map），但共享 DB 与 Redis。
	replicaA := newRotationReplicaStore(db)
	replicaB := newRotationReplicaStore(db)

	ctx := context.Background()

	// 首次授权（在 A 上）：拿到第一把 refresh token。
	authReq := &AuthRequest{Subject: BuildSubject("95"), ClientID: "client-mr", TenantID: "1"}
	_, firstRefresh, _, err := replicaA.CreateAccessAndRefreshTokens(ctx, authReq, "")
	if err != nil {
		t.Fatalf("initial issue on A failed: %v", err)
	}

	// 第一次刷新落到 A：轮换成功，旧 token 被撤销、新 token 写入 Redis 共享宽限缓存。
	reqA, err := replicaA.TokenRequestByRefreshToken(ctx, firstRefresh)
	if err != nil {
		t.Fatalf("A TokenRequestByRefreshToken failed: %v", err)
	}
	_, rotated, _, err := replicaA.CreateAccessAndRefreshTokens(ctx, reqA, firstRefresh)
	if err != nil {
		t.Fatalf("A rotation failed: %v", err)
	}
	if rotated == "" || rotated == firstRefresh {
		t.Fatalf("rotation must issue a new refresh token, got %q", rotated)
	}

	// 同一把旧 token 的重试落到另一副本 B（A 的进程内宽限缓存对 B 不可见）。
	// 必须被幂等重放成同一把新 token，而不是被判复用、撤销家族。
	reqB, err := replicaB.TokenRequestByRefreshToken(ctx, firstRefresh)
	if err != nil {
		t.Fatalf("B must read the just-rotated token inside the shared grace window, got %v", err)
	}
	_, replayed, _, err := replicaB.CreateAccessAndRefreshTokens(ctx, reqB, firstRefresh)
	if err != nil {
		t.Fatalf("B replicate retry must NOT be treated as reuse (user would be logged out), got %v", err)
	}
	if replayed != rotated {
		t.Fatalf("cross-replica replay must return the same refresh token: want %q, got %q", rotated, replayed)
	}

	// 家族仍然有效：rotated 可继续使用，且库里活跃 refresh token 恰为 1（未被误撤销）。
	if _, err := replicaA.TokenRequestByRefreshToken(ctx, rotated); err != nil {
		t.Fatalf("token family must stay alive after cross-replica replay, got %v", err)
	}
	var active int64
	if err := db.Table(model.TableNameRefreshToken).
		Where("person_id = ? AND revoked_at IS NULL", "95").Count(&active).Error; err != nil {
		t.Fatalf("count active refresh tokens: %v", err)
	}
	if active != 1 {
		t.Fatalf("expected exactly the latest refresh token active, got %d", active)
	}
}

// newRotationReplicaStore 构造一个连到给定 db 的 PersistentStore（模拟一个 auth 副本实例）。
// 只接线令牌轮换所需的 DAO（user / applicationClient / refreshToken / tenant）。
func newRotationReplicaStore(db *gorm.DB) *PersistentStore {
	getter := func(c context.Context) *gorm.DB { return db.WithContext(c) }
	s := NewPersistentStore()
	s.userDao = func(opts ...dao.DaoOption) *dao.UserDao {
		return &dao.UserDao{Dao: gormdao.NewDao[model.UserEntity, model.UserEntityList, string](
			model.TableNameUser, "UserDao", getter)}
	}
	s.applicationClientDao = func(opts ...dao.DaoOption) *dao.ApplicationClientDao {
		return dao.NewApplicationClientDao(dao.WithDBGetter(getter))
	}
	s.refreshTokenDao = func(opts ...dao.DaoOption) *dao.RefreshTokenDao {
		return &dao.RefreshTokenDao{Dao: gormdao.NewDao[model.RefreshTokenEntity, model.RefreshTokenEntityList, string](
			model.TableNameRefreshToken, "RefreshTokenDao", getter)}
	}
	s.tenantDao = func(opts ...dao.DaoOption) *dao.TenantDao {
		return &dao.TenantDao{Dao: gormdao.NewDao[model.TenantEntity, model.TenantEntityList, string](
			model.TableNameTenant, "TenantDao", getter)}
	}
	s.db = getter
	return s
}
