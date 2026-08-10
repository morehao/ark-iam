package svcapplicationclient

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gerror"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteSystemApplicationClient(t *testing.T) {
	dsn := fmt.Sprintf("file:application_client_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_ = db.AutoMigrate(&model.ApplicationClientEntity{})

	oldNew := newApplicationClientDAO
	defer func() { newApplicationClientDAO = oldNew }()
	newApplicationClientDAO = func() *dao.ApplicationClientDao {
		return dao.NewApplicationClientDaoWithDB(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) })
	}

	_ = db.Create(&model.ApplicationClientEntity{
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

	svc := NewApplicationClientSvc()
	err = svc.Delete(newOAuthDeleteCtx(1, 0), &dtoapplicationclient.DeleteReq{ApplicationClientID: 1})
	if err == nil {
		t.Fatal("expected error for system-built-in oauth client")
	}
	gerr, ok := err.(*gerror.Error)
	if !ok || gerr.Code != int(code.ApplicationClientSystemBuiltInErr) {
		t.Fatalf("expected ApplicationClientSystemBuiltInErr, got %v", err)
	}
}

func newOAuthDeleteCtx(tenantID, userID uint) *gin.Context {
	ctx := &gin.Context{}
	ctx.Set("tenantID", tenantID)
	ctx.Set("userID", userID)
	return ctx
}

func TestDeleteNonSystemApplicationClient(t *testing.T) {
	repo := &stubApplicationClientDeleteRepo{
		getByIDEntity: &model.ApplicationClientEntity{
			Model:    gorm.Model{ID: 2},
			TenantID: 1,
			ClientID: "y",
			Name:     "Blog Client",
			IsSystem: 0,
		},
	}
	installApplicationClientDeleteRepo(t, repo)

	svc := NewApplicationClientSvc()
	err := svc.Delete(newOAuthDeleteCtx(1, 7), &dtoapplicationclient.DeleteReq{ApplicationClientID: 2})
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

type stubApplicationClientDeleteRepo struct {
	getByIDEntity *model.ApplicationClientEntity
	getByIDErr    error
	deletedID     uint
	deletedBy     uint
}

func (r *stubApplicationClientDeleteRepo) GetByID(ctx context.Context, id uint) (*model.ApplicationClientEntity, error) {
	return r.getByIDEntity, r.getByIDErr
}

func (r *stubApplicationClientDeleteRepo) GetByCond(ctx context.Context, cond gormdao.Cond) (*model.ApplicationClientEntity, error) {
	return nil, nil
}

func (r *stubApplicationClientDeleteRepo) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.ApplicationClientEntityList, int64, error) {
	return nil, 0, nil
}

func (r *stubApplicationClientDeleteRepo) GetSecretByID(ctx context.Context, id uint) (*model.ApplicationClientSecretEntity, error) {
	return nil, nil
}

func (r *stubApplicationClientDeleteRepo) DeleteSecret(ctx context.Context, id, userID uint) error {
	return nil
}

func (r *stubApplicationClientDeleteRepo) Delete(ctx context.Context, id, userID uint) error {
	r.deletedID = id
	r.deletedBy = userID
	return nil
}

func installApplicationClientDeleteRepo(t *testing.T, repo applicationClientScopeRepository) {
	t.Helper()
	prev := newApplicationClientScopeRepo
	newApplicationClientScopeRepo = func() applicationClientScopeRepository {
		return repo
	}
	t.Cleanup(func() {
		newApplicationClientScopeRepo = prev
	})
}

var _ applicationClientScopeRepository = (*stubApplicationClientDeleteRepo)(nil)
