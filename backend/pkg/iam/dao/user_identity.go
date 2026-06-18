package dao

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type UserIdentityCond struct {
	*genericdao.BaseCond
	PersonID        uint
	ConnectorID     uint
	Provider        string
	Issuer          string
	ExternalSubject string
}

func (c *UserIdentityCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.PersonID != 0 {
		*db = *db.Where(tableName+".person_id = ?", c.PersonID)
	}
	if c.ConnectorID != 0 {
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
	*genericdao.GenericDao[model.UserIdentityEntity, model.UserIdentityEntityList]
}

func NewUserIdentityDao() *UserIdentityDao {
	return &UserIdentityDao{
		GenericDao: genericdao.NewGenericDao[model.UserIdentityEntity, model.UserIdentityEntityList](
			model.TableNameUserIdentity, "UserIdentityDao",
			dbclient.IamDB,
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

func (dao *UserIdentityDao) UpdateBinding(ctx context.Context, identityID, personID uint, issuer string, detail []byte) error {
	return dao.UpdateMap(ctx, identityID, map[string]any{
		"person_id":  personID,
		"issuer":     issuer,
		"detail":     json.RawMessage(detail),
		"updated_by": personID,
	})
}
