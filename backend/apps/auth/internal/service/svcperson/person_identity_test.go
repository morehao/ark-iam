package svcperson

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/auth/internal/dto/dtouser"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"gorm.io/gorm"
)

func TestPersonIdentityCreatePersistsPersonID(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyUserID, uint(501))
	ginCtx.Set(gcontext.KeyTenantID, uint(88))

	repo := &stubPersonIdentityRepo{}
	installPersonIdentityRepo(t, repo)
	installPersonIdentityUserRepo(t, &stubPersonIdentityUserRepo{usersByPerson: model.UserEntityList{{Model: gorm.Model{ID: 601}, TenantID: 88, PersonID: 66}}})

	svc := &personSvc{}
	resp, err := svc.Create(ginCtx, &dtouser.UserIdentityCreateReq{
		TenantID:   88,
		UserID:     66,
		Issuer:     "https://issuer.example.com",
		IdentityID: "external-subject-1",
		Detail: map[string]any{
			"source": "oidc",
		},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if resp == nil || resp.UserIdentityID != 701 {
		t.Fatalf("expected created identity response, got %#v", resp)
	}
	if repo.inserted == nil {
		t.Fatal("expected identity insert to be called")
	}
	if repo.inserted.PersonID != 66 {
		t.Fatalf("expected person_id 66, got %d", repo.inserted.PersonID)
	}
	if repo.inserted.Issuer != "https://issuer.example.com" {
		t.Fatalf("expected issuer to be persisted, got %q", repo.inserted.Issuer)
	}
	if repo.inserted.ExternalSubject != "external-subject-1" {
		t.Fatalf("expected external subject to be persisted, got %q", repo.inserted.ExternalSubject)
	}
	if repo.inserted.CreatedBy != 501 {
		t.Fatalf("expected created_by 501, got %d", repo.inserted.CreatedBy)
	}
}

func TestPersonIdentityUpdateDoesNotPersistFakeUpdatedByWhenOperatorMissing(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(31))
	repo := &stubPersonIdentityRepo{detail: &model.UserIdentityEntity{Model: gorm.Model{ID: 9}, PersonID: 71, Detail: []byte(`{}`)}}
	userRepo := &stubPersonIdentityUserRepo{usersByPerson: model.UserEntityList{{Model: gorm.Model{ID: 101}, TenantID: 31, PersonID: 71}}}
	installPersonIdentityRepo(t, repo)
	installPersonIdentityUserRepo(t, userRepo)

	svc := &personSvc{}
	err := svc.Update(ginCtx, &dtouser.UserIdentityUpdateReq{UserIdentityID: 9, UserID: 71, Issuer: "issuer-a", IdentityID: "external-a", Detail: map[string]any{"k": "v"}})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if _, ok := repo.updated["updated_by"]; ok {
		t.Fatalf("expected updated_by to be omitted when operator missing, got %#v", repo.updated)
	}
}

func TestPersonIdentityDetailRejectsCrossTenantPerson(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(31))
	repo := &stubPersonIdentityRepo{detail: &model.UserIdentityEntity{Model: gorm.Model{ID: 9}, PersonID: 71, Detail: []byte(`{}`)}}
	userRepo := &stubPersonIdentityUserRepo{usersByPerson: model.UserEntityList{{Model: gorm.Model{ID: 101}, TenantID: 99, PersonID: 71}}}
	installPersonIdentityRepo(t, repo)
	installPersonIdentityUserRepo(t, userRepo)

	svc := &personSvc{}
	_, err := svc.Detail(ginCtx, &dtouser.UserIdentityDetailReq{UserIdentityID: 9})
	if err == nil {
		t.Fatal("expected cross-tenant person identity detail to fail")
	}
}

type stubPersonIdentityRepo struct {
	inserted *model.UserIdentityEntity
	updated  map[string]any
	detail   *model.UserIdentityEntity
	pageList model.UserIdentityEntityList
	total    int64
	err      error
	lastCond *dao.UserIdentityCond
}

func (r *stubPersonIdentityRepo) Insert(ctx context.Context, entity *model.UserIdentityEntity) error {
	clone := *entity
	clone.Model = gorm.Model{ID: 701}
	entity.ID = 701
	r.inserted = &clone
	return r.err
}

func (r *stubPersonIdentityRepo) GetByID(ctx context.Context, id uint) (*model.UserIdentityEntity, error) {
	return r.detail, r.err
}

func (r *stubPersonIdentityRepo) Delete(ctx context.Context, id uint, deletedBy uint) error {
	return r.err
}

func (r *stubPersonIdentityRepo) UpdateMap(ctx context.Context, id uint, updates map[string]any) error {
	r.updated = updates
	return r.err
}

func (r *stubPersonIdentityRepo) GetPageListByCond(ctx context.Context, cond *dao.UserIdentityCond) (model.UserIdentityEntityList, int64, error) {
	r.lastCond = cond
	return r.pageList, r.total, r.err
}

func installPersonIdentityRepo(t *testing.T, repo personIdentityRepository) {
	t.Helper()
	prev := newPersonIdentityRepo
	newPersonIdentityRepo = func() personIdentityRepository {
		return repo
	}
	t.Cleanup(func() {
		newPersonIdentityRepo = prev
	})
}

type stubPersonIdentityUserRepo struct {
	userByID       *model.UserEntity
	usersByPerson  model.UserEntityList
	err            error
	lastGetListCond *dao.UserCond
}

func (r *stubPersonIdentityUserRepo) GetByID(ctx context.Context, id uint) (*model.UserEntity, error) {
	return r.userByID, r.err
}

func (r *stubPersonIdentityUserRepo) GetListByCond(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, error) {
	clone := *cond
	r.lastGetListCond = &clone
	return r.usersByPerson, r.err
}

func installPersonIdentityUserRepo(t *testing.T, repo personIdentityUserRepository) {
	t.Helper()
	prev := newPersonIdentityUserRepo
	newPersonIdentityUserRepo = func() personIdentityUserRepository {
		return repo
	}
	t.Cleanup(func() {
		newPersonIdentityUserRepo = prev
	})
}
