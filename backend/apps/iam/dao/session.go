package dao

import (
	"context"

	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type SessionCond struct {
	*genericdao.BaseCond
	PersonID uint
	TenantID uint
	UserID   uint
}

func (c *SessionCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.PersonID != 0 {
		db.Where(tableName + ".person_id = ?", c.PersonID)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.UserID != 0 {
		db.Where(tableName + ".user_id = ?", c.UserID)
	}
}

type SessionDao struct {
	*genericdao.GenericDao[model.RefreshTokenEntity, model.RefreshTokenEntityList]
}

func NewSessionDao() *SessionDao {
	return &SessionDao{
		GenericDao: genericdao.NewGenericDao[model.RefreshTokenEntity, model.RefreshTokenEntityList](
			model.TableNameRefreshToken, "SessionDao",
			dbclient.IamDB,
		),
	}
}

func (d *SessionDao) GetPageListByCond(ctx context.Context, cond *SessionCond, page, pageSize int) ([]model.RefreshTokenEntity, int64, error) {
	var total int64
	db := dbclient.IamDB(ctx)
	query := db.Model(&model.RefreshTokenEntity{})

	cond.BuildCondition(query, d.TableName)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list model.RefreshTokenEntityList
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (d *SessionDao) RevokeByID(ctx context.Context, id, personID, tenantID, userID uint) error {
	return dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).
		Where("id = ? AND person_id = ? AND tenant_id = ? AND user_id = ?", id, personID, tenantID, userID).
		Update("revoked_at", gorm.Expr("NOW()")).Error
}

func (d *SessionDao) RevokeAllByUserID(ctx context.Context, personID, tenantID, userID uint) error {
	return dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).
		Where("person_id = ? AND tenant_id = ? AND user_id = ? AND revoked_at IS NULL", personID, tenantID, userID).
		Update("revoked_at", gorm.Expr("NOW()")).Error
}