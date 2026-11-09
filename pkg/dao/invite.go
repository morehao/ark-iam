package dao

import (
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InviteCond struct {
	*gormdao.BaseCond
	TenantID string
	Code     string
	Status   model.InviteStatus
	IDs      []string
}

func (c *InviteCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	col := condCols(tableName)
	if c.TenantID != "" {
		db.Where(clause.Eq{Column: col("tenant_id"), Value: c.TenantID})
	}
	if c.Code != "" {
		db.Where(clause.Eq{Column: col("code"), Value: c.Code})
	}
	if c.Status != "" {
		db.Where(clause.Eq{Column: col("status"), Value: c.Status})
	}
	if len(c.IDs) > 0 {
		db.Where(clause.IN{Column: col("id"), Values: toAnySlice(c.IDs)})
	}
}

type InviteDao struct {
	*gormdao.Dao[model.InviteEntity, model.InviteEntityList, string]
}

func NewInviteDao(opts ...DaoOption) *InviteDao {
	return &InviteDao{
		Dao: gormdao.NewDao[model.InviteEntity, model.InviteEntityList, string](
			model.TableNameInvite, "InviteDao",
			resolveDBGetter(opts...),
		),
	}
}
