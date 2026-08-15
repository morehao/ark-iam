package dao

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type UserIdentityCond struct {
	*gormdao.BaseCond
	PersonID        string
	ConnectorID     string
	Provider        string
	Issuer          string
	ExternalSubject string
}

func (c *UserIdentityCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.PersonID != "" {
		*db = *db.Where(tableName+".person_id = ?", c.PersonID)
	}
	if c.ConnectorID != "" {
		*db = *db.Where(tableName+".connector_id = ?", c.ConnectorID)
	}
	if c.Provider != "" {
		*db = *db.Where(tableName+".provider = ?", c.Provider)
	}
	if c.Issuer != "" {
		*db = *db.Where(tableName+".issuer = ?", c.Issuer)
	}
	if c.ExternalSubject != "" {
		*db = *db.Where(tableName+".external_subject = ?", c.ExternalSubject)
	}
}

type UserIdentityDao struct {
	*gormdao.Dao[model.UserIdentityEntity, model.UserIdentityEntityList, string]
}

func NewUserIdentityDao(opts ...DaoOption) *UserIdentityDao {
	return &UserIdentityDao{
		Dao: gormdao.NewDao[model.UserIdentityEntity, model.UserIdentityEntityList, string](
			model.TableNameUserIdentity, "UserIdentityDao",
			resolveDBGetter(opts...),
		),
	}
}

func (dao *UserIdentityDao) GetByIssuerAndExternalSubject(ctx context.Context, issuer, externalSubject string) (*model.UserIdentityEntity, error) {
	db := dbclient.IamDB(ctx)
	var entity model.UserIdentityEntity
	err := db.
		Where("issuer = ? AND external_subject = ? AND deleted_at IS NULL", issuer, externalSubject).
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (dao *UserIdentityDao) UpdateBinding(ctx context.Context, identityID, personID string, issuer string, detail []byte) error {
	return dao.UpdateMap(ctx, identityID, map[string]any{
		"person_id":  personID,
		"issuer":     issuer,
		"detail":     json.RawMessage(detail),
		"updated_by": personID,
	})
}
