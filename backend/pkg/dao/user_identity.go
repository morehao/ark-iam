package dao

import (
	"context"
	"errors"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserIdentityCond struct {
	*gormdao.BaseCond
	PersonID        string
	ConnectorID     string
	Provider        model.ConnectorProvider
	Issuer          string
	ExternalSubject string
}

func (c *UserIdentityCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.PersonID != "" {
		db.Where(clause.Eq{Column: col("person_id"), Value: c.PersonID})
	}
	if c.ConnectorID != "" {
		db.Where(clause.Eq{Column: col("connector_id"), Value: c.ConnectorID})
	}
	if c.Provider != "" {
		db.Where(clause.Eq{Column: col("provider"), Value: c.Provider})
	}
	if c.Issuer != "" {
		db.Where(clause.Eq{Column: col("issuer"), Value: c.Issuer})
	}
	if c.ExternalSubject != "" {
		db.Where(clause.Eq{Column: col("external_subject"), Value: c.ExternalSubject})
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

func (dao *UserIdentityDao) UpdateBinding(ctx context.Context, identityID, personID string, issuer string, detail model.UserIdentityDetail) error {
	entity := &model.UserIdentityEntity{
		PersonID:  personID,
		Issuer:    issuer,
		Detail:    detail,
		UpdatedBy: personID,
	}
	return UpdateFields(ctx, dao.Dao, identityID, entity, "person_id", "issuer", "detail", "updated_by")
}
