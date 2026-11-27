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

type SessionAuditCond struct {
	*gormdao.BaseCond
	PersonID  string
	SessionID string
	Issuer    string
}

func (c *SessionAuditCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.PersonID != "" {
		db.Where(clause.Eq{Column: col("person_id"), Value: c.PersonID})
	}
	if c.SessionID != "" {
		db.Where(clause.Eq{Column: col("session_id"), Value: c.SessionID})
	}
	if c.Issuer != "" {
		db.Where(clause.Eq{Column: col("issuer"), Value: c.Issuer})
	}
}

type SessionAuditDao struct {
	*gormdao.Dao[model.SessionAuditEntity, model.SessionAuditEntityList, string]
}

func NewSessionAuditDao(opts ...DaoOption) *SessionAuditDao {
	return &SessionAuditDao{
		Dao: gormdao.NewDao[model.SessionAuditEntity, model.SessionAuditEntityList, string](
			model.TableNameSession, "SessionAuditDao",
			resolveDBGetter(opts...),
		),
	}
}

// GetByIssuerAndSessionID 按 (issuer, session_id) 定位单条会话审计行（P0 多 issuer 会话隔离口径）。
// 用 CrossTenantContext：token/logout 入口可能无租户作用域，issuer+session_id 唯一即可定位。
func (dao *SessionAuditDao) GetByIssuerAndSessionID(ctx context.Context, issuer, sessionID string) (*model.SessionAuditEntity, error) {
	var entity model.SessionAuditEntity
	err := dao.DB(dbclient.CrossTenantContext(ctx)).Model(&model.SessionAuditEntity{}).
		Where("issuer = ? AND session_id = ?", issuer, sessionID).
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}
