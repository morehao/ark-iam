package svcapplicationclient

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"

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

func generateClientCode() string {
	return uuid.New().String()
}

// marshalJSONSlice 将切片序列化为 JSON 列（nil 落空数组）。
// 元素可为任意类型：枚举具名类型底层是 string，序列化结果与 []string 一致。
func marshalJSONSlice[T any](s []T) datatypes.JSON {
	if s == nil {
		s = []T{}
	}
	b, _ := json.Marshal(s)
	return datatypes.JSON(b)
}

func (svc *oAuthClientSvc) Create(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientCreateReq) (*dtoapplicationclient.ApplicationClientCreateResp, error) {
	insertEntity := &model.ApplicationClientEntity{
		TenantID:                gincontext.GetTenantIDString(ctx),
		AppID:                   req.AppID,
		Code:                    generateClientCode(),
		Name:                    req.Name,
		RedirectURIs:            marshalJSONSlice(req.RedirectURIs),
		PostLogoutRedirectURIs:  marshalJSONSlice(req.PostLogoutRedirectURIs),
		BackChannelLogoutURI:    req.BackChannelLogoutURI,
		GrantTypes:              marshalJSONSlice(req.GrantTypes),
		ResponseTypes:           marshalJSONSlice(req.ResponseTypes),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		AllowedOrigins:          marshalJSONSlice(req.AllowedOrigins),
		RequirePKCE:             req.RequirePKCE,
		RequireAuthTime:         req.RequireAuthTime,
		DefaultScopes:           marshalJSONSlice(req.DefaultScopes),
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

// clientSeedFieldsChanged 判断请求是否改动了种子拥有的客户端字段（字段权威矩阵：
// application_client 的 reconcile 字段 = name）。内置客户端名称由平台版本定义，控制台拒写。
func clientSeedFieldsChanged(entity *model.ApplicationClientEntity, req *dtoapplicationclient.ApplicationClientUpdateReq) bool {
	current := map[string]any{"name": entity.Name}
	desired := map[string]any{"name": req.Name}
	for field, want := range desired {
		if model.SeedOwnsField(model.SeedEntityApplicationClient, field) && current[field] != want {
			return true
		}
	}
	return false
}

func (svc *oAuthClientSvc) Update(ctx *gin.Context, req *dtoapplicationclient.ApplicationClientUpdateReq) error {
	if !isValidApplicationClientStatus(req.Status) {
		glog.Errorf(ctx, "[svcapplicationclient.Update] 非法客户端状态, req:%s", gutil.ToJsonString(req))
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
	if entity.Source == model.ApplicationClientSourceBuiltin && clientSeedFieldsChanged(entity, req) {
		glog.Errorf(ctx, "[svcapplicationclient.Update] 拒绝修改内置客户端名称, clientID:%s, req:%s", req.ApplicationClientID, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientBuiltInFieldImmutableError)
	}

	userID := gincontext.GetUserIDString(ctx)
	updateMap := map[string]any{
		"name":                       req.Name,
		"redirect_uris":              marshalJSONSlice(req.RedirectURIs),
		"post_logout_redirect_uris":  marshalJSONSlice(req.PostLogoutRedirectURIs),
		"back_channel_logout_uri":    req.BackChannelLogoutURI,
		"grant_types":                marshalJSONSlice(req.GrantTypes),
		"response_types":             marshalJSONSlice(req.ResponseTypes),
		"token_endpoint_auth_method": req.TokenEndpointAuthMethod,
		"allowed_origins":            marshalJSONSlice(req.AllowedOrigins),
		"require_pkce":               req.RequirePKCE,
		"require_auth_time":          req.RequireAuthTime,
		"default_scopes":             marshalJSONSlice(req.DefaultScopes),
		"access_token_ttl":           req.AccessTokenTTL,
		"refresh_token_ttl":          req.RefreshTokenTTL,
		"updated_by":                 userID,
	}
	// status 留空表示不修改：不写该列，避免把状态覆盖为空串
	if req.Status != "" {
		updateMap["status"] = req.Status
	}
	if err := dao.NewApplicationClientDao().UpdateMap(ctx, req.ApplicationClientID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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

// buildDetailResp 由实体组装详情出参：解码 JSON 列 + 回填所属应用名称。
func buildDetailResp(ctx *gin.Context, entity *model.ApplicationClientEntity) (*dtoapplicationclient.ApplicationClientDetailResp, error) {
	appNames, err := loadAppNames(ctx, model.ApplicationClientEntityList{*entity})
	if err != nil {
		return nil, err
	}
	var redirectURIs, postLogoutRedirectURIs []string
	var grantTypes []model.GrantType
	var responseTypes []string
	var allowedOrigins, defaultScopes []string
	_ = json.Unmarshal(entity.RedirectURIs, &redirectURIs)
	_ = json.Unmarshal(entity.PostLogoutRedirectURIs, &postLogoutRedirectURIs)
	_ = json.Unmarshal(entity.GrantTypes, &grantTypes)
	_ = json.Unmarshal(entity.ResponseTypes, &responseTypes)
	_ = json.Unmarshal(entity.AllowedOrigins, &allowedOrigins)
	_ = json.Unmarshal(entity.DefaultScopes, &defaultScopes)

	return &dtoapplicationclient.ApplicationClientDetailResp{
		ApplicationClientID:     entity.ID,
		TenantID:                entity.TenantID,
		AppID:                   entity.AppID,
		AppName:                 appNames[entity.AppID],
		Code:                    entity.Code,
		Name:                    entity.Name,
		RedirectURIs:            redirectURIs,
		PostLogoutRedirectURIs:  postLogoutRedirectURIs,
		BackChannelLogoutURI:    entity.BackChannelLogoutURI,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		TokenEndpointAuthMethod: entity.TokenEndpointAuthMethod,
		AllowedOrigins:          allowedOrigins,
		RequirePKCE:             entity.RequirePKCE,
		RequireAuthTime:         entity.RequireAuthTime,
		DefaultScopes:           defaultScopes,
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
		var grantTypes []model.GrantType
		_ = json.Unmarshal(v.GrantTypes, &grantTypes)

		items = append(items, dtoapplicationclient.PageListItem{
			ApplicationClientID:     v.ID,
			AppID:                   v.AppID,
			AppName:                 appNames[v.AppID],
			Code:                    v.Code,
			Name:                    v.Name,
			Source:                  v.Source,
			Status:                  v.Status,
			GrantTypes:              grantTypes,
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
