package svcuser

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

func TestUserIdentityPageListPassesFiltersToDAOAndKeepsDAOTotal(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(23))
	repo := &stubUserIdentityRepo{
		pageList: model.UserIdentityEntityList{
			{
				Model:           gorm.Model{ID: 1, UpdatedAt: time.Unix(1700000000, 0)},
				PersonID:        101,
				Issuer:          "issuer-a",
				ExternalSubject: "external-1",
				Detail:          []byte(`{"name":"first"}`),
			},
		},
		total: 7,
	}
	installUserIdentityRepo(t, repo)

	svc := &userIdentitySvc{}
	resp, err := svc.PageList(ginCtx, &dtouser.UserIdentityPageListReq{
		PageQuery:  gobject.PageQuery{Page: 2, PageSize: 5},
		TenantID:   23,
		UserID:     101,
		Issuer:     "issuer-a",
		IdentityID: "external-1",
	})
	if err != nil {
		t.Fatalf("PageList returned error: %v", err)
	}
	if repo.lastCond == nil {
		t.Fatalf("expected DAO condition to be captured")
	}
	if repo.lastCond.PersonID != 101 {
		t.Fatalf("expected person id 101, got %d", repo.lastCond.PersonID)
	}

	if repo.lastCond.Issuer != "issuer-a" {
		t.Fatalf("expected issuer issuer-a, got %q", repo.lastCond.Issuer)
	}
	if repo.lastCond.ExternalSubject != "external-1" {
		t.Fatalf("expected external subject external-1, got %q", repo.lastCond.ExternalSubject)
	}
	if repo.lastCond.BaseCond == nil || repo.lastCond.Page != 2 || repo.lastCond.PageSize != 5 {
		t.Fatalf("expected page condition {2,5}, got %#v", repo.lastCond.BaseCond)
	}
	if resp.Total != 7 {
		t.Fatalf("expected total 7 from DAO, got %d", resp.Total)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 list item, got %d", len(resp.List))
	}
}

func TestUserIdentityGetByUserUsesTenantScopedDAOCondition(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(45))
	relatedUserRepo := &stubUserIdentityUserRepo{detail: &model.UserEntity{Model: gorm.Model{ID: 202}, TenantID: 45, PersonID: 302}}
	installUserIdentityUserRepo(t, relatedUserRepo)
	repo := &stubUserIdentityRepo{
		pageList: model.UserIdentityEntityList{
			{
				Model:           gorm.Model{ID: 2, UpdatedAt: time.Unix(1700000001, 0)},
				PersonID:        302,
				Issuer:          "issuer-b",
				ExternalSubject: "external-2",
				Detail:          []byte(`{"name":"second"}`),
			},
		},
		total: 3,
	}
	installUserIdentityRepo(t, repo)

	svc := &userIdentitySvc{}
	resp, err := svc.GetByUser(ginCtx, &dtouser.UserIdentityByUserReq{UserID: 202})
	if err != nil {
		t.Fatalf("GetByUser returned error: %v", err)
	}
	if repo.lastCond == nil {
		t.Fatalf("expected DAO condition to be captured")
	}
	if repo.lastCond.PersonID != 302 {
		t.Fatalf("expected mapped person id 302, got %d", repo.lastCond.PersonID)
	}
	if repo.lastCond.BaseCond != nil {
		t.Fatalf("expected no pagination base condition, got %#v", repo.lastCond.BaseCond)
	}
	if resp.Total != 3 {
		t.Fatalf("expected total 3 from DAO, got %d", resp.Total)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 list item, got %d", len(resp.List))
	}
}

func TestUserIdentityDetailUsesTenantUserToResolvePerson(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(84))
	installUserIdentityUserRepo(t, &stubUserIdentityUserRepo{detail: &model.UserEntity{Model: gorm.Model{ID: 9}, TenantID: 84, PersonID: 902}})
	repo := &stubUserIdentityRepo{detail: &model.UserIdentityEntity{Model: gorm.Model{ID: 11}, PersonID: 902, Issuer: "issuer-a", ExternalSubject: "external-a", Detail: []byte(`{"name":"mapped"}`)}}
	installUserIdentityRepo(t, repo)

	svc := &userIdentitySvc{}
	resp, err := svc.Detail(ginCtx, &dtouser.UserIdentityDetailReq{UserIdentityID: 9})
	if err != nil {
		t.Fatalf("Detail returned error: %v", err)
	}
	if resp == nil || resp.UserID != 902 {
		t.Fatalf("expected mapped person response, got %#v", resp)
	}
}

type stubUserIdentityRepo struct {
	detail   *model.UserIdentityEntity
	pageList model.UserIdentityEntityList
	total    int64
	err      error
	lastCond *dao.UserIdentityCond
}

func (r *stubUserIdentityRepo) Insert(ctx context.Context, entity *model.UserIdentityEntity) error {
	return errors.New("unexpected call to Insert")
}

func (r *stubUserIdentityRepo) GetByID(ctx context.Context, id uint) (*model.UserIdentityEntity, error) {
	return r.detail, r.err
}

func (r *stubUserIdentityRepo) Delete(ctx context.Context, id uint, deletedBy uint) error {
	return errors.New("unexpected call to Delete")
}

func (r *stubUserIdentityRepo) UpdateMap(ctx context.Context, id uint, updates map[string]any) error {
	return errors.New("unexpected call to UpdateMap")
}

func (r *stubUserIdentityRepo) GetPageListByCond(ctx context.Context, cond *dao.UserIdentityCond) (model.UserIdentityEntityList, int64, error) {
	r.lastCond = cloneUserIdentityCond(cond)
	return r.pageList, r.total, r.err
}

func installUserIdentityRepo(t *testing.T, repo userIdentityRepository) {
	t.Helper()
	prev := newUserIdentityRepo
	prevSvc := newPersonIdentitySvc
	newUserIdentityRepo = func() userIdentityRepository {
		return repo
	}
	newPersonIdentitySvc = func() delegatedPersonIdentitySvc {
		return &stubDelegatedPersonIdentitySvc{repo: repo, userRepo: newUserIdentityUserRepo()}
	}
	t.Cleanup(func() {
		newUserIdentityRepo = prev
		newPersonIdentitySvc = prevSvc
	})
}

type stubUserIdentityUserRepo struct {
	detail *model.UserEntity
	err    error
}

func (r *stubUserIdentityUserRepo) GetByID(ctx context.Context, id uint) (*model.UserEntity, error) {
	return r.detail, r.err
}

func installUserIdentityUserRepo(t *testing.T, repo userIdentityUserResolver) {
	t.Helper()
	prev := newUserIdentityUserRepo
	newUserIdentityUserRepo = func() userIdentityUserResolver {
		return repo
	}
	t.Cleanup(func() {
		newUserIdentityUserRepo = prev
	})
}

func cloneUserIdentityCond(cond *dao.UserIdentityCond) *dao.UserIdentityCond {
	if cond == nil {
		return nil
	}
	clone := *cond
	if cond.BaseCond != nil {
		base := *cond.BaseCond
		clone.BaseCond = &base
	}
	return &clone
}

type stubDelegatedPersonIdentitySvc struct {
	repo     userIdentityRepository
	userRepo userIdentityUserResolver
}

func (s *stubDelegatedPersonIdentitySvc) Create(ctx *gin.Context, req *dtouser.UserIdentityCreateReq) (*dtouser.UserIdentityCreateResp, error) {
	return nil, errors.New("unexpected call to Create")
}

func (s *stubDelegatedPersonIdentitySvc) Delete(ctx *gin.Context, req *dtouser.UserIdentityDeleteReq) error {
	return errors.New("unexpected call to Delete")
}

func (s *stubDelegatedPersonIdentitySvc) Update(ctx *gin.Context, req *dtouser.UserIdentityUpdateReq) error {
	return errors.New("unexpected call to Update")
}

func (s *stubDelegatedPersonIdentitySvc) Detail(ctx *gin.Context, req *dtouser.UserIdentityDetailReq) (*dtouser.UserIdentityDetailResp, error) {
	// 模拟 personSvc.Detail 的租户可见性校验（identity 归属 person 需在当前租户有成员关系）
	if s.userRepo != nil {
		userEntity, err := s.userRepo.GetByID(ctx, req.UserIdentityID)
		if err != nil || userEntity == nil {
			return nil, err
		}
		if userEntity.TenantID != gincontext.GetTenantID(ctx) {
			return nil, code.GetError(code.UserIdentityNotExistError)
		}
	}
	entity, err := s.repo.GetByID(ctx, req.UserIdentityID)
	if err != nil || entity == nil {
		return nil, err
	}
	var detail any
	_ = json.Unmarshal(entity.Detail, &detail)
	return &dtouser.UserIdentityDetailResp{UserIdentityID: entity.ID, UserID: entity.PersonID, Issuer: entity.Issuer, IdentityID: entity.ExternalSubject, Detail: detail}, nil
}

func (s *stubDelegatedPersonIdentitySvc) PageList(ctx *gin.Context, req *dtouser.UserIdentityPageListReq) (*dtouser.UserIdentityPageListResp, error) {
	list, total, err := s.repo.GetPageListByCond(ctx, &dao.UserIdentityCond{BaseCond: &gormdao.BaseCond{Page: req.Page, PageSize: req.PageSize}, PersonID: req.UserID, Issuer: req.Issuer, ExternalSubject: req.IdentityID})
	if err != nil {
		return nil, err
	}
	respList := make([]dtouser.UserIdentityPageListItem, 0, len(list))
	for _, v := range list {
		var detail any
		_ = json.Unmarshal(v.Detail, &detail)
		respList = append(respList, dtouser.UserIdentityPageListItem{UserIdentityID: v.ID, UserID: v.PersonID, Issuer: v.Issuer, IdentityID: v.ExternalSubject, Detail: detail})
	}
	return &dtouser.UserIdentityPageListResp{List: respList, Total: total}, nil
}

func (s *stubDelegatedPersonIdentitySvc) GetByUser(ctx *gin.Context, req *dtouser.UserIdentityByUserReq) (*dtouser.UserIdentityPageListResp, error) {
	personID := req.UserID
	if s.userRepo != nil {
		userEntity, err := s.userRepo.GetByID(ctx, req.UserID)
		if err != nil || userEntity == nil {
			return nil, err
		}
		personID = userEntity.PersonID
	}
	list, total, err := s.repo.GetPageListByCond(ctx, &dao.UserIdentityCond{PersonID: personID})
	if err != nil {
		return nil, err
	}
	respList := make([]dtouser.UserIdentityPageListItem, 0, len(list))
	for _, v := range list {
		var detail any
		_ = json.Unmarshal(v.Detail, &detail)
		respList = append(respList, dtouser.UserIdentityPageListItem{UserIdentityID: v.ID, UserID: v.PersonID, Issuer: v.Issuer, IdentityID: v.ExternalSubject, Detail: detail})
	}
	return &dtouser.UserIdentityPageListResp{List: respList, Total: total}, nil
}

var _ userIdentityRepository = (*stubUserIdentityRepo)(nil)

var _ = gormdao.BaseCond{}
