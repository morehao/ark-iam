package person

import (
	"context"
	"errors"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/sso"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UpdatePassword commits the password and refresh token revocation together.
func UpdatePassword(ctx context.Context, db gormdao.DBGetter, personID, hash string, status model.PasswordStatus, operatorID, expectedHash string) error {
	ctx = dbclient.CrossTenantContext(ctx)
	if db == nil {
		db = dbclient.IamDB
	}
	return db(ctx).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		person := tx.Model(&model.PersonEntity{}).Where("id = ? AND deleted_at IS NULL", personID)
		if expectedHash != "" {
			person = person.Where("password_encrypted = ?", expectedHash)
		}
		result := person.Updates(map[string]any{
			"password_encrypted": hash,
			"password_method":    model.PasswordMethodBcrypt,
			"password_status":    status,
			"updated_by":         operatorID,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("password update did not match a person")
		}
		// Wait for in-flight rotations before the revocation UPDATE takes its snapshot.
		var activeIDs []string
		if err := tx.Model(&model.RefreshTokenEntity{}).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("person_id = ? AND revoked_at IS NULL", personID).
			Order("id").Pluck("id", &activeIDs).Error; err != nil {
			return err
		}
		refreshDAO := dao.NewRefreshTokenDao(dao.WithDBGetter(func(context.Context) *gorm.DB { return tx }))
		if err := refreshDAO.RevokeByPersonID(ctx, personID); err != nil {
			return err
		}
		if dbclient.RedisCli == nil {
			return nil
		}
		if _, err := sso.EnqueueLogoutsByPersonID(ctx, personID); err != nil {
			return err
		}
		return sso.RevokeSSOSessionsByPersonID(ctx, personID)
	})
}
