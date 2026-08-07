package svcoauthclient

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtooauthclient"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/gerror"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteSystemOAuthClient(t *testing.T) {
	dsn := fmt.Sprintf("file:oauth_client_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_ = db.AutoMigrate(&model.OAuthClientEntity{})

	oldNew := newOAuthClientDAO
	defer func() { newOAuthClientDAO = oldNew }()
	newOAuthClientDAO = func() *dao.OAuthClientDao {
		return dao.NewOAuthClientDaoWithDB(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) })
	}

	_ = db.Create(&model.OAuthClientEntity{
		Model:                   gorm.Model{ID: 1},
		TenantID:                1,
		ClientID:                "x",
		Name:                    "System Client",
		RedirectURIs:            datatypes.JSON("[]"),
		PostLogoutRedirectURIs:  datatypes.JSON("[]"),
		GrantTypes:              datatypes.JSON("[]"),
		ResponseTypes:           datatypes.JSON("[]"),
		TokenEndpointAuthMethod: "client_secret_basic",
		AllowedOrigins:          datatypes.JSON("[]"),
		DefaultScopes:           datatypes.JSON("[]"),
		Status:                  "enable",
		Type:                    "first_party",
		IsSystem:                1,
	}).Error

	svc := NewOAuthClientSvc()
	err = svc.Delete(newOAuthDeleteCtx(1, 0), &dtooauthclient.DeleteReq{OAuthClientID: 1})
	if err == nil {
		t.Fatal("expected error for system-built-in oauth client")
	}
	gerr, ok := err.(*gerror.Error)
	if !ok || gerr.Code != int(code.OAuthClientSystemBuiltInErr) {
		t.Fatalf("expected OAuthClientSystemBuiltInErr, got %v", err)
	}
}

func newOAuthDeleteCtx(tenantID, userID uint) *gin.Context {
	ctx := &gin.Context{}
	ctx.Set("tenantID", tenantID)
	ctx.Set("userID", userID)
	return ctx
}
