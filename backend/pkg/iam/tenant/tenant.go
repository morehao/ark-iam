// Package tenant 承载租户聚合根的跨表领域能力，
// 供 auth（自助开通租户）/ platformadmin（平台建租户）复用。
package tenant

import (
	"context"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"gorm.io/gorm"
)

// CreateWithRootOrgReq 构造 CreateWithRootOrg 入参。
type CreateWithRootOrgReq struct {
	Code      string // 租户编码（可空，由调用方生成后传入）
	Name      string // 租户名
	Type      model.TenantType
	DbUser    string
	IsSuspended bool
	Tag       string
	CreatedBy string
}

// CreateWithRootOrg 在 tx 事务内创建租户 + 同名根组织节点（组织树容器根）。
// 必须在调用方的事务 tx 内执行（空 tx 会 panic）。返回新建租户实体（含 ID）。
func CreateWithRootOrg(ctx context.Context, tx *gorm.DB, req *CreateWithRootOrgReq) (*model.TenantEntity, error) {
	tenantEntity := &model.TenantEntity{
		Code:        req.Code,
		Name:        req.Name,
		Type:        req.Type,
		DbUser:      req.DbUser,
		IsSuspended: req.IsSuspended,
		Tag:         req.Tag,
		CreatedBy:   req.CreatedBy,
	}
	if err := dao.NewTenantDao().WithTx(tx).Insert(ctx, tenantEntity); err != nil {
		return nil, err
	}
	// 每个租户创建时自动创建同名的根组织节点（组织树容器根）
	rootOrg := &model.OrganizationEntity{
		TenantID:  tenantEntity.ID,
		ParentID:  "",
		Name:      req.Name,
		Status:    string(model.OrgNodeStatusActive),
		CreatedBy: req.CreatedBy,
	}
	if err := dao.NewOrganizationDao().WithTx(tx).Insert(ctx, rootOrg); err != nil {
		return nil, err
	}
	// 根节点路径："/"+id，深度 1（ID 由 BeforeCreate 生成，需创建后补写）
	if err := dao.NewOrganizationDao().WithTx(tx).UpdateMap(ctx, rootOrg.ID, map[string]any{
		"org_path":  "/" + rootOrg.ID,
		"org_depth": 1,
	}); err != nil {
		return nil, err
	}
	return tenantEntity, nil
}
