package svctenant

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/audit"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/person"
	"github.com/morehao/ark-iam/pkg/core/tenant"
	"github.com/morehao/ark-iam/pkg/core/user"
	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objtenant"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

type TenantSvc interface {
	Create(ctx *gin.Context, req *dtotenant.TenantCreateReq) (*dtotenant.TenantCreateResp, error)
	ResetAdminPassword(ctx *gin.Context, req *dtotenant.TenantAdminResetPasswordReq) (*dtotenant.TenantAdminResetPasswordResp, error)
	Delete(ctx *gin.Context, req *dtotenant.TenantDeleteReq) error
	Update(ctx *gin.Context, req *dtotenant.TenantUpdateReq) error
	Detail(ctx *gin.Context, req *dtotenant.TenantDetailReq) (*dtotenant.TenantDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenant.TenantPageListReq) (*dtotenant.TenantPageListResp, error)
}

type tenantSvc struct {
}

var _ TenantSvc = (*tenantSvc)(nil)

func NewTenantSvc() TenantSvc {
	return &tenantSvc{}
}

// Create 创建租户管理。
// 租户编码由服务端按统一规则自动生成（见 pkg/core/tenant.GenerateCode），入参不接收编码；
// 编码创建后不可变更（Update 不修改 code）。
//
// 一个事务内完成：租户 + 同名根部门 + 内置管理员用户（source=builtin）+ 租户自服务权限开通
// （应用订阅 / 内置角色 / 菜单授权 / 角色绑定）。管理员初始密码为系统生成的临时密码，
// 仅在响应中返回一次，且该管理员首次登录必须改密（见
// docs/design/system-design.md §5.8）。
func (svc *tenantSvc) Create(ctx *gin.Context, req *dtotenant.TenantCreateReq) (*dtotenant.TenantCreateResp, error) {
	admin := req.Admin
	if admin == nil {
		return nil, code.GetError(code.UserContactRequiredError)
	}
	// 邮箱或手机号至少填写一个：既是自然人识别键，也是后续"密码找回/交接"的唯一联系信息
	if admin.PrimaryEmail == "" && admin.PrimaryPhone == "" {
		return nil, code.GetError(code.UserContactRequiredError)
	}

	tenantCode, err := tenant.GenerateCode()
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantCreate] generate tenant code fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantCreateError)
	}
	// 临时密码：每个租户管理员各不相同，规则统一走 pkg/credential（需求②）
	tempPassword, err := credential.GenerateTemporaryPassword()
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantCreate] generate temporary password fail, err:%v", err)
		return nil, code.GetError(code.TenantCreateError)
	}
	passwordHash, err := gcrypto.GeneratePasswordHash(tempPassword)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantCreate] GeneratePasswordHash fail, err:%v", err)
		return nil, code.GetError(code.PasswordHashError)
	}

	userID := gincontext.GetUserIDString(ctx)
	tenantType := req.Type
	if tenantType != model.TenantTypeCustomer && tenantType != model.TenantTypePlatform {
		tenantType = model.TenantTypeCustomer
	}

	var (
		tenantID         string
		adminUserID      string
		adminPersonIsNew bool
	)
	txErr := dbclient.IamDB(ctx).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result, cErr := tenant.CreateTenantWithBuiltinAdmin(ctx, tx, &tenant.CreateTenantWithBuiltinAdminReq{
			Tenant: &tenant.CreateWithRootDeptReq{
				Code:      tenantCode,
				CreatedBy: userID,
				DbUser:    req.DbUser,
				Name:      req.Name,
				Status:    model.NormalizeTenantStatus(req.Status),
				Tag:       req.Tag,
				Type:      tenantType,
			},
			AdminUser: &user.CreateReq{
				Person: &person.FindOrCreateReq{
					Username:          admin.Username,
					PrimaryEmail:      admin.PrimaryEmail,
					PrimaryPhone:      admin.PrimaryPhone,
					PasswordEncrypted: passwordHash,
					PasswordMethod:    model.PasswordMethodBcrypt,
					PasswordStatus:    model.PasswordStatusMustChange,
					Name:              admin.Name,
					CreatedBy:         userID,
				},
				Name:      admin.Name,
				CreatedBy: userID,
			},
		})
		if cErr != nil {
			return cErr
		}
		tenantID = result.Tenant.ID
		adminUserID = result.AdminUser.ID
		adminPersonIsNew = result.AdminPersonCreated
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svctenant.TenantCreate] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantCreateError)
	}
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionTenantCreate,
		TenantID:   tenantID,
		Result:     model.AuditResultSuccess,
		TargetType: model.AuditTargetTypeTenant,
		TargetID:   tenantID,
	})

	resp := &dtotenant.TenantCreateResp{
		TenantID:    tenantID,
		AdminUserID: adminUserID,
	}
	// 命中已有自然人时不回显初始密码：其密码未被改动（见 person.FindOrCreate 约定），
	// 此时若仍需交付凭据，由运营调用 ResetAdminPassword 重新生成。
	if adminPersonIsNew {
		// TODO(delivery): 临时密码目前只能在本响应中回显一次（系统尚无邮件/短信通道）；
		// 接入通道后改为下发给账号本人，本响应不再返回明文。
		// 见 docs/design/system-design.md §5.8。
		resp.AdminInitialPassword = tempPassword
	}
	return resp, nil
}

// ResetAdminPassword 重置租户内置管理员（source=builtin）的密码（兜底路径，D5）。
//
// 授权边界：只作用于该租户 source=builtin + user_type=member 的首位用户，
// 即建租户时由平台创建的管理员（自助建租户场景下为 owner）；**不触碰**租户手工创建的
// manual 成员——平台没有管理租户内部成员的正当场景（见 docs/design/system-design.md §5.8）。
// 命中不到（含只有 manual 成员）与"不允许"统一返回 UserNotExistError，不暴露租户成员结构。
//
// 生成新临时密码 → 置 password_status=must_change → 撤销该自然人既有会话 → 写审计；
// 明文只在响应中返回一次，不落库、不写日志。
func (svc *tenantSvc) ResetAdminPassword(ctx *gin.Context, req *dtotenant.TenantAdminResetPasswordReq) (*dtotenant.TenantAdminResetPasswordResp, error) {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.ResetAdminPassword] dao GetByID tenant fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantAdminResetPasswordError)
	}
	if tenantEntity == nil || tenantEntity.ID == "" {
		return nil, code.GetError(code.TenantNotExistError)
	}

	// 目标租户来自请求参数（可能不是调用方所在的平台租户）：显式声明「指定租户」作用域，
	// 否则 tenant_user 查询会被当前租户过滤成 0 行，"内置管理员不存在"变成静默误判。
	builtinAdmin, err := dao.NewUserDao().GetByCond(dbclient.ExplicitTenantContext(ctx, tenantEntity.ID), &dao.UserCond{
		TenantID: tenantEntity.ID,
		Source:   model.UserSourceBuiltin,
		UserType: model.UserTypeMember,
	})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.ResetAdminPassword] dao GetByCond builtin admin fail, err:%v, tenantID:%s", err, tenantEntity.ID)
		return nil, code.GetError(code.TenantAdminResetPasswordError)
	}
	if builtinAdmin == nil || builtinAdmin.ID == "" || builtinAdmin.PersonID == "" {
		return nil, code.GetError(code.UserNotExistError)
	}

	tempPassword, err := credential.GenerateTemporaryPassword()
	if err != nil {
		glog.Errorf(ctx, "[svctenant.ResetAdminPassword] generate temporary password fail, err:%v", err)
		return nil, code.GetError(code.TenantAdminResetPasswordError)
	}
	passwordHash, err := gcrypto.GeneratePasswordHash(tempPassword)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.ResetAdminPassword] GeneratePasswordHash fail, err:%v", err)
		return nil, code.GetError(code.PasswordHashError)
	}
	operatorID := gincontext.GetUserIDString(ctx)
	if err := dao.NewPersonDao().UpdateMap(ctx, builtinAdmin.PersonID, map[string]any{
		"password_encrypted": passwordHash,
		"password_method":    model.PasswordMethodBcrypt,
		"password_status":    model.PasswordStatusMustChange,
		"updated_by":         operatorID,
	}); err != nil {
		glog.Errorf(ctx, "[svctenant.ResetAdminPassword] person UpdateMap fail, err:%v, personID:%s", err, builtinAdmin.PersonID)
		return nil, code.GetError(code.TenantAdminResetPasswordError)
	}

	// 改密即全局登出：旧会话/refresh token 立即失效，新口令首次登录必须改密
	tenant.RevokePersonSessions(ctx, builtinAdmin.PersonID)

	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionTenantAdminPasswordReset,
		TenantID:   tenantEntity.ID,
		Result:     model.AuditResultSuccess,
		TargetType: model.AuditTargetTypeUser,
		TargetID:   builtinAdmin.ID,
	})
	// TODO(delivery): 临时密码目前只能在本响应中回显一次（系统尚无邮件/短信通道）；
	// 接入通道后改为下发给账号本人，本响应不再返回明文。
	// 见 docs/design/system-design.md §5.8。
	return &dtotenant.TenantAdminResetPasswordResp{
		UserID:          builtinAdmin.ID,
		InitialPassword: tempPassword,
	}, nil
}

// Delete 删除租户。
//
// 平台自运营租户（种子租户 t_platform）禁删：它是平台控制台自身所在租户，删除即整栈失联，
// 且产品内没有恢复路径（只能改库）。判定依据与 Update 的「不可挂起」同源——平台租户编码是
// 种子身份（model.SeedPlatformTenantCode），与租户改名无关。
func (svc *tenantSvc) Delete(ctx *gin.Context, req *dtotenant.TenantDeleteReq) error {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantDelete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantDeleteError)
	}
	if tenantEntity == nil || tenantEntity.ID == "" {
		return code.GetError(code.TenantNotExistError)
	}
	if tenantEntity.Code == model.SeedPlatformTenantCode {
		glog.Errorf(ctx, "[svctenant.TenantDelete] refuse to delete platform tenant, tenantID:%s, req:%s", req.TenantID, gutil.ToJsonString(req))
		return code.GetError(code.TenantBuiltInDeleteForbiddenError)
	}

	userID := gincontext.GetUserIDString(ctx)

	if err := dao.NewTenantDao().Delete(ctx, req.TenantID, userID); err != nil {
		glog.Errorf(ctx, "[svctenant.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantDeleteError)
	}
	return nil
}

// Update 更新租户管理
func (svc *tenantSvc) Update(ctx *gin.Context, req *dtotenant.TenantUpdateReq) error {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantUpdate] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantUpdateError)
	}
	if tenantEntity == nil || tenantEntity.ID == "" {
		return code.GetError(code.TenantNotExistError)
	}

	userID := gincontext.GetUserIDString(ctx)
	tenantType := req.Type
	if tenantType != model.TenantTypeCustomer && tenantType != model.TenantTypePlatform {
		tenantType = model.TenantTypeCustomer
	}
	tenantStatus := model.NormalizeTenantStatus(req.Status)
	// 禁止挂起操作者自己所在的租户：挂起后该租户整体无法登录、本控制台随之失联，
	// 且产品内没有恢复路径（只能改库），属于不可逆自锁。
	if tenantStatus == model.TenantStatusSuspended && req.TenantID == gincontext.GetTenantIDString(ctx) {
		glog.Errorf(ctx, "[svctenant.TenantUpdate] refuse to suspend own tenant, tenantID:%s, req:%s", req.TenantID, gutil.ToJsonString(req))
		return code.GetError(code.TenantSuspendSelfForbiddenError)
	}
	// 平台自运营租户不可挂起：它是平台控制台自身所在租户，挂起会导致整栈失联且无恢复路径。
	// 与种子的 status=reconcile 不变式同源（字段权威矩阵），此处拒写以消除双写者。
	if tenantStatus == model.TenantStatusSuspended && tenantEntity.Code == model.SeedPlatformTenantCode {
		glog.Errorf(ctx, "[svctenant.TenantUpdate] refuse to suspend platform tenant, tenantID:%s, req:%s", req.TenantID, gutil.ToJsonString(req))
		return code.GetError(code.TenantPlatformSuspendForbiddenError)
	}
	updateMap := map[string]any{
		"db_user":    req.DbUser,
		"name":       req.Name,
		"status":     tenantStatus,
		"tag":        req.Tag,
		"type":       tenantType,
		"updated_by": userID,
	}
	if err := dao.NewTenantDao().UpdateMap(ctx, req.TenantID, updateMap); err != nil {
		glog.Errorf(ctx, "[svctenant.TenantUpdate] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantUpdateError)
	}
	// 由非挂起转为挂起：立即撤销该租户全部成员的 refresh token 与 SSO 会话，
	// 切断既有登录态（access token 依赖其短 TTL 自然过期）。撤销失败仅告警，
	// 不阻断挂起本身——租户状态已在库中生效，登录/签发令牌两个门禁会独立拦截。
	if tenantStatus == model.TenantStatusSuspended && tenantEntity.Status != model.TenantStatusSuspended {
		if rErr := tenant.RevokeMemberSessions(ctx, req.TenantID); rErr != nil {
			glog.Errorf(ctx, "[svctenant.TenantUpdate] revoke member sessions fail, tenantID:%s, err:%v", req.TenantID, rErr)
		}
	}
	return nil
}

// Detail 根据id获取租户管理
func (svc *tenantSvc) Detail(ctx *gin.Context, req *dtotenant.TenantDetailReq) (*dtotenant.TenantDetailResp, error) {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantDetail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantGetDetailError)
	}
	if tenantEntity == nil || tenantEntity.ID == "" {
		return nil, code.GetError(code.TenantNotExistError)
	}
	resp := &dtotenant.TenantDetailResp{
		TenantID: tenantEntity.ID,
		TenantBaseInfo: objtenant.TenantBaseInfo{
			Code:   tenantEntity.Code,
			DbUser: tenantEntity.DbUser,
			Name:   tenantEntity.Name,
			Status: tenantEntity.Status,
			Tag:    tenantEntity.Tag,
			Type:   tenantEntity.Type,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: tenantEntity.CreatedAt.Unix(),
			UpdatedAt: tenantEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

// PageList 分页获取租户管理列表。
// 列表需同时回传创建时间与更新时间（前端两列都展示），故两个时间字段均需赋值。
// 状态筛选走白名单校验：非法值直接报错，避免静默返回"看起来正常"的错误集合。
func (svc *tenantSvc) PageList(ctx *gin.Context, req *dtotenant.TenantPageListReq) (*dtotenant.TenantPageListResp, error) {
	switch req.Status {
	case "", model.TenantStatusActive, model.TenantStatusSuspended:
	default:
		glog.Errorf(ctx, "[svctenant.TenantPageList] invalid status filter, status:%s, req:%s", req.Status, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantPageListStatusInvalidError)
	}
	cond := &dao.TenantCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		Keyword: strings.TrimSpace(req.Name),
		Status:  req.Status,
	}
	tenantEntityList, total, err := dao.NewTenantDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantPageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantGetPageListError)
	}
	list := make([]dtotenant.TenantPageListItem, 0, len(tenantEntityList))
	for _, v := range tenantEntityList {
		list = append(list, dtotenant.TenantPageListItem{
			TenantID: v.ID,
			TenantBaseInfo: objtenant.TenantBaseInfo{
				Code:   v.Code,
				DbUser: v.DbUser,
				Name:   v.Name,
				Status: v.Status,
				Tag:    v.Tag,
				Type:   v.Type,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				CreatedAt: v.CreatedAt.Unix(),
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtotenant.TenantPageListResp{
		List:  list,
		Total: total,
	}, nil
}
