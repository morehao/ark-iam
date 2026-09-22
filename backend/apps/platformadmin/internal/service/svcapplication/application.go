package svcapplication

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/morehao/ark-iam/pkg/audit"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/tenant"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplication"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ApplicationSvc interface {
	Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error)
	Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error
	Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error
	Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error)
	PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error)
}

type applicationSvc struct{}

var _ ApplicationSvc = (*applicationSvc)(nil)

func NewApplicationSvc() ApplicationSvc {
	return &applicationSvc{}
}

// isValidAppStatus 校验应用状态取值：空值表示「本次不修改状态」，非空必须命中白名单常量。
// 校验归 service（AGENTS.md 硬规则 3）：DTO 绑定的是前端传来的原始字符串，非法值在此拦截。
func isValidAppStatus(status model.AppStatus) bool {
	switch status {
	case "", model.AppStatusEnable, model.AppStatusDisable:
		return true
	default:
		return false
	}
}

// isValidPersonCreateTenantPolicy 校验「个人可自助建租户」策略：空值＝未提供（创建按 disable，更新不修改）。
func isValidPersonCreateTenantPolicy(policy model.AppPersonCreateTenantPolicy) bool {
	switch policy {
	case "", model.AppPersonCreateTenantPolicyEnable, model.AppPersonCreateTenantPolicyDisable:
		return true
	default:
		return false
	}
}

// normalizePersonCreateTenantPolicy 把未提供的策略（空串）归一为 disable：
// 列默认值即 disable（NULL ≡ disable），原 *bool 语义下 nil/缺省同样等价于关闭。
func normalizePersonCreateTenantPolicy(policy model.AppPersonCreateTenantPolicy) model.AppPersonCreateTenantPolicy {
	if policy == "" {
		return model.AppPersonCreateTenantPolicyDisable
	}
	return policy
}

// isValidJoinByInvitePolicy 校验「允许邀请加入租户」策略：空值＝未提供（创建按 disable，更新不修改）。
func isValidJoinByInvitePolicy(policy model.AppJoinByInvitePolicy) bool {
	switch policy {
	case "", model.AppJoinByInvitePolicyEnable, model.AppJoinByInvitePolicyDisable:
		return true
	default:
		return false
	}
}

// normalizeJoinByInvitePolicy 把未提供的策略（空串）归一为 disable（NULL ≡ disable）。
func normalizeJoinByInvitePolicy(policy model.AppJoinByInvitePolicy) model.AppJoinByInvitePolicy {
	if policy == "" {
		return model.AppJoinByInvitePolicyDisable
	}
	return policy
}

// maxRoleTemplateItems 单个应用的角色模板条目上限：模板是"本应用对外提供的契约值"清单，
// 实际只有个位数；设上限只为拦住把整张下游策略表灌进来的误用。
const maxRoleTemplateItems = 64

// maxRoleTemplateItemNameLen 模板角色名称长度上限（对齐 role.name 列宽 varchar(128)）。
const maxRoleTemplateItemNameLen = 128

// buildRoleTemplate 校验并归一化应用角色模板（名称去空白），返回可直接落 JSON 列的具名切片。
// 返回 ok=false 表示入参非法：条目数超限、code 形状不符 model.RoleCodePattern、code 在模板内重复、
// 名称为空或超长、或声明了产品锚点编码（锚点由开通链路按常量写入，模板占用同码会改写锚点角色名称）。
// 序列化交由 GORM 的 serializer:json，此处不存在会被折叠成"非法"的系统错误。
func buildRoleTemplate(items []model.RoleTemplateItem) (model.RoleTemplateItemList, bool) {
	if len(items) == 0 {
		return model.RoleTemplateItemList{}, true
	}
	if len(items) > maxRoleTemplateItems {
		return nil, false
	}
	seen := make(map[model.RoleCode]struct{}, len(items))
	normalized := make(model.RoleTemplateItemList, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if !model.IsValidRoleCode(item.Code) || model.IsProductAnchorRoleCode(item.Code) {
			return nil, false
		}
		if name == "" || len(name) > maxRoleTemplateItemNameLen {
			return nil, false
		}
		if _, ok := seen[item.Code]; ok {
			return nil, false
		}
		seen[item.Code] = struct{}{}
		normalized = append(normalized, model.RoleTemplateItem{Code: item.Code, Name: name})
	}
	return normalized, true
}

func (svc *applicationSvc) Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error) {
	// 编码规则（model.AppCodePattern）：小写字母开头，仅含小写字母/数字/下划线。
	// 非法编码（如连字符）在此拦截，避免落库后再靠人工纠正。
	if !model.IsValidAppCode(req.Code) {
		glog.Errorf(ctx, "[svcapplication.Create] 非法应用编码, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCodeInvalidError)
	}
	roleTemplate, ok := buildRoleTemplate(req.RoleTemplate)
	if !ok {
		glog.Errorf(ctx, "[svcapplication.Create] 非法应用角色模板, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationRoleTemplateInvalidError)
	}
	// 自助建租户/邀请加入两个策略来自前端：未提供按列默认 disable 处理，非法值直接拒绝。
	if !isValidPersonCreateTenantPolicy(req.AllowPersonCreateTenant) {
		glog.Errorf(ctx, "[svcapplication.Create] 非法个人建租户策略, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCreateError)
	}
	if !isValidJoinByInvitePolicy(req.AllowJoinByInvite) {
		glog.Errorf(ctx, "[svcapplication.Create] 非法邀请加入策略, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCreateError)
	}
	entity := &model.ApplicationEntity{
		Code:                    req.Code,
		Name:                    req.Name,
		Description:             req.Description,
		LogoURL:                 req.LogoURL,
		HomepageURL:             req.HomepageURL,
		Source:                  model.AppSourceThirdParty, // 控制台创建的应用恒为第三方接入，builtin/first_party 仅由种子与运维产生
		AllowPersonCreateTenant: normalizePersonCreateTenantPolicy(req.AllowPersonCreateTenant),
		AllowJoinByInvite:       normalizeJoinByInvitePolicy(req.AllowJoinByInvite),
		RoleTemplate:            roleTemplate,
		Sort:                    req.Sort,
		CreatedBy:               gincontext.GetUserIDString(ctx),
	}
	if err := dao.NewApplicationDao().Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCreateError)
	}
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionApplicationCreate,
		TenantID:   "",
		Result:     model.AuditResultSuccess,
		TargetType: model.AuditTargetTypeApplication,
		TargetID:   entity.ID,
	})
	return &dtoapplication.ApplicationCreateResp{
		AppID: entity.ID,
		Code:  entity.Code,
	}, nil
}

// 内置应用的写入约束：字段权威矩阵里 application 的 immutable 字段是 source（内置标记）与
// code（编码），两者都不在 ApplicationUpdateReq 中（source 控制台无写入入口，code 由下方显式拒绝）。
// 名称/描述/启停/排序/logo/主页都归运维（create_only）：控制台改完不被 L1 回写。
// code（应用编码）自建应用可改，**内置应用拒改**：控制台菜单入口仍按该编码定位
// （svcpermission.MyTree 按 platform_admin 查应用、tenantadmin loadConsoleApps 只保留 tenant_admin），
// 改名会当场让对应控制台侧边栏失联且无法从界面恢复（真要换属版本级动作）。种子定位用 seed_key，与此无关。
func (svc *applicationSvc) Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error {
	if !isValidAppStatus(req.Status) {
		glog.Errorf(ctx, "[svcapplication.Update] 非法应用状态, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	if !isValidPersonCreateTenantPolicy(req.AllowPersonCreateTenant) {
		glog.Errorf(ctx, "[svcapplication.Update] 非法个人建租户策略, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	if !isValidJoinByInvitePolicy(req.AllowJoinByInvite) {
		glog.Errorf(ctx, "[svcapplication.Update] 非法邀请加入策略, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	entity, err := dao.NewApplicationDao().GetByID(ctx, req.AppID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	if entity == nil || entity.ID == "" {
		return code.GetError(code.ApplicationNotExistError)
	}
	// 走 dao.UpdateFields（结构化 Updates）而非 UpdateMap：role_template 是 serializer:json 列，
	// 用 map 写会绕过 serializer 静默落脏值（设计文档 D6）。只填要改的字段，列名与 model 的 column tag 一致。
	updateEntity := &model.ApplicationEntity{
		Name:        req.Name,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		HomepageURL: req.HomepageURL,
		Sort:        req.Sort,
		UpdatedBy:   gincontext.GetUserIDString(ctx),
	}
	fields := []string{"name", "description", "logo_url", "homepage_url", "sort", "updated_by"}
	// status 留空表示不修改：不写该列，避免把状态覆盖为空串
	if req.Status != "" {
		updateEntity.Status = req.Status
		fields = append(fields, "status")
	}
	// code 留空表示不修改；非空且确有变化时：内置应用拒改（见函数头注释），其余按创建时的规则校验
	// （model.AppCodePattern）。应用内唯一由唯一索引兜底，撞重返回本领域更新错误码。
	if req.Code != "" && req.Code != entity.Code {
		if entity.Source.IsBuiltin() {
			glog.Errorf(ctx, "[svcapplication.Update] 拒绝修改内置应用编码, appID:%s, req:%s",
				req.AppID, gutil.ToJsonString(req))
			return code.GetError(code.ApplicationBuiltInCodeImmutableError)
		}
		if !model.IsValidAppCode(req.Code) {
			glog.Errorf(ctx, "[svcapplication.Update] 非法应用编码, req:%s", gutil.ToJsonString(req))
			return code.GetError(code.ApplicationCodeInvalidError)
		}
		updateEntity.Code = req.Code
		fields = append(fields, "code")
	}
	// 两个策略留空表示不修改：非空值已通过白名单校验，直接按常量写入
	if req.AllowPersonCreateTenant != "" {
		updateEntity.AllowPersonCreateTenant = req.AllowPersonCreateTenant
		fields = append(fields, "allow_person_create_tenant")
	}
	if req.AllowJoinByInvite != "" {
		updateEntity.AllowJoinByInvite = req.AllowJoinByInvite
		fields = append(fields, "allow_join_by_invite")
	}
	// 角色模板：null 表示不修改（与本结构其余可选字段一致），[] 表示清空，传值即全量替换。
	// 模板是契约值的唯一来源，改动后必须同步到所有已订阅该应用的租户（新增/改名 → 物化角色，
	// 移除 → 撤下角色），故与本次更新同事务：模板落库与租户侧物化要么都生效、要么都不生效。
	templateChanged := req.RoleTemplate != nil
	if templateChanged {
		roleTemplate, ok := buildRoleTemplate(req.RoleTemplate)
		if !ok {
			glog.Errorf(ctx, "[svcapplication.Update] 非法应用角色模板, req:%s", gutil.ToJsonString(req))
			return code.GetError(code.ApplicationRoleTemplateInvalidError)
		}
		updateEntity.RoleTemplate = roleTemplate
		fields = append(fields, "role_template")
	}
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		// WithTx 定义在内嵌的 *gormdao.Dao 上，返回值本身就是 *gormdao.Dao，直接传给 UpdateFields
		if err := dao.UpdateFields(ctx, dao.NewApplicationDao().WithTx(tx), req.AppID, updateEntity, fields...); err != nil {
			return err
		}
		if !templateChanged {
			return nil
		}
		return tenant.SyncAppRoleTemplateToTenants(ctx, tx, &tenant.SyncAppRoleTemplateToTenantsReq{
			AppID:     req.AppID,
			CreatedBy: gincontext.GetUserIDString(ctx),
		})
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcapplication.Update] update app/role template fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	return nil
}

func (svc *applicationSvc) Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error {
	entity, err := dao.NewApplicationDao().GetByID(ctx, req.AppID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	if entity != nil && entity.Source.IsBuiltin() {
		return code.GetError(code.ApplicationBuiltInErr)
	}
	userID := gincontext.GetUserIDString(ctx)
	if err := dao.NewApplicationDao().Delete(ctx, req.AppID, userID); err != nil {
		glog.Errorf(ctx, "[svcapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	return nil
}

func (svc *applicationSvc) Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error) {
	entity, err := dao.NewApplicationDao().GetByID(ctx, req.AppID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetDetailError)
	}
	if entity == nil || entity.ID == "" {
		return nil, code.GetError(code.ApplicationNotExistError)
	}
	return &dtoapplication.ApplicationDetailResp{
		AppID:                   entity.ID,
		Code:                    entity.Code,
		Name:                    entity.Name,
		Description:             entity.Description,
		LogoURL:                 entity.LogoURL,
		HomepageURL:             entity.HomepageURL,
		Source:                  entity.Source,
		Status:                  entity.Status,
		Sort:                    entity.Sort,
		AllowPersonCreateTenant: entity.AllowPersonCreateTenant,
		AllowJoinByInvite:       entity.AllowJoinByInvite,
		RoleTemplate:            entity.RoleTemplate,
		CreatedAt:               entity.CreatedAt.Unix(),
	}, nil
}

func (svc *applicationSvc) PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error) {
	// 非法来源过滤值直接拒绝，避免 DAO 落成「查不到任何数据」的空结果而看不出原因
	if req.Source != "" {
		switch req.Source {
		case model.AppSourceBuiltin, model.AppSourceFirstParty, model.AppSourceThirdParty:
		default:
			glog.Errorf(ctx, "[svcapplication.PageList] 非法 source 过滤值, req:%s", gutil.ToJsonString(req))
			return nil, code.GetError(code.ApplicationGetPageListError)
		}
	}
	cond := &dao.ApplicationCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		Name:   req.Name,
		Source: req.Source,
		Status: req.Status,
	}
	list, total, err := dao.NewApplicationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetPageListError)
	}
	items := make([]dtoapplication.PageListItem, 0, len(list))
	for _, v := range list {
		items = append(items, dtoapplication.PageListItem{
			AppID:                   v.ID,
			Code:                    v.Code,
			Name:                    v.Name,
			Description:             v.Description,
			Source:                  v.Source,
			Status:                  v.Status,
			Sort:                    v.Sort,
			AllowPersonCreateTenant: v.AllowPersonCreateTenant,
			AllowJoinByInvite:       v.AllowJoinByInvite,
			// 列表带上角色模板：编辑弹窗以列表行为初值，缺了它会把"未修改"误判成"清空"
			RoleTemplate: v.RoleTemplate,
			CreatedAt:    v.CreatedAt.Unix(),
			UpdatedAt:    v.UpdatedAt.Unix(),
		})
	}
	return &dtoapplication.ApplicationPageListResp{List: items, Total: total}, nil
}
