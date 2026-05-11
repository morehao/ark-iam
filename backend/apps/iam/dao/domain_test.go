package dao

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newGenericDaoFromDB(db *gorm.DB, tableName, daoName string) *genericdao.GenericDao[model.DomainEntity, model.DomainEntityList] {
	return genericdao.NewGenericDao[model.DomainEntity, model.DomainEntityList](
		tableName, daoName,
		func(_ context.Context) *gorm.DB { return db },
	)
}

func TestDomainDao_InsertAndGetByID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:domaintest_insert?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.DomainEntity{}); err != nil {
		t.Fatalf("migrate domain: %v", err)
	}

dao := &DomainDao{
		GenericDao: newGenericDaoFromDB(db, model.TableNameDomain, "DomainDao"),
	}
	entity := &model.DomainEntity{
		TenantID:   1,
		Domain:     "example.com",
		IsVerified: 0,
		VerifiedAt: sql.NullTime{},
	}
	ctx := context.Background()
	if err := dao.Insert(ctx, entity); err != nil {
		t.Fatalf("insert domain: %v", err)
	}
	if entity.ID == 0 {
		t.Fatalf("expected non-zero ID after insert")
	}

	got, err := dao.GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.ID != entity.ID {
		t.Fatalf("expected domain id %d, got %+v", entity.ID, got)
	}
	if got.Domain != "example.com" {
		t.Fatalf("expected domain example.com, got %s", got.Domain)
	}
}

func TestDomainDao_GetByTenantAndDomain(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:domaintest_tenantdomain?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.DomainEntity{}); err != nil {
		t.Fatalf("migrate domain: %v", err)
	}

	dao := &DomainDao{
		GenericDao: newGenericDaoFromDB(db, model.TableNameDomain, "DomainDao"),
	}
	entity := &model.DomainEntity{
		TenantID:   1,
		Domain:     "example.com",
		IsVerified: 0,
	}
	ctx := context.Background()
	if err := dao.Insert(ctx, entity); err != nil {
		t.Fatalf("insert domain: %v", err)
	}

	got, err := dao.GetByTenantAndDomain(ctx, 1, "example.com")
	if err != nil {
		t.Fatalf("get by tenant and domain: %v", err)
	}
	if got == nil || got.ID != entity.ID {
		t.Fatalf("expected domain id %d, got %+v", entity.ID, got)
	}

	notFound, err := dao.GetByTenantAndDomain(ctx, 1, "notfound.com")
	if err != nil {
		t.Fatalf("get by non-existent domain: %v", err)
	}
	if notFound != nil {
		t.Fatalf("expected nil for non-existent domain, got %+v", notFound)
	}

	differentTenant, err := dao.GetByTenantAndDomain(ctx, 2, "example.com")
	if err != nil {
		t.Fatalf("get by different tenant: %v", err)
	}
	if differentTenant != nil {
		t.Fatalf("expected nil for different tenant, got %+v", differentTenant)
	}
}

func TestDomainDao_GetPageListByCond(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:domaintest_pagelist?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.DomainEntity{}); err != nil {
		t.Fatalf("migrate domain: %v", err)
	}

	dao := &DomainDao{
		GenericDao: newGenericDaoFromDB(db, model.TableNameDomain, "DomainDao"),
	}
	seeds := []model.DomainEntity{
		{TenantID: 1, Domain: "alpha.com", IsVerified: 1},
		{TenantID: 1, Domain: "beta.com", IsVerified: 0},
		{TenantID: 2, Domain: "gamma.com", IsVerified: 1},
	}
	ctx := context.Background()
	for i := range seeds {
		if err := dao.Insert(ctx, &seeds[i]); err != nil {
			t.Fatalf("seed domain %d: %v", i, err)
		}
	}

	cond := &DomainCond{
		BaseCond: &genericdao.BaseCond{Page: 1, PageSize: 10},
		TenantID: 1,
	}
	list, total, err := dao.GetPageListByCond(ctx, cond)
	if err != nil {
		t.Fatalf("page list: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2 for tenant 1, got %d", total)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 domains for tenant 1, got %d", len(list))
	}

	condFilter := &DomainCond{
		BaseCond: &genericdao.BaseCond{Page: 1, PageSize: 10},
		TenantID: 1,
		Domain:   "beta",
	}
	filteredList, filteredTotal, err := dao.GetPageListByCond(ctx, condFilter)
	if err != nil {
		t.Fatalf("filtered page list: %v", err)
	}
	if filteredTotal != 1 {
		t.Fatalf("expected total 1 for domain filter 'beta', got %d", filteredTotal)
	}
	if len(filteredList) != 1 || filteredList[0].Domain != "beta.com" {
		t.Fatalf("expected beta.com, got %+v", filteredList)
	}
}

func TestDomainDao_Delete(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open("file:domaintest_delete?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.DomainEntity{}); err != nil {
		t.Fatalf("migrate domain: %v", err)
	}

	dao := &DomainDao{
		GenericDao: newGenericDaoFromDB(db, model.TableNameDomain, "DomainDao"),
	}
	entity := &model.DomainEntity{
		TenantID:   1,
		Domain:     "example.com",
		IsVerified: 0,
	}
	if err := dao.Insert(ctx, entity); err != nil {
		t.Fatalf("insert domain: %v", err)
	}

	if err := dao.Delete(ctx, entity.ID, 1); err != nil {
		t.Fatalf("delete domain: %v", err)
	}

	got, err := dao.GetByID(ctx, entity.ID)
	if err != nil {
		t.Fatalf("get deleted domain: %v", err)
	}
	if got.ID != 0 {
		t.Fatalf("expected zero ID after delete, got %+v", got)
	}
}

func TestDomainCond_BuildCondition(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open("file:domaintest_cond?mode=memory&cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.DomainEntity{}); err != nil {
		t.Fatalf("migrate domain: %v", err)
	}

	dao := &DomainDao{
		GenericDao: newGenericDaoFromDB(db, model.TableNameDomain, "DomainDao"),
	}
	seeds := []model.DomainEntity{
		{TenantID: 1, Domain: "alpha.example.com", IsVerified: 1},
		{TenantID: 1, Domain: "beta.test.com", IsVerified: 0},
		{TenantID: 2, Domain: "gamma.demo.com", IsVerified: 1},
	}
	for i := range seeds {
		if err := dao.Insert(ctx, &seeds[i]); err != nil {
			t.Fatalf("seed domain %d: %v", i, err)
		}
	}

	query := db.Model(&model.DomainEntity{}).Table(model.TableNameDomain)
	cond := &DomainCond{TenantID: 1, Domain: "alpha"}
	cond.BuildCondition(query, model.TableNameDomain)

	var list model.DomainEntityList
	if err := query.Find(&list).Error; err != nil {
		t.Fatalf("find domains: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(list))
	}
	if list[0].Domain != "alpha.example.com" {
		t.Fatalf("unexpected filtered row: %+v", list[0])
	}

	emptyQuery := db.Model(&model.DomainEntity{}).Table(model.TableNameDomain)
	emptyCond := &DomainCond{}
	emptyCond.BuildCondition(emptyQuery, model.TableNameDomain)

	var emptyList model.DomainEntityList
	if err := emptyQuery.Find(&emptyList).Error; err != nil {
		t.Fatalf("find all domains: %v", err)
	}
	if len(emptyList) != 3 {
		t.Fatalf("expected 3 all domains, got %d", len(emptyList))
	}

	now := time.Now()
	deletedEntity := &model.DomainEntity{
		TenantID:   1,
		Domain:     "deleted.com",
		IsVerified: 0,
		Model: gorm.Model{
			DeletedAt: gorm.DeletedAt{Time: now, Valid: true},
		},
	}
	if err := dao.Insert(ctx, deletedEntity); err != nil {
		t.Fatalf("insert deleted domain: %v", err)
	}

	deletedQuery := db.Unscoped().Model(&model.DomainEntity{}).Table(model.TableNameDomain)
	deletedCond := &DomainCond{}
	deletedCond.BuildCondition(deletedQuery, model.TableNameDomain)

	var deletedList model.DomainEntityList
	if err := deletedQuery.Find(&deletedList).Error; err != nil {
		t.Fatalf("find unscoped: %v", err)
	}
	if len(deletedList) != 4 {
		t.Fatalf("expected 4 with deleted, got %d", len(deletedList))
	}
}
