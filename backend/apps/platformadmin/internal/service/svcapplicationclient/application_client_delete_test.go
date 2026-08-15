package svcapplicationclient

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
	"gorm.io/datatypes"
)

// newOAuthDeleteCtx 构造带租户与操作人上下文的 gin.Context。
func newOAuthDeleteCtx(tenantID, userID string) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, tenantID)
	ctx.Set(gcontext.KeyUserID, userID)
	return ctx
}

// newTestClientEntity 构造一条完整的应用客户端记录（IsSystem 按需传入，其余取常规默认值）。
func newTestClientEntity(name, clientID string, isSystem bool) *model.ApplicationClientEntity {
	return &model.ApplicationClientEntity{
		TenantID:                "1",
		Code:                   clientID,
		Name:                    name,
		RedirectURIs:            datatypes.JSON("[]"),
		PostLogoutRedirectURIs:  datatypes.JSON("[]"),
		GrantTypes:              datatypes.JSON("[]"),
		ResponseTypes:           datatypes.JSON("[]"),
		TokenEndpointAuthMethod: "client_secret_basic",
		AllowedOrigins:          datatypes.JSON("[]"),
		DefaultScopes:           datatypes.JSON("[]"),
		Status:                  "enable",
		Type:                    "first_party",
		IsSystem:                isSystem,
	}
}

func TestDeleteSystemApplicationClient(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{}, &model.ApplicationClientSecretEntity{})

	entity := newTestClientEntity("System Client", "x", true)
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewApplicationClientSvc()
	err := svc.Delete(newOAuthDeleteCtx("1", "0"), &dtoapplicationclient.ApplicationClientDeleteReq{ApplicationClientID: entity.ID})
	if err == nil {
		t.Fatal("expected error for system-built-in oauth client")
	}
	if gerror.GetCode(err) != int(code.ApplicationClientSystemBuiltInErr) {
		t.Fatalf("expected ApplicationClientSystemBuiltInErr, got %v", err)
	}
}

func TestDeleteNonSystemApplicationClient(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.ApplicationClientEntity{}, &model.ApplicationClientSecretEntity{})

	entity := newTestClientEntity("Blog Client", "y", false)
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	ctx := newOAuthDeleteCtx("1", "7")
	svc := NewApplicationClientSvc()
	if err := svc.Delete(ctx, &dtoapplicationclient.ApplicationClientDeleteReq{ApplicationClientID: entity.ID}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// 真实 dao 断言：软删除后按 ID 查不到
	got, err := dao.NewApplicationClientDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil && got.ID != "" {
		t.Fatalf("expected client soft-deleted, got %+v", got)
	}

	// 删除人应写入 7（对应原 stub 断言 deletedBy == 7）
	var deleted model.ApplicationClientEntity
	if err := db.Unscoped().Where("id = ?", entity.ID).First(&deleted).Error; err != nil {
		t.Fatalf("query deleted row: %v", err)
	}
	if deleted.DeletedBy != "7" {
		t.Fatalf("expected deletedBy 7, got %s", deleted.DeletedBy)
	}
}
