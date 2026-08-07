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
	"github.com/morehao/golib/dbaccess/gormdao"
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

func TestDeleteNonSystemOAuthClient(t *testing.T) {
	repo := &stubOAuthClientDeleteRepo{
		getByIDEntity: &model.OAuthClientEntity{
			Model:    gorm.Model{ID: 2},
			TenantID: 1,
			ClientID: "y",
			Name:     "Blog Client",
			IsSystem: 0,
		},
	}
	installOAuthClientDeleteRepo(t, repo)

	svc := NewOAuthClientSvc()
	err := svc.Delete(newOAuthDeleteCtx(1, 7), &dtooauthclient.DeleteReq{OAuthClientID: 2})
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if repo.deletedID != 2 {
		t.Fatalf("expected delete by id 2, got %d", repo.deletedID)
	}
	if repo.deletedBy != 7 {
		t.Fatalf("expected deletedBy 7, got %d", repo.deletedBy)
	}
}

type stubOAuthClientDeleteRepo struct {
	getByIDEntity *model.OAuthClientEntity
	getByIDErr    error
	deletedID     uint
	deletedBy     uint
}

func (r *stubOAuthClientDeleteRepo) GetByID(ctx context.Context, id uint) (*model.OAuthClientEntity, error) {
	return r.getByIDEntity, r.getByIDErr
}

func (r *stubOAuthClientDeleteRepo) GetByCond(ctx context.Context, cond gormdao.Cond) (*model.OAuthClientEntity, error) {
	return nil, nil
}

func (r *stubOAuthClientDeleteRepo) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.OAuthClientEntityList, int64, error) {
	return nil, 0, nil
}

func (r *stubOAuthClientDeleteRepo) GetSecretByID(ctx context.Context, id uint) (*model.OAuthClientSecretEntity, error) {
	return nil, nil
}

func (r *stubOAuthClientDeleteRepo) DeleteSecret(ctx context.Context, id, userID uint) error {
	return nil
}

func (r *stubOAuthClientDeleteRepo) Delete(ctx context.Context, id, userID uint) error {
	r.deletedID = id
	r.deletedBy = userID
	return nil
}

func installOAuthClientDeleteRepo(t *testing.T, repo oauthClientScopeRepository) {
	t.Helper()
	prev := newOAuthClientScopeRepo
	newOAuthClientScopeRepo = func() oauthClientScopeRepository {
		return repo
	}
	t.Cleanup(func() {
		newOAuthClientScopeRepo = prev
	})
}

var _ oauthClientScopeRepository = (*stubOAuthClientDeleteRepo)(nil)
