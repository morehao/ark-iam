package svcapplication

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/gerror"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteSystemApplication(t *testing.T) {
	dsn := fmt.Sprintf("file:app_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	_ = db.AutoMigrate(&model.ApplicationEntity{})

	oldNew := newApplicationRepo
	defer func() { newApplicationRepo = oldNew }()
	newApplicationRepo = func() applicationRepository {
		return dao.NewApplicationDaoWithDB(func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) })
	}

	_ = db.Create(&model.ApplicationEntity{
		Model:    gorm.Model{ID: 1},
		Code:     "admin",
		Name:     "管理后台",
		IsSystem: 1,
	}).Error

	svc := NewApplicationSvc()
	err = svc.Delete(newDeleteCtx(0), &dtoapplication.DeleteReq{AppID: 1})
	if err == nil {
		t.Fatal("expected error for system-built-in application")
	}
	gerr, ok := err.(*gerror.Error)
	if !ok || gerr.Code != int(code.ApplicationSystemBuiltInErr) {
		t.Fatalf("expected ApplicationSystemBuiltInErr, got %v", err)
	}
}

func newDeleteCtx(userID uint) *gin.Context {
	ctx := &gin.Context{}
	ctx.Set("tenantID", uint(1))
	ctx.Set("userID", userID)
	return ctx
}
