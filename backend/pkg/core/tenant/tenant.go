// Package tenant 承载租户聚合根的跨表领域能力，
// 供 auth（自助开通租户）/ platformadmin（平台建租户）复用。
package tenant

import (
	"context"

	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/sso"
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
		RevokePersonSessions(ctx, personID)
	}
	return nil
}

// RevokePersonSessions 撤销某自然人的全部会话凭证（refresh token + SSO 会话），
// 并向其已登录的各应用（含第三方 RP）投递 back-channel logout 通知。
//
// 用于"改密/重置密码后必须立即全局登出"的场景（自助改密、管理员重置成员密码、
// 平台侧重置内置管理员密码、首次登录强制改密）：旧会话与 refresh token 必须在改密
// 成功的同时失效，否则旧口令泄露的影响面不会随改密收敛。
//
// 幂等、失败不阻断：单个环节失败只记日志——密码哈希本身已经更新，调用方主流程不应因此失败。
func RevokePersonSessions(ctx context.Context, personID string) {
	if personID == "" {
		return
	}
	if err := dao.NewRefreshTokenDao().RevokeByPersonID(ctx, personID); err != nil {
		glog.Errorf(ctx, "[tenant.RevokePersonSessions] revoke refresh token fail, personID:%s, err:%v", personID, err)
	}
	// 顺序要求：先入队 back-channel 通知，再撤销 SSO 会话——登记查询依赖 sso_user_sessions 索引，
	// 会话撤销后索引即清除，通知将无法投递（与 svcauth.Logout 的处置一致）。
	if count, bErr := sso.EnqueueLogoutsByPersonID(ctx, personID); bErr != nil {
		glog.Warnf(ctx, "[tenant.RevokePersonSessions] enqueue back-channel logout fail, personID:%s, err:%v", personID, bErr)
	} else if count > 0 {
		glog.Infof(ctx, "[tenant.RevokePersonSessions] back-channel logout enqueued, personID:%s, count:%d", personID, count)
	}
	if err := sso.RevokeSSOSessionsByPersonID(ctx, personID); err != nil {
		glog.Errorf(ctx, "[tenant.RevokePersonSessions] revoke sso session fail, personID:%s, err:%v", personID, err)
	}
}

// CreateWithRootDeptReq 构造 CreateWithRootDept 入参。
type CreateWithRootDeptReq struct {
	Code      string // 租户编码（可空，由调用方生成后传入）
	Name      string // 租户名
	Type      model.TenantType
	DbUser    string
	Status    model.TenantStatus // 租户状态（可空，空值按 active 处理）
	Tag       string
	CreatedBy string
}

// CreateWithRootDept 在 tx 事务内创建租户 + 同名根部门节点（部门树容器根）。
// 必须在调用方的事务 tx 内执行（空 tx 会 panic）。返回新建租户实体与根部门实体（均含 ID）：
// 根部门需要被调用方用作首位成员（内置管理员 / 自助建租户 owner）的行政主部门。
func CreateWithRootDept(ctx context.Context, tx *gorm.DB, req *CreateWithRootDeptReq) (*model.TenantEntity, *model.DepartmentEntity, error) {
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
		return nil, nil, err
	}
	// 每个租户创建时自动创建同名的根部门节点（部门树容器根）
	rootDept := &model.DepartmentEntity{
		TenantID:  tenantEntity.ID,
		ParentID:  "",
		Name:      req.Name,
		Status:    model.DeptNodeStatusEnable,
		CreatedBy: req.CreatedBy,
	}
	if err := dao.NewDepartmentDao().WithTx(tx).Insert(ctx, rootDept); err != nil {
		return nil, nil, err
	}
	// 根节点路径："/"+id，深度 1（ID 由 BeforeCreate 生成，需创建后补写）
	if err := dao.NewDepartmentDao().WithTx(tx).UpdateMap(ctx, rootDept.ID, map[string]any{
		"dept_path":  "/" + rootDept.ID,
		"dept_depth": 1,
	}); err != nil {
		return nil, nil, err
	}
	return tenantEntity, rootDept, nil
}
