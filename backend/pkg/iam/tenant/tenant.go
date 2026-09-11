// Package tenant 承载租户聚合根的跨表领域能力，
// 供 auth（自助开通租户）/ platformadmin（平台建租户）复用。
package tenant

import (
	"context"

	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

// RevokeMemberSessions 撤销某租户全部成员的会话凭证（refresh token + SSO 会话），
// 并向其已登录的各应用（含第三方 RP）投递 back-channel logout 通知，
// 用于租户被挂起时立即切断该租户成员的既有登录态（access token 依赖其短 TTL 自然过期）。
// 单个成员撤销失败仅记录日志并继续，不阻断调用方主流程：租户状态本身的准入拦截已独立生效。
func RevokeMemberSessions(ctx context.Context, tenantID string) error {
	if tenantID == "" {
		return nil
	}
	users, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{TenantID: tenantID})
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(users))
	for i := range users {
		personID := users[i].PersonID
		if personID == "" {
			continue
		}
		if _, ok := seen[personID]; ok {
			continue
		}
		seen[personID] = struct{}{}
		if rErr := dao.NewRefreshTokenDao().RevokeByPersonID(ctx, personID); rErr != nil {
			glog.Errorf(ctx, "[tenant.RevokeMemberSessions] revoke refresh token fail, personID:%s, err:%v", personID, rErr)
		}
		// 顺序要求：先入队 back-channel 通知，再撤销 SSO 会话——登记查询依赖 sso_user_sessions 索引，
		// 会话撤销后索引即清除，通知将无法投递（与 svcauth.Logout 的处置一致）。
		if count, bErr := sso.EnqueueLogoutsByPersonID(ctx, personID); bErr != nil {
			glog.Warnf(ctx, "[tenant.RevokeMemberSessions] enqueue back-channel logout fail, personID:%s, err:%v", personID, bErr)
		} else if count > 0 {
			glog.Infof(ctx, "[tenant.RevokeMemberSessions] back-channel logout enqueued, tenantID:%s, personID:%s, count:%d", tenantID, personID, count)
		}
		if sErr := sso.RevokeSSOSessionsByPersonID(ctx, personID); sErr != nil {
			glog.Errorf(ctx, "[tenant.RevokeMemberSessions] revoke sso session fail, personID:%s, err:%v", personID, sErr)
		}
	}
	return nil
}

// CreateWithRootOrgReq 构造 CreateWithRootOrg 入参。
type CreateWithRootOrgReq struct {
	Code      string // 租户编码（可空，由调用方生成后传入）
	Name      string // 租户名
	Type      model.TenantType
	DbUser    string
	Status    model.TenantStatus // 租户状态（可空，空值按 active 处理）
	Tag       string
	CreatedBy string
}

// CreateWithRootOrg 在 tx 事务内创建租户 + 同名根组织节点（组织树容器根）。
// 必须在调用方的事务 tx 内执行（空 tx 会 panic）。返回新建租户实体（含 ID）。
func CreateWithRootOrg(ctx context.Context, tx *gorm.DB, req *CreateWithRootOrgReq) (*model.TenantEntity, error) {
	tenantEntity := &model.TenantEntity{
		Code:      req.Code,
		Name:      req.Name,
		Type:      req.Type,
		DbUser:    req.DbUser,
		Status:    model.NormalizeTenantStatus(req.Status),
		Tag:       req.Tag,
		CreatedBy: req.CreatedBy,
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
