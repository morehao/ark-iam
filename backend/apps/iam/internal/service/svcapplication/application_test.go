package svcapplication

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

func TestListRolesReturnsAllApplicationRoles(t *testing.T) {
	createdAt1 := time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
	createdAt2 := time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC)

	stubAppRoleDao := &stubApplicationRoleListReader{
		list: model.ApplicationRoleEntityList{
			{ApplicationID: 9, RoleID: 101, Model: gormModel(createdAt1)},
			{ApplicationID: 9, RoleID: 102, Model: gormModel(createdAt2)},
		},
	}
	stubRoleDao := &stubRoleReader{
		roles: map[uint]*model.RoleEntity{
			101: {Model: gormModelWithID(101, time.Time{}), Name: "管理员", Code: "admin"},
			102: {Model: gormModelWithID(102, time.Time{}), Name: "审计员", Code: "auditor"},
		},
	}
	installApplicationListRolesStubs(t, stubAppRoleDao, stubRoleDao)

	gCtx, _ := gin.CreateTestContext(nil)
	svc := &applicationSvc{}

	resp, err := svc.ListRoles(gCtx, &dtoapplication.ApplicationRoleListReq{ApplicationID: 9})
	if err != nil {
		t.Fatalf("ListRoles returned error: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
	if len(resp.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(resp.Roles))
	}
	if resp.Roles[0].RoleID != 101 || resp.Roles[0].RoleName != "管理员" || resp.Roles[0].RoleCode != "admin" {
		t.Fatalf("unexpected first role: %+v", resp.Roles[0])
	}
	if resp.Roles[0].CreatedAt != "2026-05-09 10:00:00" {
		t.Fatalf("unexpected first createdAt: %s", resp.Roles[0].CreatedAt)
	}
	if resp.Roles[1].RoleID != 102 || resp.Roles[1].RoleName != "审计员" || resp.Roles[1].RoleCode != "auditor" {
		t.Fatalf("unexpected second role: %+v", resp.Roles[1])
	}
	if resp.Roles[1].CreatedAt != "2026-05-09 11:00:00" {
		t.Fatalf("unexpected second createdAt: %s", resp.Roles[1].CreatedAt)
	}
	if stubAppRoleDao.lastCond == nil || stubAppRoleDao.lastCond.ApplicationID != 9 {
		t.Fatalf("expected application role query by application 9, got %+v", stubAppRoleDao.lastCond)
	}
	if !containsAllRoleLookups(stubRoleDao.calls, 101, 102) {
		t.Fatalf("expected role lookups to include 101 and 102, got %+v", stubRoleDao.calls)
	}
	if resp.Total != int64(len(resp.Roles)) {
		t.Fatalf("expected total to equal roles length, got total=%d len=%d", resp.Total, len(resp.Roles))
	}
}

func TestListRolesSkipsMissingRoleRecords(t *testing.T) {
	createdAt1 := time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
	createdAt2 := time.Date(2026, 5, 9, 11, 0, 0, 0, time.UTC)

	stubAppRoleDao := &stubApplicationRoleListReader{
		list: model.ApplicationRoleEntityList{
			{ApplicationID: 9, RoleID: 101, Model: gormModel(createdAt1)},
			{ApplicationID: 9, RoleID: 102, Model: gormModel(createdAt2)},
		},
	}
	stubRoleDao := &stubRoleReader{
		roles: map[uint]*model.RoleEntity{
			101: {Model: gormModelWithID(101, time.Time{}), Name: "管理员", Code: "admin"},
		},
		errs: map[uint]error{
			102: errors.New("role not found"),
		},
	}
	installApplicationListRolesStubs(t, stubAppRoleDao, stubRoleDao)

	gCtx, _ := gin.CreateTestContext(nil)
	svc := &applicationSvc{}

	resp, err := svc.ListRoles(gCtx, &dtoapplication.ApplicationRoleListReq{ApplicationID: 9})
	if err != nil {
		t.Fatalf("ListRoles returned error: %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Total)
	}
	if len(resp.Roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(resp.Roles))
	}
	if resp.Roles[0].RoleID != 101 || resp.Roles[0].RoleName != "管理员" {
		t.Fatalf("unexpected role after skipping missing detail: %+v", resp.Roles[0])
	}
	if !containsAllRoleLookups(stubRoleDao.calls, 101, 102) {
		t.Fatalf("expected role lookups to include 101 and 102, got %+v", stubRoleDao.calls)
	}
	if resp.Total != int64(len(resp.Roles)) {
		t.Fatalf("expected total to equal roles length, got total=%d len=%d", resp.Total, len(resp.Roles))
	}
}

type stubApplicationRoleListReader struct {
	list     model.ApplicationRoleEntityList
	err      error
	lastCond *dao.ApplicationRoleCond
}

func (s *stubApplicationRoleListReader) GetListByCond(ctx context.Context, cond genericdao.Cond) (model.ApplicationRoleEntityList, error) {
	appRoleCond, _ := cond.(*dao.ApplicationRoleCond)
	s.lastCond = appRoleCond
	return s.list, s.err
}

type stubRoleReader struct {
	roles map[uint]*model.RoleEntity
	errs  map[uint]error
	calls []uint
}

func (s *stubRoleReader) GetByID(ctx context.Context, id uint) (*model.RoleEntity, error) {
	s.calls = append(s.calls, id)
	if err, ok := s.errs[id]; ok {
		return nil, err
	}
	return s.roles[id], nil
}

func installApplicationListRolesStubs(t *testing.T, appRoleDao applicationRoleListReader, roleDao roleReader) {
	t.Helper()
	prevAppRoleReader := newApplicationRoleListReader
	prevRoleReader := newRoleReader
	newApplicationRoleListReader = func() applicationRoleListReader {
		return appRoleDao
	}
	newRoleReader = func() roleReader {
		return roleDao
	}
	t.Cleanup(func() {
		newApplicationRoleListReader = prevAppRoleReader
		newRoleReader = prevRoleReader
	})
}

func gormModel(createdAt time.Time) gorm.Model {
	return gorm.Model{
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func gormModelWithID(id uint, createdAt time.Time) gorm.Model {
	base := gormModel(createdAt)
	base.ID = id
	return base
}

func containsAllRoleLookups(calls []uint, ids ...uint) bool {
	seen := make(map[uint]struct{}, len(calls))
	for _, id := range calls {
		seen[id] = struct{}{}
	}
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			return false
		}
	}
	return true
}
