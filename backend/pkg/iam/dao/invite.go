package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type InviteCond struct {
	*gormdao.BaseCond
	TenantID string
	Code     string
	Status   string
	IDs      []string
}

func (c *InviteCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.Code != "" {
		db.Where(tableName+".code = ?", c.Code)
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
	if len(c.IDs) > 0 {
		db.Where(tableName+".id IN ?", c.IDs)
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
