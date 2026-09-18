package svctenant

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/testutil"
)

// newTenantScopeGinCtx 构造带租户作用域的测试 gin.Context；tenantID 为空表示平台侧
// 「全部租户」视角（按主键列表、建租户链路）。同时挂上真实 Request（生产链路必有），
// 避免服务内 ctx.Request.Context() 空指针。
func newTenantScopeGinCtx(tenantID string) *gin.Context {
	return testutil.NewGinCtx(tenantID, "")
}

// TestTenantScopedUserInvisibleAcrossTenants 租户隔离插件（fail-closed）：
// ctx 声明的租户作用域与行归属不一致时，即使按主键直查也读不到该行
// （tenant_user 含 tenant_id，插件自动注入 tenant_id 过滤）。
// 原「日志详情拒绝跨租户实体」用例随 log 表下线删除，本用例以 tenant_user 承担同一不变式。
func TestTenantScopedUserInvisibleAcrossTenants(t *testing.T) {
	db := testutil.SetupSQLite(t, &model.UserEntity{})
	ctx := newTenantScopeGinCtx("71")

	user := &model.UserEntity{TenantID: "91", Name: "跨租户成员", UserType: model.UserTypeMember}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed tenant user: %v", err)
	}

	got, err := dao.NewUserDao().GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if got != nil && got.ID != "" {
		t.Fatalf("跨租户 tenant_user 不得可见, got %+v", got)
	}
}
