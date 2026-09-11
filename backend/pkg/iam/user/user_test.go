package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/person"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestDB 内存 SQLite 承载 person/tenant_user/organization_user 三张表。
// 这里直接用 tx（WithTx 路径）调用领域能力，无需注册全局 iam 库。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:iam_user_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.PersonEntity{},
		&model.UserEntity{},
		&model.OrganizationUserEntity{},
	))
	t.Cleanup(func() {
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func countRows(t *testing.T, db *gorm.DB, entity any, query string, args ...any) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(entity).Where(query, args...).Count(&count).Error)
	return count
}

func TestCreate_NewPersonWithOrgRelations(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	created, personCreated, err := Create(ctx, db, &CreateReq{
		TenantID: "t1",
		Person: &person.FindOrCreateReq{
			Username:           "acme-admin",
			PrimaryEmail:       "admin@acme.com",
			PasswordEncrypted:  "hash",
			PasswordMethod:     "bcrypt",
			MustChangePassword: true,
			Name:               "张三",
		},
		Name:          "张三",
		IsOwner:       true,
		CreatedBy:     "operator1",
		PrimaryOrgIDs: []string{"org-root"},
		SecondaryOrgIDs: []string{
			"org-2", "org-3",
		},
	})
	require.NoError(t, err)
	require.True(t, personCreated, "新建 person 时 personCreated 必须为 true（调用方据此回显临时密码）")
	require.NotEmpty(t, created.ID)

	// 来源/类型默认值：未显式指定 source 时落 manual
	require.Equal(t, model.UserSourceManual, created.Source)
	require.Equal(t, model.UserTypeMember, created.UserType)
	require.True(t, created.IsOwner)
	require.False(t, created.IsBuiltin())

	// person 的强制改密标记随新建落库
	storedPerson, err := dao.NewPersonDao().WithTx(db).GetByID(ctx, created.PersonID)
	require.NoError(t, err)
	require.NotNil(t, storedPerson)
	require.True(t, storedPerson.MustChangePassword)

	// 组织归属：1 primary + 2 secondary
	require.Equal(t, int64(1), countRows(t, db, &model.OrganizationUserEntity{},
		"tenant_id = ? AND user_id = ? AND relation_type = ?", "t1", created.ID, model.OrgUserRelationPrimary))
	require.Equal(t, int64(2), countRows(t, db, &model.OrganizationUserEntity{},
		"tenant_id = ? AND user_id = ? AND relation_type = ?", "t1", created.ID, model.OrgUserRelationSecondary))
}

func TestCreate_BuiltinSourcePersisted(t *testing.T) {
	db := newTestDB(t)
	created, personCreated, err := Create(context.Background(), db, &CreateReq{
		TenantID: "t1",
		Person: &person.FindOrCreateReq{
			Username: "acme-admin", PasswordEncrypted: "hash", PasswordMethod: "bcrypt", Name: "张三",
		},
		Source:        model.UserSourceBuiltin,
		Name:          "张三",
		PrimaryOrgIDs: []string{"org-root"},
	})
	require.NoError(t, err)
	require.True(t, personCreated)
	require.Equal(t, model.UserSourceBuiltin, created.Source)
	require.True(t, created.IsBuiltin())
}

func TestCreate_ExplicitPersonKeepsExistingPassword(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	existing := &model.PersonEntity{
		Username:          model.StrPtr("existing"),
		PrimaryEmail:      model.StrPtr("existing@example.com"),
		PasswordEncrypted: "existing-hash",
		PasswordMethod:    "bcrypt",
		Name:              "既有账号",
		Profile:           json.RawMessage(`{}`),
		CustomData:        json.RawMessage(`{}`),
	}
	require.NoError(t, db.WithContext(ctx).Create(existing).Error)

	_, personCreated, err := Create(ctx, db, &CreateReq{
		TenantID:      "t2",
		PersonID:      existing.ID,
		Name:          "既有账号",
		PrimaryOrgIDs: []string{"org-root"},
	})
	require.NoError(t, err)
	require.False(t, personCreated, "复用已有自然人时 personCreated 必须为 false（不得回显临时密码）")

	storedPerson, err := dao.NewPersonDao().WithTx(db).GetByID(ctx, existing.ID)
	require.NoError(t, err)
	require.NotNil(t, storedPerson)
	require.Equal(t, "existing-hash", storedPerson.PasswordEncrypted, "绝不覆盖既有自然人的密码")
	require.False(t, storedPerson.MustChangePassword, "复用既有自然人时不得置强制改密")
}

func TestCreate_DuplicatePersonInSameTenant(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	existing := &model.PersonEntity{
		Username: model.StrPtr("dup"), PasswordEncrypted: "hash", PasswordMethod: "bcrypt",
		Profile: json.RawMessage(`{}`), CustomData: json.RawMessage(`{}`),
	}
	require.NoError(t, db.WithContext(ctx).Create(existing).Error)

	req := &CreateReq{TenantID: "t1", PersonID: existing.ID, Name: "dup", PrimaryOrgIDs: []string{"org-root"}}
	_, _, err := Create(ctx, db, req)
	require.NoError(t, err)

	_, _, err = Create(ctx, db, req)
	require.ErrorIs(t, err, ErrAlreadyInTenant)
}

func TestCreate_MultiplePrimaryOrgRejected(t *testing.T) {
	db := newTestDB(t)
	_, _, err := Create(context.Background(), db, &CreateReq{
		TenantID: "t1",
		Person: &person.FindOrCreateReq{
			Username: "multi", PasswordEncrypted: "hash", PasswordMethod: "bcrypt", Name: "multi",
		},
		Name:          "multi",
		PrimaryOrgIDs: []string{"org-1", "org-2"},
	})
	require.ErrorIs(t, err, ErrMultiplePrimaryOrg)
	// 事务性：约束失败不得留下任何 user 行
	require.Equal(t, int64(0), countRows(t, db, &model.UserEntity{}, "tenant_id = ?", "t1"))
}

func TestCreate_LeaderConflict(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 已有用户占用了 org-1 的负责人
	occupier := &model.UserEntity{TenantID: "t1", PersonID: "p-occupier", Name: "occupier", Profile: json.RawMessage(`{}`), CustomData: json.RawMessage(`{}`)}
	require.NoError(t, db.WithContext(ctx).Create(occupier).Error)
	require.NoError(t, db.WithContext(ctx).Create(&model.OrganizationUserEntity{
		TenantID: "t1", OrganizationID: "org-1", UserID: occupier.ID, RelationType: model.OrgUserRelationLeader,
	}).Error)

	_, _, err := Create(ctx, db, &CreateReq{
		TenantID: "t1",
		Person: &person.FindOrCreateReq{
			Username: "newleader", PasswordEncrypted: "hash", PasswordMethod: "bcrypt", Name: "newleader",
		},
		Name:          "newleader",
		PrimaryOrgIDs: []string{"org-root"},
		LeaderOrgIDs:  []string{"org-1"},
	})
	require.ErrorIs(t, err, ErrOrgLeaderConflict)
}

func TestCreate_LeaderSameOrgAllowedForSingleLeader(t *testing.T) {
	db := newTestDB(t)
	created, _, err := Create(context.Background(), db, &CreateReq{
		TenantID: "t1",
		Person: &person.FindOrCreateReq{
			Username: "leader", PasswordEncrypted: "hash", PasswordMethod: "bcrypt", Name: "leader",
		},
		Name:          "leader",
		PrimaryOrgIDs: []string{"org-root"},
		LeaderOrgIDs:  []string{"org-1", "org-2"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), countRows(t, db, &model.OrganizationUserEntity{},
		"tenant_id = ? AND user_id = ? AND relation_type = ?", "t1", created.ID, model.OrgUserRelationLeader))
}

func TestCreate_NilTxRejected(t *testing.T) {
	_, _, err := Create(context.Background(), nil, &CreateReq{TenantID: "t1"})
	require.True(t, errors.Is(err, ErrNilTx))
}
