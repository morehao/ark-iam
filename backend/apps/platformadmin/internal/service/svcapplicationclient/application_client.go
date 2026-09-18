package svcapplicationclient

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/audit"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplicationclient"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ApplicationClientSvc interface {
	Create(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientCreateReq) (*dtoapplicationclient.ApplicationClientCreateResp, error)
	Delete(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientDeleteReq) error
	Update(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientUpdateReq) error
	Detail(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientDetailReq) (*dtoapplicationclient.ApplicationClientDetailResp, error)
	PageList(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientPageListReq) (*dtoapplicationclient.ApplicationClientPageListResp, error)
	GetByClientID(ctx *gin.Context, clientID string) (*dtoapplicationclient.ApplicationClientDetailResp, error)
	ListSecrets(ctx *gin.Context, req *dtoapplicationclient.SecretListReq) (*dtoapplicationclient.SecretListResp, error)
	CreateSecret(ctx *gin.Context, req *dtoapplicationclient.SecretCreateReq) (*dtoapplicationclient.SecretCreateResp, error)
	DeleteSecret(ctx *gin.Context, req *dtoapplicationclient.SecretDeleteReq) error
}

type oAuthClientSvc struct{}

func applicationClientVisibleToTenant(entity *model.ApplicationClientEntity, tenantID string) bool {
	return entity != nil && entity.ID != "" && entity.TenantID == tenantID
}

var _ ApplicationClientSvc = (*oAuthClientSvc)(nil)

func NewApplicationClientSvc() ApplicationClientSvc {
	return &oAuthClientSvc{}
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

func (svc *oAuthClientSvc) Create(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientCreateReq) (*dtoapplicationclient.ApplicationClientCreateResp, error) {
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
	insertEntity := &model.ApplicationClientEntity{
		TenantID:                gincontext.GetTenantIDString(ctx),
		AppID:                   req.AppID,
		Code:                    req.Code,
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
		CreatedBy:               gincontext.GetUserIDString(ctx),
	}

	if err := dao.NewApplicationClientDao().Insert(ctx, insertEntity); err != nil {
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

func (svc *oAuthClientSvc) Delete(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientDeleteReq) error {
	entity, err := dao.NewApplicationClientDao().GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientDeleteError)
	}
	if !applicationClientVisibleToTenant(entity, gincontext.GetTenantIDString(ctx)) {
		return code.GetError(code.ApplicationClientNotExistError)
	}
	if entity.Source.IsBuiltin() {
		return code.GetError(code.ApplicationClientBuiltInErr)
	}

	userID := gincontext.GetUserIDString(ctx)
	if err := dao.NewApplicationClientDao().Delete(ctx, req.ApplicationClientID, userID); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientDeleteError)
	}
	return nil
}

// 内置客户端的写入约束：字段权威矩阵里 application_client 的 reconcile 字段只有 source 与
// app_id（内置标记与归属应用），两者都不在 ApplicationClientUpdateReq 中，控制台无写入入口。
// code（= client_id）**可改，但内置客户端拒改**：它同时是网关 aud 白名单与前端构建期
// client_id 的取值来源，从控制台改会当场把该控制台锁死且无法从界面恢复（见客户端编码方案文档）。
// 名称/回调地址/授权类型/TTL 都归运维（create_only）。
func (svc *oAuthClientSvc) Update(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientUpdateReq) error {
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
	entity, err := dao.NewApplicationClientDao().GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientUpdateError)
	}
	if !applicationClientVisibleToTenant(entity, gincontext.GetTenantIDString(ctx)) {
		return code.GetError(code.ApplicationClientNotExistError)
	}

	userID := gincontext.GetUserIDString(ctx)
	// 走 dao.UpdateFields（结构化 Updates）而非 UpdateMap：JSON 列必须经 GORM serializer 落库，
	// 用 map 写会绕过 serializer 静默落脏值（设计文档 D6）。只填要改的字段，列名与 model 的 column tag 一致。
	updateEntity := &model.ApplicationClientEntity{
		Name:                    req.Name,
		RedirectURIs:            model.RedirectURIList(emptyIfNil(req.RedirectURIs)),
		PostLogoutRedirectURIs:  model.PostLogoutRedirectURIList(emptyIfNil(req.PostLogoutRedirectURIs)),
		BackChannelLogoutURI:    req.BackChannelLogoutURI,
		GrantTypes:              model.GrantTypeList(emptyIfNil(req.GrantTypes)),
		ResponseTypes:           toResponseTypeList(req.ResponseTypes),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		AllowedOrigins:          model.AllowedOriginList(emptyIfNil(req.AllowedOrigins)),
		// 两个策略在更新中是全量字段（原 bool 亦为全量写入）：空串归一为 disable，保持原「未提供 = false」行为
		RequirePKCE:     normalizeClientPKCEPolicy(req.RequirePKCE),
		RequireAuthTime: normalizeClientAuthTimeClaimPolicy(req.RequireAuthTime),
		DefaultScopes:   model.DefaultScopeList(emptyIfNil(req.DefaultScopes)),
		AccessTokenTTL:  req.AccessTokenTTL,
		RefreshTokenTTL: req.RefreshTokenTTL,
		UpdatedBy:       userID,
	}
	fields := []string{
		"name",
		"redirect_uris",
		"post_logout_redirect_uris",
		"back_channel_logout_uri",
		"grant_types",
		"response_types",
		"token_endpoint_auth_method",
		"allowed_origins",
		"require_pkce",
		"require_auth_time",
		"default_scopes",
		"access_token_ttl",
		"refresh_token_ttl",
		"updated_by",
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
	if err := dao.UpdateFields(ctx, dao.NewApplicationClientDao().Dao, req.ApplicationClientID, updateEntity, fields...); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Update] dao UpdateFields fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientUpdateError)
	}
	return nil
}

func (svc *oAuthClientSvc) Detail(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientDetailReq) (*dtoapplicationclient.ApplicationClientDetailResp, error) {
	entity, err := dao.NewApplicationClientDao().GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientGetDetailError)
	}
	if !applicationClientVisibleToTenant(entity, gincontext.GetTenantIDString(ctx)) {
		return nil, code.GetError(code.ApplicationClientNotExistError)
	}

	detail, err := buildDetailResp(ctx, entity)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Detail] buildDetailResp fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientGetDetailError)
	}
	return detail, nil
}

// buildDetailResp 由实体组装详情出参：读取 JSON 列（具名切片）+ 回填所属应用名称。
func buildDetailResp(ctx *gin.Context, entity *model.ApplicationClientEntity) (*dtoapplicationclient.ApplicationClientDetailResp, error) {
	appNames, err := loadAppNames(ctx, model.ApplicationClientEntityList{*entity})
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

func (svc *oAuthClientSvc) PageList(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientPageListReq) (*dtoapplicationclient.ApplicationClientPageListResp, error) {
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
		TenantID: gincontext.GetTenantIDString(ctx),
		Name:     req.Name,
		Source:   req.Source,
		Status:   req.Status,
	}
	list, total, err := dao.NewApplicationClientDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientGetPageListError)
	}
	appNames, err := loadAppNames(ctx, list)
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

func (svc *oAuthClientSvc) GetByClientID(ctx *gin.Context, clientID string) (*dtoapplicationclient.ApplicationClientDetailResp, error) {
	entity, err := dao.NewApplicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{
		Code: clientID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.GetByClientID] dao GetByCond fail, err:%v, clientID:%s", err, clientID)
		return nil, code.GetError(code.ApplicationClientGetDetailError)
	}
	if entity == nil || entity.ID == "" {
		return nil, code.GetError(code.ApplicationClientNotExistError)
	}
	detail, err := buildDetailResp(ctx, entity)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.GetByClientID] buildDetailResp fail, err:%v, clientID:%s", err, clientID)
		return nil, code.GetError(code.ApplicationClientGetDetailError)
	}
	return detail, nil
}

// loadAppNames 批量回填所属应用名称（列表/详情一次查询，避免前端按行再查或 N+1）。
// 名称缺失不报错：调用方以空名返回，前端退化为展示应用 ID。
func loadAppNames(ctx *gin.Context, list model.ApplicationClientEntityList) (map[string]string, error) {
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
	apps, err := dao.NewApplicationDao().GetListByCond(ctx, &dao.ApplicationCond{IDs: appIDs})
	if err != nil {
		return nil, err
	}
	for _, a := range apps {
		appNames[a.ID] = a.Name
	}
	return appNames, nil
}

func (svc *oAuthClientSvc) ListSecrets(ctx *gin.Context, req *dtoapplicationclient.SecretListReq) (*dtoapplicationclient.SecretListResp, error) {
	secretDao := dao.NewApplicationClientSecretDao()

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

func (svc *oAuthClientSvc) CreateSecret(ctx *gin.Context, req *dtoapplicationclient.SecretCreateReq) (*dtoapplicationclient.SecretCreateResp, error) {
	entity, err := dao.NewApplicationClientDao().GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.CreateSecret] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientSecretCreateError)
	}
	if !applicationClientVisibleToTenant(entity, gincontext.GetTenantIDString(ctx)) {
		return nil, code.GetError(code.ApplicationClientNotExistError)
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
		CreatedBy:           gincontext.GetUserIDString(ctx),
	}

	if err := dao.NewApplicationClientSecretDao().Insert(ctx, secretEntity); err != nil {
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

func (svc *oAuthClientSvc) DeleteSecret(ctx *gin.Context, req *dtoapplicationclient.SecretDeleteReq) error {
	secretEntity, err := dao.NewApplicationClientSecretDao().GetByID(ctx, req.SecretID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.DeleteSecret] get secret fail, err:%v", err)
		return code.GetError(code.ApplicationClientSecretDeleteError)
	}
	if secretEntity == nil || secretEntity.ID == "" {
		return code.GetError(code.ApplicationClientSecretNotExistError)
	}

	entity, err := dao.NewApplicationClientDao().GetByID(ctx, secretEntity.ApplicationClientID)
	if err != nil || !applicationClientVisibleToTenant(entity, gincontext.GetTenantIDString(ctx)) {
		return code.GetError(code.ApplicationClientSecretNotExistError)
	}

	if err := dao.NewApplicationClientSecretDao().Delete(ctx, req.SecretID, gincontext.GetUserIDString(ctx)); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.DeleteSecret] delete fail, err:%v", err)
		return code.GetError(code.ApplicationClientSecretDeleteError)
	}

	return nil
}
