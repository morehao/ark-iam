package dao

import (
	"context"
	"time"

	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type RefreshTokenCond struct {
	*gormdao.BaseCond
	PersonID string
	TenantID string
	UserID   string
	Token    string
}

func (c *RefreshTokenCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.PersonID != "" {
		db.Where(tableName+".person_id = ?", c.PersonID)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.UserID != "" {
		db.Where(tableName+".user_id = ?", c.UserID)
	}
	if c.Token != "" {
		db.Where(tableName+".token = ?", c.Token)
	}
}

type RefreshTokenDao struct {
	*gormdao.Dao[model.RefreshTokenEntity, model.RefreshTokenEntityList, string]
}

func NewRefreshTokenDao(opts ...DaoOption) *RefreshTokenDao {
	return &RefreshTokenDao{
		Dao: gormdao.NewDao[model.RefreshTokenEntity, model.RefreshTokenEntityList, string](
			model.TableNameRefreshToken, "RefreshTokenDao",
			resolveDBGetter(opts...),
		),
	}
}

func (d *RefreshTokenDao) RevokeByPersonID(ctx context.Context, personID string) error {
	now := time.Now()
	return dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).
		Where("person_id = ?", personID).
		Where("revoked_at IS NULL").
		Update("revoked_at", &now).Error
}

// RevokeByCond 按条件批量撤销 refresh token（如"我的会话"列表的全量撤销）。
func (d *RefreshTokenDao) RevokeByCond(ctx context.Context, cond *RefreshTokenCond) error {
	now := time.Now()
	db := dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken)
	cond.BuildCondition(db, model.TableNameRefreshToken)
	return db.Where("revoked_at IS NULL").Update("revoked_at", &now).Error
}

// RevokeByID 按主键 + 归属条件撤销单条 refresh token，返回是否命中（用于归属校验）。
func (d *RefreshTokenDao) RevokeByID(ctx context.Context, id string, personID string, tenantID string) (bool, error) {
	now := time.Now()
	db := dbclient.IamDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
		Where("id = ?", id).Where("revoked_at IS NULL")
	if personID != "" {
		db = db.Where("person_id = ?", personID)
	}
	if tenantID != "" {
		db = db.Where("tenant_id = ?", tenantID)
	}
	res := db.Update("revoked_at", &now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
