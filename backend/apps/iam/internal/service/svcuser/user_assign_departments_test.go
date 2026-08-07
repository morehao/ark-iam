package svcuser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gerror"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAssignDepartmentsUsesTenantAndSkipsDuplicates(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = newAssignDepartmentsRequest(t)
	ginCtx.Set(gcontext.KeyTenantID, uint(23))
	ginCtx.Set(gcontext.KeyUserID, uint(88))

	db := newAssignDepartmentsTestDB(t)
	installTestIamDB(t, db)
	if err := db.Create(&model.UserDepartmentEntity{
		TenantID:     23,
		UserID:       100,
		DepartmentID: 2,
		IsPrimary:    0,
		CreatedBy:    1,
	}).Error; err != nil {
		t.Fatalf("seed duplicate relation: %v", err)
	}

	svc := &userSvc{}
	err := svc.AssignDepartments(ginCtx, &dtouser.AssignDepartmentsReq{UserID: 100, DepartmentIDs: []uint{1, 2, 3}})
	if err != nil {
		t.Fatalf("AssignDepartments returned error: %v", err)
	}

	var relations []model.UserDepartmentEntity
	if err := db.Order("department_id asc").Find(&relations).Error; err != nil {
		t.Fatalf("query relations: %v", err)
	}
	if len(relations) != 3 {
		t.Fatalf("expected 3 total relations after skipping duplicate, got %d", len(relations))
	}
	if relations[0].DepartmentID != 1 || relations[1].DepartmentID != 2 || relations[2].DepartmentID != 3 {
		t.Fatalf("unexpected department ids: %d, %d, %d", relations[0].DepartmentID, relations[1].DepartmentID, relations[2].DepartmentID)
	}
	if relations[0].TenantID != 23 || relations[2].TenantID != 23 {
		t.Fatalf("expected inserted tenant id to be 23, got %d and %d", relations[0].TenantID, relations[2].TenantID)
	}
	if relations[0].CreatedBy != 88 || relations[2].CreatedBy != 88 {
		t.Fatalf("expected inserted created_by to be 88, got %d and %d", relations[0].CreatedBy, relations[2].CreatedBy)
	}
	if relations[0].IsPrimary != 0 || relations[2].IsPrimary != 0 {
		t.Fatalf("expected inserted is_primary to stay 0, got %d and %d", relations[0].IsPrimary, relations[2].IsPrimary)
	}
}

func TestAssignDepartmentsReturnsErrorWhenInsertFailsInTransaction(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = newAssignDepartmentsRequest(t)
	ginCtx.Set(gcontext.KeyTenantID, uint(9))
	ginCtx.Set(gcontext.KeyUserID, uint(7))

	db := newAssignDepartmentsTestDB(t)
	installTestIamDB(t, db)
	if err := db.Callback().Create().Before("gorm:create").Register("test_fail_create", func(tx *gorm.DB) {
		_ = tx.AddError(errors.New("insert failed"))
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	defer func() {
		_ = db.Callback().Create().Remove("test_fail_create")
	}()

	svc := &userSvc{}
	err := svc.AssignDepartments(ginCtx, &dtouser.AssignDepartmentsReq{UserID: 100, DepartmentIDs: []uint{1}})
	assertAssignDepartmentsCode(t, err, code.UserDepartmentCreateError)
}

func TestAssignDepartmentsReturnsErrorWhenTransactionFails(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Request = newAssignDepartmentsRequest(t)
	ginCtx.Set(gcontext.KeyTenantID, uint(9))
	ginCtx.Set(gcontext.KeyUserID, uint(7))

	db := newAssignDepartmentsTestDB(t)
	installTestIamDB(t, db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql db: %v", err)
	}

	svc := &userSvc{}
	err = svc.AssignDepartments(ginCtx, &dtouser.AssignDepartmentsReq{UserID: 100, DepartmentIDs: []uint{1}})
	assertAssignDepartmentsCode(t, err, code.UserDepartmentCreateError)
}

func newAssignDepartmentsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeAssignDepartmentsTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.UserDepartmentEntity{}); err != nil {
		t.Fatalf("migrate relation table: %v", err)
	}
	return db
}

func installTestIamDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	prev := iamDBFromContext
	iamDBFromContext = func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}
	t.Cleanup(func() {
		iamDBFromContext = prev
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
}

func sanitizeAssignDepartmentsTestName(name string) string {
	replacer := strings.NewReplacer("/", "_", " ", "_", ":", "_")
	return replacer.Replace(name)
}

func assertAssignDepartmentsCode(t *testing.T, err error, want int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d, got nil", want)
	}
	gerr, ok := err.(*gerror.Error)
	if !ok {
		t.Fatalf("expected *gerror.Error, got %T", err)
	}
	if gerr.Code != want {
		t.Fatalf("expected error code %d, got %d", want, gerr.Code)
	}
}

func newAssignDepartmentsRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, "/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req
}
