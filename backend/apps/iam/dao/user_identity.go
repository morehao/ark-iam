package dao

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type UserIdentityCond struct {
	*genericdao.BaseCond
	TenantID        uint
	ConnectorID     uint
	ExternalSubject string
}

func (c *UserIdentityCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.ConnectorID != 0 {
		db.Where(tableName + ".connector_id = ?", c.ConnectorID)
	}
	if c.ExternalSubject != "" {
		db.Where(tableName + ".external_subject = ?", c.ExternalSubject)
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

func (dao *UserIdentityDao) GetByConnectorAndExternalSubject(ctx context.Context, tenantID, connectorID uint, externalSubject string) (*model.UserIdentityEntity, error) {
	db := dbclient.IamDB(ctx)
	var entity model.UserIdentityEntity
	err := db.
		Where("tenant_id = ? AND connector_id = ? AND external_subject = ? AND deleted_at IS NULL", tenantID, connectorID, externalSubject).
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (dao *UserIdentityDao) UpdateBinding(ctx context.Context, identityID, userID uint, issuer string, detail []byte) error {
	return dao.UpdateMap(ctx, identityID, map[string]any{
		"user_id":    userID,
		"issuer":     issuer,
		"detail":     json.RawMessage(detail),
		"updated_by": userID,
	})
}
