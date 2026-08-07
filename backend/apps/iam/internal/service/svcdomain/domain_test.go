package svcdomain

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtodomain"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type stubDomainRepo struct {
	inserted    *model.DomainEntity
	getByID     *model.DomainEntity
	pageList    model.DomainEntityList
	total       int64
	byTenantAnd *model.DomainEntity
	err         error
}

func (r *stubDomainRepo) Insert(ctx context.Context, entity *model.DomainEntity) error {
	entity.ID = 1
	clone := *entity
	r.inserted = &clone
	return r.err
}

func (r *stubDomainRepo) GetByID(ctx context.Context, id uint) (*model.DomainEntity, error) {
	return r.getByID, r.err
}

func (r *stubDomainRepo) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.DomainEntityList, int64, error) {
	return r.pageList, r.total, r.err
}

func (r *stubDomainRepo) GetByTenantAndDomain(ctx context.Context, tenantID uint, domain string) (*model.DomainEntity, error) {
	return r.byTenantAnd, r.err
}

func (r *stubDomainRepo) UpdateMap(ctx context.Context, id uint, updateMap map[string]any) error {
	return r.err
}

func (r *stubDomainRepo) Delete(ctx context.Context, id uint, deletedBy uint) error {
	return r.err
}

func installDomainRepo(t *testing.T, repo *stubDomainRepo) {
	t.Helper()
	prev := newDomainRepo
	newDomainRepo = func() domainRepository { return repo }
	t.Cleanup(func() { newDomainRepo = prev })
}

func TestDomainSvc_Create_Success(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(10))
	ginCtx.Set(gcontext.KeyUserID, uint(100))

	repo := &stubDomainRepo{}
	installDomainRepo(t, repo)

	svc := &domainSvc{}
	resp, err := svc.Create(ginCtx, &dtodomain.CreateDomainReq{Domain: "example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != 1 {
		t.Fatalf("expected id 1, got %d", resp.ID)
	}
	if repo.inserted == nil {
		t.Fatalf("expected insert to be called")
	}
	if repo.inserted.Domain != "example.com" {
		t.Fatalf("expected domain example.com, got %s", repo.inserted.Domain)
	}
	if repo.inserted.TenantID != 10 {
		t.Fatalf("expected tenantID 10, got %d", repo.inserted.TenantID)
	}
}

func TestDomainSvc_Create_DomainAlreadyExists(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(10))
	ginCtx.Set(gcontext.KeyUserID, uint(100))

	repo := &stubDomainRepo{
		byTenantAnd: &model.DomainEntity{Model: gorm.Model{ID: 5}, Domain: "example.com", TenantID: 10},
	}
	installDomainRepo(t, repo)

	svc := &domainSvc{}
	resp, err := svc.Create(ginCtx, &dtodomain.CreateDomainReq{Domain: "example.com"})
	if err == nil {
		t.Fatalf("expected error for duplicate domain, got resp=%+v", resp)
	}
}

func TestDomainSvc_Create_EmptyDomain(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(10))
	ginCtx.Set(gcontext.KeyUserID, uint(100))

	svc := &domainSvc{}
	resp, err := svc.Create(ginCtx, &dtodomain.CreateDomainReq{Domain: ""})
	if err == nil {
		t.Fatalf("expected error for empty domain, got resp=%+v", resp)
	}
}

func TestDomainSvc_PageList(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(10))

	repo := &stubDomainRepo{
		pageList: model.DomainEntityList{
			{Model: gorm.Model{ID: 1}, TenantID: 10, Domain: "alpha.com", IsVerified: 1},
		},
		total: 1,
	}
	installDomainRepo(t, repo)

	svc := &domainSvc{}
	resp, err := svc.PageList(ginCtx, &dtodomain.DomainPageListReq{Domain: "alpha"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Total)
	}
	if len(resp.List) != 1 || resp.List[0].Domain != "alpha.com" {
		t.Fatalf("expected alpha.com, got %+v", resp.List)
	}
}

func TestDomainSvc_Delete_Success(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(10))
	ginCtx.Set(gcontext.KeyUserID, uint(100))

	repo := &stubDomainRepo{
		getByID: &model.DomainEntity{Model: gorm.Model{ID: 1}, TenantID: 10, Domain: "example.com"},
	}
	installDomainRepo(t, repo)

	svc := &domainSvc{}
	err := svc.Delete(ginCtx, &dtodomain.DeleteDomainReq{ID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDomainSvc_Delete_NotExist(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(10))
	ginCtx.Set(gcontext.KeyUserID, uint(100))

	repo := &stubDomainRepo{}
	installDomainRepo(t, repo)

	svc := &domainSvc{}
	err := svc.Delete(ginCtx, &dtodomain.DeleteDomainReq{ID: 999})
	if err == nil {
		t.Fatalf("expected error for non-existent domain")
	}
}
