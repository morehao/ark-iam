package dao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newUpdateFieldsTestDB 打开独立内存 sqlite（不挂租户插件），并把 ApplicationDao 指向它：
// 本文件验证的是 UpdateFields 自身的契约，不需要租户作用域，故沿用 audit_log_test.go 的注入方式。
func newUpdateFieldsTestDB(t *testing.T) (*gorm.DB, *ApplicationDao) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", sanitizeAuditTestName(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&model.ApplicationEntity{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db, NewApplicationDao(WithDBGetter(func(ctx context.Context) *gorm.DB {
		return db.WithContext(ctx)
	}))
}

func seedUpdateFieldsApplication(t *testing.T, applicationDao *ApplicationDao, entity *model.ApplicationEntity) {
	t.Helper()
	if entity.RoleTemplate == nil {
		entity.RoleTemplate = model.RoleTemplateItemList{}
	}
	if err := applicationDao.Insert(context.Background(), entity); err != nil {
		t.Fatalf("insert application: %v", err)
	}
	if entity.ID == "" {
		t.Fatal("expected non-zero id after insert")
	}
}

func reloadUpdateFieldsApplication(t *testing.T, db *gorm.DB, id string) *model.ApplicationEntity {
	t.Helper()
	var got model.ApplicationEntity
	if err := db.Where("id = ?", id).First(&got).Error; err != nil {
		t.Fatalf("reload application: %v", err)
	}
	return &got
}

// TestUpdateFieldsRejectsEmptyFieldList 钉住「调用方必须显式列出字段」的契约：
// 空字段列表在发出 SQL 之前就被拒绝，绝不能退化成「更新实体上全部非零字段」。
func TestUpdateFieldsRejectsEmptyFieldList(t *testing.T) {
	db, applicationDao := newUpdateFieldsTestDB(t)
	entity := &model.ApplicationEntity{
		Code:   "app_update_guard",
		Name:   "原名称",
		Source: model.AppSourceThirdParty,
		Status: model.AppStatusEnable,
	}
	seedUpdateFieldsApplication(t, applicationDao, entity)

	err := UpdateFields(context.Background(), applicationDao.Dao, entity.ID,
		&model.ApplicationEntity{Name: "不应落库的名称"})
	if !errors.Is(err, ErrUpdateFieldsEmpty) {
		t.Fatalf("空字段列表应返回 ErrUpdateFieldsEmpty，got %v", err)
	}

	if got := reloadUpdateFieldsApplication(t, db, entity.ID); got.Name != "原名称" {
		t.Fatalf("守卫命中后不得写入任何列，name = %q", got.Name)
	}
}

// TestUpdateFieldsWritesOnlyListedColumns 钉住 Select 语义：
// 同一实体内未列出的字段即使非零也不得被写回——这正是空列表守卫要保护的契约。
func TestUpdateFieldsWritesOnlyListedColumns(t *testing.T) {
	db, applicationDao := newUpdateFieldsTestDB(t)
	entity := &model.ApplicationEntity{
		Code:              "app_update_select",
		Name:              "原名称",
		Source:            model.AppSourceThirdParty,
		Status:            model.AppStatusEnable,
		AllowJoinByInvite: model.AppJoinByInvitePolicyDisable,
	}
	seedUpdateFieldsApplication(t, applicationDao, entity)

	err := UpdateFields(context.Background(), applicationDao.Dao, entity.ID, &model.ApplicationEntity{
		Name:              "新名称",
		Status:            model.AppStatusDisable,
		AllowJoinByInvite: model.AppJoinByInvitePolicyEnable,
	}, "name")
	if err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	got := reloadUpdateFieldsApplication(t, db, entity.ID)
	if got.Name != "新名称" {
		t.Fatalf("name 应被更新，got %q", got.Name)
	}
	if got.Status != model.AppStatusEnable {
		t.Fatalf("未列出的 status 不得被写回，got %q", got.Status)
	}
	if got.AllowJoinByInvite != model.AppJoinByInvitePolicyDisable {
		t.Fatalf("未列出的 allow_join_by_invite 不得被写回，got %q", got.AllowJoinByInvite)
	}
}

// TestUpdateFieldsSerializesJSONColumn 钉住 UpdateFields 是 JSON 列的结构化写路径：
// 原始列文本必须是合法 JSON（UpdateMap/Update("col", v) 会绕过 serializer 静默落 map[...] 脏值）。
func TestUpdateFieldsSerializesJSONColumn(t *testing.T) {
	db, applicationDao := newUpdateFieldsTestDB(t)
	entity := &model.ApplicationEntity{
		Code:   "app_update_json",
		Name:   "JSON 应用",
		Source: model.AppSourceThirdParty,
		Status: model.AppStatusEnable,
	}
	seedUpdateFieldsApplication(t, applicationDao, entity)

	roleTemplate := model.RoleTemplateItemList{{Code: "ops", Name: "运维"}}
	if err := UpdateFields(context.Background(), applicationDao.Dao, entity.ID,
		&model.ApplicationEntity{RoleTemplate: roleTemplate}, "role_template"); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	var raw string
	if err := db.Raw("SELECT role_template FROM application WHERE id = ?", entity.ID).Scan(&raw).Error; err != nil {
		t.Fatalf("read raw role_template: %v", err)
	}
	var decoded model.RoleTemplateItemList
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("role_template 原始列值不是合法 JSON: %q, err:%v", raw, err)
	}
	if len(decoded) != 1 || decoded[0].Code != "ops" || decoded[0].Name != "运维" {
		t.Fatalf("role_template 落库内容不符: %s", raw)
	}
}
