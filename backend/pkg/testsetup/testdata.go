package testsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/gcrypto"
)

type TestDataIDs struct {
	TenantIDs []uint
	PersonIDs []uint
	UserIDs   []uint
}

func PrepareTestTenant(ctx context.Context, name, tag string) (*model.TenantEntity, error) {
	db := dbclient.IamDB(ctx)
	entity := &model.TenantEntity{
		Name: name,
		Code: name,
		Tag:  tag,
	}
	if err := db.Create(entity).Error; err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	return entity, nil
}

func PrepareTestPerson(ctx context.Context, username, email, phone, password, name string) (*model.PersonEntity, error) {
	db := dbclient.IamDB(ctx)

	passwordHash, err := gcrypto.GeneratePasswordHash(password)
	if err != nil {
		return nil, fmt.Errorf("generate password hash: %w", err)
	}

	entity := &model.PersonEntity{
		Username:          model.StrPtr(username),
		PrimaryEmail:      model.StrPtr(email),
		PrimaryPhone:      model.StrPtr(phone),
		PasswordEncrypted: passwordHash,
		PasswordMethod:    "bcrypt",
		Name:              name,
		Profile:           json.RawMessage("{}"),
		CustomData:        json.RawMessage("{}"),
	}
	if err := db.Create(entity).Error; err != nil {
		return nil, fmt.Errorf("create person: %w", err)
	}
	return entity, nil
}

func PrepareTestUser(ctx context.Context, tenantID, personID uint, name string, isOwner int8) (*model.UserEntity, error) {
	db := dbclient.IamDB(ctx)
	now := time.Now()
	entity := &model.UserEntity{
		TenantID:   tenantID,
		PersonID:   personID,
		Name:       name,
		Profile:    json.RawMessage("{}"),
		CustomData: json.RawMessage("{}"),
		IsOwner:    isOwner,
		JoinedAt:   &now,
	}
	if err := db.Create(entity).Error; err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return entity, nil
}

func CleanupTestData(ctx context.Context, ids TestDataIDs) error {
	db := dbclient.IamDB(ctx)

	// 使用 model 表名常量，避免与当前 schema（user/person/tenant，无 iam_ 前缀）漂移。
	if len(ids.UserIDs) > 0 {
		for _, id := range ids.UserIDs {
			db.Exec("DELETE FROM "+model.TableNameUser+" WHERE id = ?", id)
		}
	}
	if len(ids.PersonIDs) > 0 {
		for _, id := range ids.PersonIDs {
			db.Exec("DELETE FROM "+model.TableNamePerson+" WHERE id = ?", id)
		}
	}
	if len(ids.TenantIDs) > 0 {
		for _, id := range ids.TenantIDs {
			db.Exec("DELETE FROM "+model.TableNameTenant+" WHERE id = ?", id)
		}
	}

	return nil
}

func UniqueName(prefix string) string {
	return fmt.Sprintf("test_%s_%d", prefix, time.Now().UnixNano())
}

func MustGeneratePasswordHash(password string) string {
	hash, err := gcrypto.GeneratePasswordHash(password)
	if err != nil {
		panic(fmt.Sprintf("GeneratePasswordHash failed: %v", err))
	}
	return hash
}

func PasswordMatches(hash, password string) bool {
	return gcrypto.ComparePasswordHash(hash, password) == nil
}

func BuildTestPersonUsername(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0] + "_test"
	}
	return email + "_test"
}