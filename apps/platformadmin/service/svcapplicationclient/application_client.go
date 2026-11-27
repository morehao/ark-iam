package svcapplicationclient

import (
	"context"
	"time"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtoapplicationclient"
	"github.com/morehao/ark-iam/pkg/audit"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/oidcctx"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ApplicationClientSvc interface {
	Create(ctx context.Context, req *dtoapplicationclient.ApplicationClientCreateReq) (*dtoapplicationclient.ApplicationClientCreateResp, error)
	Delete(ctx context.Context, req *dtoapplicationclient.ApplicationClientDeleteReq) error
	Update(ctx context.Context, req *dtoapplicationclient.ApplicationClientUpdateReq) error
	Detail(ctx context.Context, req *dtoapplicationclient.ApplicationClientDetailReq) (*dtoapplicationclient.ApplicationClientDetailResp, error)
	PageList(ctx context.Context, req *dtoapplicationclient.ApplicationClientPageListReq) (*dtoapplicationclient.ApplicationClientPageListResp, error)
	GetByClientID(ctx context.Context, clientID string) (*dtoapplicationclient.ApplicationClientDetailResp, error)
	ListSecrets(ctx context.Context, req *dtoapplicationclient.SecretListReq) (*dtoapplicationclient.SecretListResp, error)
	CreateSecret(ctx context.Context, req *dtoapplicationclient.SecretCreateReq) (*dtoapplicationclient.SecretCreateResp, error)
	DeleteSecret(ctx context.Context, req *dtoapplicationclient.SecretDeleteReq) error
}

type oAuthClientSvc struct {
	db gormdao.DBGetter
}

func applicationClientVisibleToTenant(entity *model.ApplicationClientEntity, tenantID string) bool {
	return entity != nil && entity.ID != "" && entity.TenantID == tenantID
}

var _ ApplicationClientSvc = (*oAuthClientSvc)(nil)

func NewApplicationClientSvc(db gormdao.DBGetter) ApplicationClientSvc {
	if db == nil {
		db = dbclient.IamDB
	}
	return &oAuthClientSvc{db: db}
}

// isValidApplicationClientStatus 校验客户端状态取值：空值表示「本次不修改状态」，非空必须命中白名单常量。
// 校验归 service（AGENTS.md 硬规则 3）：DTO 绑定的是前端传来的原始字符串，非法值在此拦截。
func isValidApplicationClientStatus(status model.ApplicationClientStatus) bool {
	switch status {
	case "", model.ApplicationClientStatusEnable, model.ApplicationClientStatusDisable:
		return true
	default:
		return false
	}
}

// isValidClientPKCEPolicy 校验 PKCE 强制策略：空串表示未提供（按 disable 处理）。
func isValidClientPKCEPolicy(policy model.ClientPKCEPolicy) bool {
	switch policy {
	case "", model.ClientPKCEPolicyEnable, model.ClientPKCEPolicyDisable:
		return true
	default:
		return false
	}
}

// normalizeClientPKCEPolicy 把未提供的策略（空串）归一为 disable：列默认值即 disable，
// 且为空串时原 bool 字段绑定出的 false 正是 disable，行为保持不变。
func normalizeClientPKCEPolicy(policy model.ClientPKCEPolicy) model.ClientPKCEPolicy {
	if policy == "" {
		return model.ClientPKCEPolicyDisable
	}
	return policy
}

// isValidClientAuthTimeClaimPolicy 校验 auth_time 声明策略：空串表示未提供（按 disable 处理）。
func isValidClientAuthTimeClaimPolicy(policy model.ClientAuthTimeClaimPolicy) bool {
	switch policy {
	case "", model.ClientAuthTimeClaimPolicyEnable, model.ClientAuthTimeClaimPolicyDisable:
		return true
	default:
		return false
	}
}

// normalizeClientAuthTimeClaimPolicy 把未提供的策略（空串）归一为 disable（NULL ≡ disable）。
func normalizeClientAuthTimeClaimPolicy(policy model.ClientAuthTimeClaimPolicy) model.ClientAuthTimeClaimPolicy {
	if policy == "" {
		return model.ClientAuthTimeClaimPolicyDisable
	}
	return policy
}

// isValidTokenEndpointAuthMethod 校验令牌端点认证方式：空串表示「创建时取列默认 / 更新时不修改」，
// 非空必须命中白名单常量。校验归 service（AGENTS.md 硬规则 3）：非法值不允许落库——
// 未知取值会在 oidcop 侧落到 fail-closed 的 private_key_jwt 分支，使客户端静默不可用。
func isValidTokenEndpointAuthMethod(method model.TokenEndpointAuthMethod) bool {
	switch method {
	case "",
		model.TokenEndpointAuthMethodBasic,
		model.TokenEndpointAuthMethodPost,
		model.TokenEndpointAuthMethodNone:
		return true
	default:
		return false
	}
}

// isConfidentialAuthMethod 判断认证方式是否属于机密客户端（可在令牌端点用密钥认证）。
// 公共客户端（none）无法保密凭据（RFC 6749 §2.1），故不得为其签发客户端密钥（RFC 6749 §10.1）。
func isConfidentialAuthMethod(method model.TokenEndpointAuthMethod) bool {
	switch method {
	case model.TokenEndpointAuthMethodBasic, model.TokenEndpointAuthMethodPost:
		return true
	default:
		return false
	}
}

// emptyIfNil 把请求里缺省的 nil 切片落成空切片（沿用本服务原有的「nil 落空数组」口径）。
// 不可把 nil 直接交给 serializer：GORM 对 NOT NULL 的 JSON 列会把 nil 序列化为空串，
// 在 PostgreSQL 的 json 列上是非法 JSON（更新路径必须显式 Select，nil 字段会被写进去）。
func emptyIfNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

// toResponseTypeList 把请求里的响应类型字符串转成列载具类型：元素是具名枚举，
// 切片之间不能直接转换，逐项转一次（nil 落空切片，与其余 JSON 列的落库口径一致）。
func toResponseTypeList(values []string) model.ResponseTypeList {
	list := make(model.ResponseTypeList, 0, len(values))
	for _, v := range values {
		list = append(list, model.ResponseType(v))
	}
	return list
}

// responseTypeStrings 把列的响应类型读成出参用的 []string。
func responseTypeStrings(list model.ResponseTypeList) []string {
	values := make([]string, 0, len(list))
	for _, v := range list {
		values = append(values, string(v))
	}
	return values
}

// resolveWriteTenantScope 解析创建 client 时的写入作用域 ctx 与期望归属租户 id（评审 H5）。
//
// application_client 是受租户插件注入的租户表，tenant_id 由插件按 ctx 的作用域落库。创建入口
// 可能发生在平台上下文（平台代租户建 client，作用域是平台/AllScope），此时 TenantIDFromContext
// 返回空串，直接落库会让租户侧永远查不到。故：
//   - 按 app_id 取应用（全局表，跨租户可读，无需作用域）；
//   - owner_type=tenant：用 dbclient.ExplicitTenantContext(ctx, app.OwnerTenantID) 作为写入 ctx，
//     使插件按其归属租户落 tenant_id=app.owner_tenant_id；返回期望归属租户 id 供不变式校验；
//   - owner_type=platform 或应用不存在：保持现行为（写入 ctx=原 ctx，tenant_id 仍取 TenantIDFromContext），
//     期望归属租户 id 返回空串表示不做归属强校验。
//
// **显式例外（登记）**：内置客户端（platform_admin_web / tenant_admin_web 等，source=builtin）与
// 显式登记的多租户共用外部应用客户端，是「跨租户共用同一 OIDC 客户端」的既定形态——它们的
// tenant_id 归属属于平台侧既定配置，不由本函数的 owner_type 推导强制覆盖。这些例外不经过
// Create 控制台路径（内置客户端由种子写入、共用外部客户端由运维登记），因此本函数不在此处做
// 特判；若未来出现经 Create 创建的例外，须在此显式登记并跳过 owner 强校验，禁止靠 owner_type
// 判断静默放行。
//
// 错误返回仅覆盖“取应用”的系统错误；应用不存在不算错误（保持现行为）。
func (svc *oAuthClientSvc) resolveWriteTenantScope(ctx context.Context, appID string) (context.Context, string, error) {
	if appID == "" {
		return ctx, "", nil
	}
	app, err := dao.NewApplicationDao(dao.WithDBGetter(svc.db)).GetByID(ctx, appID)
	if err != nil {
		return nil, "", err
	}
	if app == nil || app.ID == "" {
		return ctx, "", nil
	}
	if app.OwnerType == model.AppOwnerTypeTenant && app.OwnerTenantID != "" {
		return dbclient.ExplicitTenantContext(ctx, app.OwnerTenantID), app.OwnerTenantID, nil
	}
	return ctx, "", nil
}

func (svc *oAuthClientSvc) Create(ctx context.Context, req *dtoapplicationclient.ApplicationClientCreateReq) (*dtoapplicationclient.ApplicationClientCreateResp, error) {
	// 编码规则校验（校验归 service，与 svcapplication.Create 同款）：客户端编码即 OIDC client_id，
	// 同时是网关侧 audience 白名单值；非法值（连字符/数字/大写/空）必须在此拦截，
	// 而不是落库后再靠人工纠正——写错一个字符会让该客户端签发的令牌全部 401。
	if !model.IsValidClientCode(req.Code) {
		glog.Errorf(ctx, "[svcapplicationclient.Create] 非法客户端编码, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientCodeInvalidError)
	}
	// PKCE 与 auth_time 两个策略来自前端：未提供按 disable 处理，非法值直接拒绝。
	if !isValidClientPKCEPolicy(req.RequirePKCE) {
		glog.Errorf(ctx, "[svcapplicationclient.Create] 非法 PKCE 策略, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientCreateError)
	}
	if !isValidClientAuthTimeClaimPolicy(req.RequireAuthTime) {
		glog.Errorf(ctx, "[svcapplicationclient.Create] 非法 auth_time 策略, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientCreateError)
	}
	// 认证方式留空取列默认（client_secret_basic）；非空必须是白名单值。
	if !isValidTokenEndpointAuthMethod(req.TokenEndpointAuthMethod) {
		glog.Errorf(ctx, "[svcapplicationclient.Create] 非法令牌端点认证方式, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientAuthMethodInvalidError)
	}
	// 归属对齐（评审 H5）：application_client 是租户表，tenant_id 由租户插件按写入 ctx 的作用域落库，
	// 而不是取 entity 上的 TenantID 字段之外的来源。平台代租户建 client 时当前请求作用域是平台
	// （AllScope/平台租户），若直接沿用 TenantIDFromContext(ctx) 会落空串/平台租户 → 租户侧永远查不到。
	// 故按应用归属解析写入作用域：owner_type=tenant 时用 ExplicitTenantContext(app.owner_tenant_id)
	// 让插件按其归属租户落 tenant_id；owner_type=platform 或应用不存在时保持现行为。
	writeCtx, ownerTenantID, err := svc.resolveWriteTenantScope(ctx, req.AppID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Create] 解析归属租户作用域失败, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientCreateError)
	}
	// issuer 写路径（P0）：普通控制台 client 绑当前请求 issuer（平台/租户控制台由入口中间件写入 oidcctx）；
	// D console client 的创建走 onboarding 专用路径（targetTenantID → 目标租户 primary issuer，见 W5）。
	// 禁止从 body 信任 issuer：此处只取服务端请求 ctx 的 issuer。
	issuer, _ := oidcctx.IssuerFromContext(ctx)
	insertEntity := &model.ApplicationClientEntity{
		TenantID:                dbclient.TenantIDFromContext(writeCtx),
		AppID:                   req.AppID,
		Code:                    req.Code,
		Issuer:                  issuer,
		Name:                    req.Name,
		RedirectURIs:            model.RedirectURIList(emptyIfNil(req.RedirectURIs)),
		PostLogoutRedirectURIs:  model.PostLogoutRedirectURIList(emptyIfNil(req.PostLogoutRedirectURIs)),
		BackChannelLogoutURI:    req.BackChannelLogoutURI,
		GrantTypes:              model.GrantTypeList(emptyIfNil(req.GrantTypes)),
		ResponseTypes:           toResponseTypeList(req.ResponseTypes),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		AllowedOrigins:          model.AllowedOriginList(emptyIfNil(req.AllowedOrigins)),
		RequirePKCE:             normalizeClientPKCEPolicy(req.RequirePKCE),
		RequireAuthTime:         normalizeClientAuthTimeClaimPolicy(req.RequireAuthTime),
		DefaultScopes:           model.DefaultScopeList(emptyIfNil(req.DefaultScopes)),
		AccessTokenTTL:          req.AccessTokenTTL,
		RefreshTokenTTL:         req.RefreshTokenTTL,
		Source:                  model.ApplicationClientSourceThirdParty, // 控制台创建的客户端恒为第三方接入
		CreatedBy:               req.Operator.UserID,
	}

	// 不变式校验：owner_type=tenant 的应用，最终落库的 client.tenant_id 必须 == app.owner_tenant_id。
	// 这是「归属租户维护的客户端默认落 owner 租户作用域」的硬约束；若写入作用域解析出错导致不一致，
	// 宁可拒绝也不落一条租户侧查不到的孤儿 client（评审 H5）。内置客户端与多租户共用外部应用为例外，
	// 见 resolveWriteTenantScope 的登记说明。
	if ownerTenantID != "" && insertEntity.TenantID != ownerTenantID {
		glog.Errorf(ctx, "[svcapplicationclient.Create] 不变式违反: client.tenant_id(%q) != app.owner_tenant_id(%q), appID:%s, req:%s",
			insertEntity.TenantID, ownerTenantID, req.AppID, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientCreateError)
	}

	if err := dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).Insert(writeCtx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientCreateError)
	}
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionApplicationClientCreate,
		TenantID:   insertEntity.TenantID,
		Result:     model.AuditResultSuccess,
		TargetType: model.AuditTargetTypeApplicationClient,
		TargetID:   insertEntity.ID,
	})
	return &dtoapplicationclient.ApplicationClientCreateResp{
		ApplicationClientID: insertEntity.ID,
		Code:                insertEntity.Code,
	}, nil
}

func (svc *oAuthClientSvc) Delete(ctx context.Context, req *dtoapplicationclient.ApplicationClientDeleteReq) error {
	entity, err := dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientDeleteError)
	}
	if !applicationClientVisibleToTenant(entity, dbclient.TenantIDFromContext(ctx)) {
		return code.GetError(code.ApplicationClientNotExistError)
	}
	if entity.Source.IsBuiltin() {
		return code.GetError(code.ApplicationClientBuiltInErr)
	}

	userID := req.Operator.UserID
	if err := dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).Delete(ctx, req.ApplicationClientID, userID); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientDeleteError)
	}
	return nil
}

// 内置客户端的写入约束：字段权威矩阵里 application_client 的 immutable 字段是 source、code
// （内置标记与 client_id）与 token_endpoint_auth_method / require_pkce（浏览器公共客户端的安全不变式）。
// source 不在 ApplicationClientUpdateReq 中（控制台无写入入口），其余由下方显式拒绝。
// app_id（归属应用）是 create_only，控制台无改归属入口。
// code（= client_id）**可改，但内置客户端拒改**：它同时是网关 aud 白名单与前端构建期
// client_id 的取值来源，从控制台改会当场把该控制台锁死且无法从界面恢复（见客户端编码方案文档）。
// 名称/回调地址/授权类型/TTL 都归运维（create_only）。
func (svc *oAuthClientSvc) Update(ctx context.Context, req *dtoapplicationclient.ApplicationClientUpdateReq) error {
	if !isValidApplicationClientStatus(req.Status) {
		glog.Errorf(ctx, "[svcapplicationclient.Update] 非法客户端状态, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientUpdateError)
	}
	if !isValidClientPKCEPolicy(req.RequirePKCE) {
		glog.Errorf(ctx, "[svcapplicationclient.Update] 非法 PKCE 策略, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientUpdateError)
	}
	if !isValidClientAuthTimeClaimPolicy(req.RequireAuthTime) {
		glog.Errorf(ctx, "[svcapplicationclient.Update] 非法 auth_time 策略, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientUpdateError)
	}
	if !isValidTokenEndpointAuthMethod(req.TokenEndpointAuthMethod) {
		glog.Errorf(ctx, "[svcapplicationclient.Update] 非法令牌端点认证方式, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientAuthMethodInvalidError)
	}
	entity, err := dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientUpdateError)
	}
	if !applicationClientVisibleToTenant(entity, dbclient.TenantIDFromContext(ctx)) {
		return code.GetError(code.ApplicationClientNotExistError)
	}

	// 内置客户端是浏览器公共客户端，认证方式与强制 PKCE 是安全不变式：
	//   1. RFC 6749 §10.1 禁止为 user-agent 类客户端签发/要求客户端凭据，RFC 10017 §6.3.3.1
	//      要求浏览器客户端登记为 public，故内置客户端必须保持 none + require_pkce=enable；
	//   2. 两个控制台前端只传 client_id、不持密钥，改成 basic/post 后 token 端点要求认证而前端
	//      无法提供 → 登录与静默续期全部 invalid_client，而修复入口恰在该控制台内部（自锁）。
	// 留空表示「本次不修改」：空串不是合法枚举，若照直写入会把存量刷成空值 / disable（见下方列处理）。
	if entity.Source.IsBuiltin() {
		if req.TokenEndpointAuthMethod != "" && req.TokenEndpointAuthMethod != entity.TokenEndpointAuthMethod {
			glog.Errorf(ctx, "[svcapplicationclient.Update] 拒绝修改内置客户端令牌端点认证方式, clientID:%s, req:%s",
				req.ApplicationClientID, gutil.ToJsonString(req))
			return code.GetError(code.ApplicationClientBuiltInAuthMethodImmutableError)
		}
		if req.RequirePKCE != "" && normalizeClientPKCEPolicy(req.RequirePKCE) != entity.RequirePKCE {
			glog.Errorf(ctx, "[svcapplicationclient.Update] 拒绝修改内置客户端强制 PKCE 策略, clientID:%s, req:%s",
				req.ApplicationClientID, gutil.ToJsonString(req))
			return code.GetError(code.ApplicationClientBuiltInAuthMethodImmutableError)
		}
	}

	userID := req.Operator.UserID
	// 走 dao.UpdateFields（结构化 Updates）而非 UpdateMap：JSON 列必须经 GORM serializer 落库，
	// 用 map 写会绕过 serializer 静默落脏值（设计文档 D6）。只填要改的字段，列名与 model 的 column tag 一致。
	updateEntity := &model.ApplicationClientEntity{
		Name:                   req.Name,
		RedirectURIs:           model.RedirectURIList(emptyIfNil(req.RedirectURIs)),
		PostLogoutRedirectURIs: model.PostLogoutRedirectURIList(emptyIfNil(req.PostLogoutRedirectURIs)),
		BackChannelLogoutURI:   req.BackChannelLogoutURI,
		GrantTypes:             model.GrantTypeList(emptyIfNil(req.GrantTypes)),
		ResponseTypes:          toResponseTypeList(req.ResponseTypes),
		AllowedOrigins:         model.AllowedOriginList(emptyIfNil(req.AllowedOrigins)),
		RequireAuthTime:        normalizeClientAuthTimeClaimPolicy(req.RequireAuthTime),
		DefaultScopes:          model.DefaultScopeList(emptyIfNil(req.DefaultScopes)),
		AccessTokenTTL:         req.AccessTokenTTL,
		RefreshTokenTTL:        req.RefreshTokenTTL,
		UpdatedBy:              userID,
	}
	fields := []string{
		"name",
		"redirect_uris",
		"post_logout_redirect_uris",
		"back_channel_logout_uri",
		"grant_types",
		"response_types",
		"allowed_origins",
		"require_auth_time",
		"default_scopes",
		"access_token_ttl",
		"refresh_token_ttl",
		"updated_by",
	}
	// 认证方式留空表示不修改：空串不是合法枚举，照直写入会让该客户端在 oidcop 侧落到 fail-closed
	// 的 private_key_jwt 分支（client.go 的 default 分支）而静默不可用——列必须按「有值才写」处理。
	if req.TokenEndpointAuthMethod != "" {
		updateEntity.TokenEndpointAuthMethod = req.TokenEndpointAuthMethod
		fields = append(fields, "token_endpoint_auth_method")
	}
	// require_pkce 对自建客户端沿用既有契约（空串归一为 disable，原「未提供 = false」行为，已被
	// status_validation_test.go 钉住）；内置客户端则留空即保持——它的 PKCE 强制是安全不变式，
	// 不能被一次省略该字段的 PUT 静默降级。
	if !entity.Source.IsBuiltin() || req.RequirePKCE != "" {
		updateEntity.RequirePKCE = normalizeClientPKCEPolicy(req.RequirePKCE)
		fields = append(fields, "require_pkce")
	}
	// status 留空表示不修改：不写该列，避免把状态覆盖为空串
	if req.Status != "" {
		updateEntity.Status = req.Status
		fields = append(fields, "status")
	}
	// code 留空表示不修改；确有变化时：内置客户端拒改（见函数头注释），其余按创建时的规则校验。
	if req.Code != "" && req.Code != entity.Code {
		if entity.Source.IsBuiltin() {
			glog.Errorf(ctx, "[svcapplicationclient.Update] 拒绝修改内置客户端编码, clientID:%s, req:%s",
				req.ApplicationClientID, gutil.ToJsonString(req))
			return code.GetError(code.ApplicationClientBuiltInCodeImmutableError)
		}
		if !model.IsValidClientCode(req.Code) {
			glog.Errorf(ctx, "[svcapplicationclient.Update] 非法客户端编码, req:%s", gutil.ToJsonString(req))
			return code.GetError(code.ApplicationClientCodeInvalidError)
		}
		updateEntity.Code = req.Code
		fields = append(fields, "code")
	}
	if err := dao.UpdateFields(ctx, dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).Dao, req.ApplicationClientID, updateEntity, fields...); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Update] dao UpdateFields fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientUpdateError)
	}
	return nil
}

func (svc *oAuthClientSvc) Detail(ctx context.Context, req *dtoapplicationclient.ApplicationClientDetailReq) (*dtoapplicationclient.ApplicationClientDetailResp, error) {
	entity, err := dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientGetDetailError)
	}
	if !applicationClientVisibleToTenant(entity, dbclient.TenantIDFromContext(ctx)) {
		return nil, code.GetError(code.ApplicationClientNotExistError)
	}

	detail, err := buildDetailResp(svc.db, ctx, entity)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Detail] buildDetailResp fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientGetDetailError)
	}
	return detail, nil
}

// buildDetailResp 由实体组装详情出参：读取 JSON 列（具名切片）+ 回填所属应用名称。
func buildDetailResp(db gormdao.DBGetter, ctx context.Context, entity *model.ApplicationClientEntity) (*dtoapplicationclient.ApplicationClientDetailResp, error) {
	appNames, err := loadAppNames(db, ctx, model.ApplicationClientEntityList{*entity})
	if err != nil {
		return nil, err
	}
	// JSON 列已是具名切片：GrantTypes 元素类型与出参一致可直接转换，
	// ResponseTypes 元素是具名枚举需逐项转，无需再手工 Unmarshal
	grantTypes := []model.GrantType(entity.GrantTypes)
	responseTypes := responseTypeStrings(entity.ResponseTypes)

	return &dtoapplicationclient.ApplicationClientDetailResp{
		ApplicationClientID:     entity.ID,
		TenantID:                entity.TenantID,
		AppID:                   entity.AppID,
		AppName:                 appNames[entity.AppID],
		Code:                    entity.Code,
		Name:                    entity.Name,
		RedirectURIs:            entity.RedirectURIs.Strings(),
		PostLogoutRedirectURIs:  entity.PostLogoutRedirectURIs.Strings(),
		BackChannelLogoutURI:    entity.BackChannelLogoutURI,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		TokenEndpointAuthMethod: entity.TokenEndpointAuthMethod,
		AllowedOrigins:          entity.AllowedOrigins.Strings(),
		RequirePKCE:             entity.RequirePKCE,
		RequireAuthTime:         entity.RequireAuthTime,
		DefaultScopes:           entity.DefaultScopes.Strings(),
		AccessTokenTTL:          entity.AccessTokenTTL,
		RefreshTokenTTL:         entity.RefreshTokenTTL,
		Source:                  entity.Source,
		Status:                  entity.Status,
		CreatedAt:               entity.CreatedAt.Unix(),
	}, nil
}

func (svc *oAuthClientSvc) PageList(ctx context.Context, req *dtoapplicationclient.ApplicationClientPageListReq) (*dtoapplicationclient.ApplicationClientPageListResp, error) {
	// 非法来源过滤值直接拒绝，避免 DAO 落成「查不到任何数据」的空结果而看不出原因
	if req.Source != "" {
		switch req.Source {
		case model.ApplicationClientSourceBuiltin, model.ApplicationClientSourceFirstParty, model.ApplicationClientSourceThirdParty:
		default:
			glog.Errorf(ctx, "[svcapplicationclient.PageList] 非法 source 过滤值, req:%s", gutil.ToJsonString(req))
			return nil, code.GetError(code.ApplicationClientGetPageListError)
		}
	}
	cond := &dao.ApplicationClientCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: dbclient.TenantIDFromContext(ctx),
		Name:     req.Name,
		Source:   req.Source,
		Status:   req.Status,
	}
	list, total, err := dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientGetPageListError)
	}
	appNames, err := loadAppNames(svc.db, ctx, list)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.PageList] loadAppNames fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientGetPageListError)
	}

	items := make([]dtoapplicationclient.PageListItem, 0, len(list))
	for _, v := range list {
		items = append(items, dtoapplicationclient.PageListItem{
			ApplicationClientID:     v.ID,
			AppID:                   v.AppID,
			AppName:                 appNames[v.AppID],
			Code:                    v.Code,
			Name:                    v.Name,
			Source:                  v.Source,
			Status:                  v.Status,
			GrantTypes:              []model.GrantType(v.GrantTypes),
			TokenEndpointAuthMethod: v.TokenEndpointAuthMethod,
			CreatedAt:               v.CreatedAt.Unix(),
			UpdatedAt:               v.UpdatedAt.Unix(),
		})
	}
	return &dtoapplicationclient.ApplicationClientPageListResp{
		List:  items,
		Total: total,
	}, nil
}

func (svc *oAuthClientSvc) GetByClientID(ctx context.Context, clientID string) (*dtoapplicationclient.ApplicationClientDetailResp, error) {
	issuer, _ := oidcctx.IssuerFromContext(ctx)
	entity, err := dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).GetByCond(ctx, &dao.ApplicationClientCond{
		Code:   clientID,
		Issuer: issuer,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.GetByClientID] dao GetByCond fail, err:%v, clientID:%s", err, clientID)
		return nil, code.GetError(code.ApplicationClientGetDetailError)
	}
	if entity == nil || entity.ID == "" {
		return nil, code.GetError(code.ApplicationClientNotExistError)
	}
	detail, err := buildDetailResp(svc.db, ctx, entity)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.GetByClientID] buildDetailResp fail, err:%v, clientID:%s", err, clientID)
		return nil, code.GetError(code.ApplicationClientGetDetailError)
	}
	return detail, nil
}

// loadAppNames 批量回填所属应用名称（列表/详情一次查询，避免前端按行再查或 N+1）。
// 名称缺失不报错：调用方以空名返回，前端退化为展示应用 ID。
func loadAppNames(db gormdao.DBGetter, ctx context.Context, list model.ApplicationClientEntityList) (map[string]string, error) {
	appIDs := make([]string, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, v := range list {
		if v.AppID == "" {
			continue
		}
		if _, ok := seen[v.AppID]; ok {
			continue
		}
		seen[v.AppID] = struct{}{}
		appIDs = append(appIDs, v.AppID)
	}
	appNames := make(map[string]string, len(appIDs))
	if len(appIDs) == 0 {
		return appNames, nil
	}
	apps, err := dao.NewApplicationDao(dao.WithDBGetter(db)).GetListByCond(ctx, &dao.ApplicationCond{IDs: appIDs})
	if err != nil {
		return nil, err
	}
	for _, a := range apps {
		appNames[a.ID] = a.Name
	}
	return appNames, nil
}

func (svc *oAuthClientSvc) ListSecrets(ctx context.Context, req *dtoapplicationclient.SecretListReq) (*dtoapplicationclient.SecretListResp, error) {
	// 归属校验：与 Detail/Update/DeleteSecret 同口径，非本租户可见的客户端不返回其密钥元数据。
	clientEntity, err := dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.ListSecrets] get client fail, err:%v, clientID:%s", err, req.ApplicationClientID)
		return nil, code.GetError(code.ApplicationClientSecretGetListError)
	}
	if !applicationClientVisibleToTenant(clientEntity, dbclient.TenantIDFromContext(ctx)) {
		return nil, code.GetError(code.ApplicationClientNotExistError)
	}

	secretDao := dao.NewApplicationClientSecretDao(dao.WithDBGetter(svc.db))

	list, total, err := secretDao.GetPageListByCond(ctx, &dao.ApplicationClientSecretCond{
		BaseCond:            &gormdao.BaseCond{Page: 1, PageSize: 100},
		ApplicationClientID: req.ApplicationClientID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.ListSecrets] get secrets fail, err:%v", err)
		return nil, code.GetError(code.ApplicationClientSecretGetListError)
	}

	secrets := make([]dtoapplicationclient.SecretResp, 0, len(list))
	for _, s := range list {
		var expiresAt *int64
		if s.ExpiredAt != nil {
			t := s.ExpiredAt.Unix()
			expiresAt = &t
		}
		secrets = append(secrets, dtoapplicationclient.SecretResp{
			ID:                  s.ID,
			ApplicationClientID: s.ApplicationClientID,
			Name:                s.Name,
			ValuePrefix:         s.ValuePrefix,
			ExpiredAt:           expiresAt,
			CreatedAt:           s.CreatedAt.Unix(),
		})
	}

	return &dtoapplicationclient.SecretListResp{
		Total:   total,
		Secrets: secrets,
	}, nil
}

func (svc *oAuthClientSvc) CreateSecret(ctx context.Context, req *dtoapplicationclient.SecretCreateReq) (*dtoapplicationclient.SecretCreateResp, error) {
	entity, err := dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.CreateSecret] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientSecretCreateError)
	}
	if !applicationClientVisibleToTenant(entity, dbclient.TenantIDFromContext(ctx)) {
		return nil, code.GetError(code.ApplicationClientNotExistError)
	}
	// 公共客户端（token_endpoint_auth_method=none）不得签发密钥：这类客户端的代码/配置会下发给
	// 每个用户，密钥无处安全保存，等于公开（RFC 6749 §2.1）；RFC 6749 §10.1 明确禁止为 user-agent
	// 类客户端签发客户端凭据；且令牌端点本就不会向这类客户端索取认证（client_credentials 通道对
	// AuthMethodNone 显式拒绝），密钥建了也用不上。故按「机密客户端白名单」fail-closed 放行。
	if !isConfidentialAuthMethod(entity.TokenEndpointAuthMethod) {
		glog.Errorf(ctx, "[svcapplicationclient.CreateSecret] 公共客户端不支持密钥, clientID:%s, authMethod:%s, req:%s",
			entity.Code, entity.TokenEndpointAuthMethod, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientPublicClientSecretForbiddenError)
	}

	secretValue, err := credential.GenerateSecret(credential.ClientSecretBytes)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.CreateSecret] credential.GenerateSecret fail, err:%v", err)
		return nil, code.GetError(code.ApplicationClientSecretCreateError)
	}
	valueHash := credential.HashSecret(secretValue)
	valuePrefix := credential.Prefix(secretValue, credential.ClientSecretPrefixLen)

	var expiresAt *time.Time
	if req.ExpiredAt > 0 {
		t := time.Unix(req.ExpiredAt, 0)
		expiresAt = &t
	}

	secretEntity := &model.ApplicationClientSecretEntity{
		ApplicationClientID: req.ApplicationClientID,
		Name:                req.Name,
		ValueHash:           valueHash,
		ValuePrefix:         valuePrefix,
		ExpiredAt:           expiresAt,
		CreatedBy:           req.Operator.UserID,
	}

	if err := dao.NewApplicationClientSecretDao(dao.WithDBGetter(svc.db)).Insert(ctx, secretEntity); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.CreateSecret] insert fail, err:%v", err)
		return nil, code.GetError(code.ApplicationClientSecretCreateError)
	}

	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionApplicationClientCreateSecret,
		TenantID:   entity.TenantID,
		Result:     model.AuditResultSuccess,
		TargetType: model.AuditTargetTypeApplicationClient,
		TargetID:   secretEntity.ApplicationClientID,
	})

	return &dtoapplicationclient.SecretCreateResp{
		ID:          secretEntity.ID,
		Name:        secretEntity.Name,
		ValuePrefix: secretEntity.ValuePrefix,
		Secret:      secretValue,
	}, nil
}

func (svc *oAuthClientSvc) DeleteSecret(ctx context.Context, req *dtoapplicationclient.SecretDeleteReq) error {
	secretEntity, err := dao.NewApplicationClientSecretDao(dao.WithDBGetter(svc.db)).GetByID(ctx, req.SecretID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.DeleteSecret] get secret fail, err:%v", err)
		return code.GetError(code.ApplicationClientSecretDeleteError)
	}
	if secretEntity == nil || secretEntity.ID == "" {
		return code.GetError(code.ApplicationClientSecretNotExistError)
	}

	entity, err := dao.NewApplicationClientDao(dao.WithDBGetter(svc.db)).GetByID(ctx, secretEntity.ApplicationClientID)
	if err != nil {
		// 系统错误与业务边界分离：DB 故障如实上报，不伪装成"密钥不存在"。
		glog.Errorf(ctx, "[svcapplicationclient.DeleteSecret] get client fail, err:%v, clientID:%s", err, secretEntity.ApplicationClientID)
		return code.GetError(code.ApplicationClientSecretDeleteError)
	}
	if !applicationClientVisibleToTenant(entity, dbclient.TenantIDFromContext(ctx)) {
		return code.GetError(code.ApplicationClientSecretNotExistError)
	}

	if err := dao.NewApplicationClientSecretDao(dao.WithDBGetter(svc.db)).Delete(ctx, req.SecretID, req.Operator.UserID); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.DeleteSecret] delete fail, err:%v", err)
		return code.GetError(code.ApplicationClientSecretDeleteError)
	}

	return nil
}
