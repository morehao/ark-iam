package dao

import (
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"gorm.io/gorm"
)

type OrganizationCond struct {
	*gormdao.BaseCond
	TenantID string
	ParentID string
	// OrgPath 子树条件（含自身）：org_path = X OR org_path LIKE X||'/%
	OrgPath string
	Status  string
	Code    string
	Name    string
}

func (c *OrganizationCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != "" {
		db.Where(tableName+".tenant_id = ?", c.TenantID)
	}
	if c.ParentID != "" {
		db.Where(tableName+".parent_id = ?", c.ParentID)
	}
	if c.OrgPath != "" {
		db.Where(tableName+".org_path = ? OR "+tableName+".org_path LIKE ?", c.OrgPath, c.OrgPath+"/%")
	}
	if c.Status != "" {
		db.Where(tableName+".status = ?", c.Status)
	}
	if c.Code != "" {
		db.Where(tableName+".code = ?", c.Code)
	}
	if c.Name != "" {
		db.Where(tableName+".name = ?", c.Name)
	}
}

type OrganizationDao struct {
	*gormdao.Dao[model.OrganizationEntity, model.OrganizationEntityList, string]
}

func NewOrganizationDao(opts ...DaoOption) *OrganizationDao {
	return &OrganizationDao{
		Dao: gormdao.NewDao[model.OrganizationEntity, model.OrganizationEntityList, string](
			model.TableNameOrganization, "OrganizationDao",
			resolveDBGetter(opts...),
		),
	}
}
