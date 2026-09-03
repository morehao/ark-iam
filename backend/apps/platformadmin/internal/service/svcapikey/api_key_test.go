package svcapikey

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapikey"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
)

// TestPageListSupervisionOwnerResolution 验证监督视图：
//  1. 忽略上下文租户，跨租户可见；
//  2. 归属主体（服务账号/真实用户）与创建人名称被正确解析；
//  3. 支持按租户过滤。
func TestPageListSupervisionOwnerResolution(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApiKeyEntity{}, &model.UserEntity{}, &model.TenantEntity{})
	svc := NewApiKeySupervisionSvc() // 默认 dao getter → 落到 SetupSQLite 注册的全局 iam 测试库

	tenantA := &model.TenantEntity{Code: "tenant-a", Name: "租户A"}
	tenantB := &model.TenantEntity{Code: "tenant-b", Name: "租户B"}
	if err := db.Create(tenantA).Error; err != nil {
		t.Fatalf("seed tenantA: %v", err)
	}
	if err := db.Create(tenantB).Error; err != nil {
		t.Fatalf("seed tenantB: %v", err)
	}

	creatorA := &model.UserEntity{TenantID: tenantA.ID, UserType: model.UserTypeMember, Name: "A管理员", Profile: json.RawMessage(`{}`), CustomData: json.RawMessage(`{}`)}
	machineA := &model.UserEntity{TenantID: tenantA.ID, UserType: model.UserTypeMachine, Name: "A服务账号", Profile: json.RawMessage(`{}`), CustomData: json.RawMessage(`{}`)}
	creatorB := &model.UserEntity{TenantID: tenantB.ID, UserType: model.UserTypeMember, Name: "B管理员", Profile: json.RawMessage(`{}`), CustomData: json.RawMessage(`{}`)}
	for _, u := range []*model.UserEntity{creatorA, machineA, creatorB} {
		if err := db.Create(u).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}

	keyDao := dao.NewApiKeyDao()
	now := time.Now()
	seedKey := func(tenantID, ownerUserID, createdBy, name string) *model.ApiKeyEntity {
		k := &model.ApiKeyEntity{
			TenantID:    tenantID,
			OwnerUserID: ownerUserID,
			Name:        name,
			KeyHash:     "hash-" + name,
			KeyPrefix:   "prefix-" + name[:1],
			Scope:       json.RawMessage(`{}`),
			RevokedAt:   &now,
			CreatedBy:   createdBy,
		}
		if err := keyDao.Insert(context.Background(), k); err != nil {
			t.Fatalf("seed key %s: %v", name, err)
		}
		return k
	}
	// 个人密钥(归属真实用户A) + 服务账号密钥(A) + 租户B服务账号密钥
	seedKey(tenantA.ID, creatorA.ID, creatorA.ID, "A-Personal-Key")
	seedKey(tenantA.ID, machineA.ID, creatorA.ID, "A-Machine-Key")
	seedKey(tenantB.ID, creatorB.ID, creatorB.ID, "B-Personal-Key")

	sup, err := svc.PageListSupervision(newTestGinCtx(tenantA.ID), &dtoapikey.ApiKeySupervisionPageListReq{})
	if err != nil {
		t.Fatalf("PageListSupervision: %v", err)
	}
	if sup.Total != 3 || len(sup.List) != 3 {
		t.Fatalf("supervision should list keys across tenants, got total=%d list=%d", sup.Total, len(sup.List))
	}
	byName := map[string]dtoapikey.ApiKeySupervisionItem{}
	for _, item := range sup.List {
		byName[item.Name] = item
	}
	machine := byName["A-Machine-Key"]
	if machine.TenantName != "租户A" || machine.CreatorName != "A管理员" {
		t.Fatalf("unexpected tenant/creator resolution: %+v", machine)
	}
	if machine.OwnerName != "A服务账号" || machine.OwnerType != string(model.UserTypeMachine) {
		t.Fatalf("unexpected machine owner resolution: %+v", machine)
	}
	personal := byName["A-Personal-Key"]
	if personal.OwnerName != "A管理员" || personal.OwnerType != string(model.UserTypeMember) {
		t.Fatalf("unexpected member owner resolution: %+v", personal)
	}
	if personal.RevokedAt == 0 {
		t.Fatal("expected revokedAt echoed in supervision item")
	}

	// 按租户过滤
	filtered, err := svc.PageListSupervision(newTestGinCtx(tenantA.ID), &dtoapikey.ApiKeySupervisionPageListReq{TenantID: tenantB.ID})
	if err != nil {
		t.Fatalf("PageListSupervision filtered: %v", err)
	}
	if filtered.Total != 1 || filtered.List[0].Name != "B-Personal-Key" {
		t.Fatalf("filtered supervision mismatch: %+v", filtered.List)
	}
}

func newTestGinCtx(tenantID string) *gin.Context {
	ctx := &gin.Context{}
	ctx.Set(gcontext.KeyTenantID, tenantID)
	return ctx
}
