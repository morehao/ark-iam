package svcuser

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/ark-iam/platformadmin/testutil"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gobject"
)

func TestUserPageListPassesIsSuspendedZeroFilterToDAOCondition(t *testing.T) {
	ginCtx, _ := gin.CreateTestContext(nil)
	ginCtx.Set(gcontext.KeyTenantID, uint(23))

	db := testutil.SetupSQLite(t, &model.UserEntity{})
	// 同租户下挂起/未挂起用户各一，isSuspended=0 过滤应只返回未挂起的
	if err := testSeedUser(db, &model.UserEntity{
		TenantID:    23,
		PersonID:    1,
		Name:        "active-user",
		Profile:     []byte("{}"),
		CustomData:  []byte("{}"),
		IsSuspended: 0,
	}); err != nil {
		t.Fatalf("seed active user: %v", err)
	}
	if err := testSeedUser(db, &model.UserEntity{
		TenantID:    23,
		PersonID:    2,
		Name:        "suspended-user",
		Profile:     []byte("{}"),
		CustomData:  []byte("{}"),
		IsSuspended: 1,
	}); err != nil {
		t.Fatalf("seed suspended user: %v", err)
	}

	isSuspended := int8(0)
	svc := &userSvc{}
	resp, err := svc.PageList(ginCtx, &dtouser.UserPageListReq{
		PageQuery:   gobject.PageQuery{Page: 1, PageSize: 10},
		TenantID:    23,
		IsSuspended: &isSuspended,
	})
	if err != nil {
		t.Fatalf("PageList returned error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected 1 active user, got total %d", resp.Total)
	}
	if len(resp.List) != 1 || resp.List[0].Name != "active-user" {
		t.Fatalf("expected active-user only, got %+v", resp.List)
	}
}
