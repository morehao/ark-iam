package svcdomain

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtodomain"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestDB 打开独立的内存 SQLite，并注册为全局 iam 库，使 service 内部直接
// dao.NewXxxDao() 的调用自动落到测试库，无需任何注入 seam。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:svcdomain_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.DomainEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	dbclient.RegisterDBForTest(dbclient.ServiceNameIam, db)
	t.Cleanup(func() {
		dbclient.ClearDBForTest(dbclient.ServiceNameIam)
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func newGinCtx(tenantID, userID string) *gin.Context {
	ctx, _ := gin.CreateTestContext(nil)
	ctx.Set(gcontext.KeyTenantID, tenantID)
	ctx.Set(gcontext.KeyUserID, userID)
	return ctx
}

func TestDomainSvc_Create_Success(t *testing.T) {
	newTestDB(t)
	ctx := newGinCtx("10", "100")

	svc := NewDomainSvc()
	resp, err := svc.Create(ctx, &dtodomain.DomainCreateReq{Domain: "example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID == "" {
		t.Fatalf("expected non-zero id")
	}

	entity, err := dao.NewDomainDao().GetByID(ctx, resp.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if entity == nil || entity.ID == "" {
		t.Fatalf("expected persisted domain")
	}
	if entity.Domain != "example.com" {
		t.Fatalf("expected domain example.com, got %s", entity.Domain)
	}
	if entity.TenantID != "10" {
		t.Fatalf("expected tenantID 10, got %s", entity.TenantID)
	}
	if entity.IsVerified {
		if entity.IsVerified { t.Fatalf("expected isVerified false, got true") }
	}
	if entity.CreatedBy != "100" {
		t.Fatalf("expected createdBy 100, got %s", entity.CreatedBy)
	}
}

func TestDomainSvc_Create_DomainAlreadyExists(t *testing.T) {
	db := newTestDB(t)
	ctx := newGinCtx("10", "100")

	if err := db.Create(&model.DomainEntity{TenantID: "10", Domain: "example.com"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewDomainSvc()
	resp, err := svc.Create(ctx, &dtodomain.DomainCreateReq{Domain: "example.com"})
	if err == nil {
		t.Fatalf("expected error for duplicate domain, got resp=%+v", resp)
	}
	if gerror.GetCode(err) != int(code.DomainAlreadyExistError) {
		t.Fatalf("expected DomainAlreadyExistError, got %v", err)
	}
}

func TestDomainSvc_Create_EmptyDomain(t *testing.T) {
	newTestDB(t)
	ctx := newGinCtx("10", "100")

	svc := NewDomainSvc()
	resp, err := svc.Create(ctx, &dtodomain.DomainCreateReq{Domain: ""})
	if err == nil {
		t.Fatalf("expected error for empty domain, got resp=%+v", resp)
	}
}

func TestDomainSvc_PageList(t *testing.T) {
	db := newTestDB(t)
	ctx := newGinCtx("10", "100")

	if err := db.Create(&model.DomainEntity{TenantID: "10", Domain: "alpha.com"}).Error; err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	if err := db.Create(&model.DomainEntity{TenantID: "10", Domain: "beta.com"}).Error; err != nil {
		t.Fatalf("seed beta: %v", err)
	}
	// 其他租户的数据不应出现
	if err := db.Create(&model.DomainEntity{TenantID: "20", Domain: "alpha-other.com"}).Error; err != nil {
		t.Fatalf("seed other tenant: %v", err)
	}

	svc := NewDomainSvc()
	resp, err := svc.PageList(ctx, &dtodomain.DomainPageListReq{Domain: "alpha"})
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
	db := newTestDB(t)
	ctx := newGinCtx("10", "100")

	entity := &model.DomainEntity{TenantID: "10", Domain: "example.com"}
	if err := db.Create(entity).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewDomainSvc()
	if err := svc.Delete(ctx, &dtodomain.DomainDeleteReq{DomainID: entity.ID}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 软删除后 GetByID 应查不到
	got, err := dao.NewDomainDao().GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil && got.ID != "" {
		t.Fatalf("expected domain soft-deleted, got %+v", got)
	}
}

func TestDomainSvc_Delete_NotExist(t *testing.T) {
	newTestDB(t)
	ctx := newGinCtx("10", "100")

	svc := NewDomainSvc()
	err := svc.Delete(ctx, &dtodomain.DomainDeleteReq{DomainID: "999"})
	if err == nil {
		t.Fatalf("expected error for non-existent domain")
	}
}
