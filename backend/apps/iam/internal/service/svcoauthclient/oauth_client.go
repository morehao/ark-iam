package svcoauthclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtooauthclient"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type OAuthClientSvc interface {
	Create(ctx *gin.Context, req *dtooauthclient.CreateReq) (*dtooauthclient.CreateResp, error)
	Delete(ctx *gin.Context, req *dtooauthclient.DeleteReq) error
	Update(ctx *gin.Context, req *dtooauthclient.UpdateReq) error
	Detail(ctx *gin.Context, req *dtooauthclient.DetailReq) (*dtooauthclient.DetailResp, error)
	PageList(ctx *gin.Context, req *dtooauthclient.PageListReq) (*dtooauthclient.PageListResp, error)
	GetByClientID(ctx *gin.Context, clientID string) (*dtooauthclient.DetailResp, error)
	ListSecrets(ctx *gin.Context, req *dtooauthclient.SecretListReq) (*dtooauthclient.SecretListResp, error)
	CreateSecret(ctx *gin.Context, req *dtooauthclient.CreateSecretReq) (*dtooauthclient.CreateSecretResp, error)
	DeleteSecret(ctx *gin.Context, req *dtooauthclient.DeleteSecretReq) error
}

type oAuthClientSvc struct{}

type oauthClientScopeRepository interface {
	GetByID(ctx context.Context, id uint) (*model.OAuthClientEntity, error)
	GetByCond(ctx context.Context, cond gormdao.Cond) (*model.OAuthClientEntity, error)
	GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.OAuthClientEntityList, int64, error)
	GetSecretByID(ctx context.Context, id uint) (*model.OAuthClientSecretEntity, error)
	DeleteSecret(ctx context.Context, id uint, userID uint) error
}

var newOAuthClientScopeRepo = func() oauthClientScopeRepository {
	return &oauthClientScopeDAO{}
}

type oauthClientScopeDAO struct{}

func (d *oauthClientScopeDAO) GetByID(ctx context.Context, id uint) (*model.OAuthClientEntity, error) {
	return dao.NewOAuthClientDao().GetByID(ctx, id)
}

func (d *oauthClientScopeDAO) GetByCond(ctx context.Context, cond gormdao.Cond) (*model.OAuthClientEntity, error) {
	return dao.NewOAuthClientDao().GetByCond(ctx, cond)
}

func (d *oauthClientScopeDAO) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.OAuthClientEntityList, int64, error) {
	return dao.NewOAuthClientDao().GetPageListByCond(ctx, cond)
}

func (d *oauthClientScopeDAO) GetSecretByID(ctx context.Context, id uint) (*model.OAuthClientSecretEntity, error) {
	return dao.NewOAuthClientSecretDao().GetByID(ctx, id)
}

func (d *oauthClientScopeDAO) DeleteSecret(ctx context.Context, id uint, userID uint) error {
	return dao.NewOAuthClientSecretDao().Delete(ctx, id, userID)
}

func oauthClientVisibleToTenant(entity *model.OAuthClientEntity, tenantID uint) bool {
	return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}

var _ OAuthClientSvc = (*oAuthClientSvc)(nil)

func NewOAuthClientSvc() OAuthClientSvc {
	return &oAuthClientSvc{}
}

func generateClientID() string {
	return uuid.New().String()
}

func marshalStringSlice(s []string) datatypes.JSON {
	if s == nil {
		s = []string{}
	}
	b, _ := json.Marshal(s)
	return datatypes.JSON(b)
}

func (svc *oAuthClientSvc) Create(ctx *gin.Context, req *dtooauthclient.CreateReq) (*dtooauthclient.CreateResp, error) {
	insertEntity := &model.OAuthClientEntity{
		TenantID:                gincontext.GetTenantID(ctx),
		AppID:                   req.AppId,
		ClientID:                generateClientID(),
		Name:                    req.Name,
		RedirectURIs:            marshalStringSlice(req.RedirectURIs),
		PostLogoutRedirectURIs:  marshalStringSlice(req.PostLogoutRedirectURIs),
		GrantTypes:              marshalStringSlice(req.GrantTypes),
		ResponseTypes:           marshalStringSlice(req.ResponseTypes),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		AllowedOrigins:          marshalStringSlice(req.AllowedOrigins),
		RequirePKCE:             req.RequirePKCE,
		RequireAuthTime:         req.RequireAuthTime,
		DefaultScopes:           marshalStringSlice(req.DefaultScopes),
		AccessTokenTTL:          req.AccessTokenTTL,
		RefreshTokenTTL:         req.RefreshTokenTTL,
		Type:                    req.Type,
		IsThirdParty:            req.IsThirdParty,
		CreatedBy:               gincontext.GetUserID(ctx),
	}

	if err := dao.NewOAuthClientDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcoauthclient.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OAuthClientCreateError)
	}
	return &dtooauthclient.CreateResp{
		OAuthClientID: insertEntity.ID,
		ClientID:      insertEntity.ClientID,
	}, nil
}

func (svc *oAuthClientSvc) Delete(ctx *gin.Context, req *dtooauthclient.DeleteReq) error {
	entity, err := newOAuthClientScopeRepo().GetByID(ctx, req.OAuthClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcoauthclient.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OAuthClientDeleteError)
	}
	if !oauthClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.OAuthClientNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewOAuthClientDao().Delete(ctx, req.OAuthClientID, userID); err != nil {
		glog.Errorf(ctx, "[svcoauthclient.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OAuthClientDeleteError)
	}
	return nil
}

func (svc *oAuthClientSvc) Update(ctx *gin.Context, req *dtooauthclient.UpdateReq) error {
	entity, err := newOAuthClientScopeRepo().GetByID(ctx, req.OAuthClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcoauthclient.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OAuthClientUpdateError)
	}
	if !oauthClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.OAuthClientNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"name":                       req.Name,
		"type":                       req.Type,
		"status":                     req.Status,
		"is_third_party":             req.IsThirdParty,
		"redirect_uris":              marshalStringSlice(req.RedirectURIs),
		"post_logout_redirect_uris":  marshalStringSlice(req.PostLogoutRedirectURIs),
		"grant_types":                marshalStringSlice(req.GrantTypes),
		"response_types":             marshalStringSlice(req.ResponseTypes),
		"token_endpoint_auth_method": req.TokenEndpointAuthMethod,
		"allowed_origins":            marshalStringSlice(req.AllowedOrigins),
		"require_pkce":               req.RequirePKCE,
		"require_auth_time":          req.RequireAuthTime,
		"default_scopes":             marshalStringSlice(req.DefaultScopes),
		"access_token_ttl":           req.AccessTokenTTL,
		"refresh_token_ttl":          req.RefreshTokenTTL,
		"updated_by":                 userID,
	}
	if err := dao.NewOAuthClientDao().UpdateMap(ctx, req.OAuthClientID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcoauthclient.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.OAuthClientUpdateError)
	}
	return nil
}

func (svc *oAuthClientSvc) Detail(ctx *gin.Context, req *dtooauthclient.DetailReq) (*dtooauthclient.DetailResp, error) {
	entity, err := newOAuthClientScopeRepo().GetByID(ctx, req.OAuthClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcoauthclient.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OAuthClientGetDetailError)
	}
	if !oauthClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.OAuthClientNotExistError)
	}

	var redirectURIs, postLogoutRedirectURIs []string
	var grantTypes, responseTypes []string
	var allowedOrigins, defaultScopes []string
	_ = json.Unmarshal(entity.RedirectURIs, &redirectURIs)
	_ = json.Unmarshal(entity.PostLogoutRedirectURIs, &postLogoutRedirectURIs)
	_ = json.Unmarshal(entity.GrantTypes, &grantTypes)
	_ = json.Unmarshal(entity.ResponseTypes, &responseTypes)
	_ = json.Unmarshal(entity.AllowedOrigins, &allowedOrigins)
	_ = json.Unmarshal(entity.DefaultScopes, &defaultScopes)

	return &dtooauthclient.DetailResp{
		OAuthClientID:           entity.ID,
		TenantID:                entity.TenantID,
		AppID:                   entity.AppID,
		ClientID:                entity.ClientID,
		Name:                    entity.Name,
		RedirectURIs:            redirectURIs,
		PostLogoutRedirectURIs:  postLogoutRedirectURIs,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		TokenEndpointAuthMethod: entity.TokenEndpointAuthMethod,
		AllowedOrigins:          allowedOrigins,
		RequirePKCE:             entity.RequirePKCE,
		RequireAuthTime:         entity.RequireAuthTime,
		DefaultScopes:           defaultScopes,
		AccessTokenTTL:          entity.AccessTokenTTL,
		RefreshTokenTTL:         entity.RefreshTokenTTL,
		Type:                    entity.Type,
		IsThirdParty:            entity.IsThirdParty,
		Status:                  entity.Status,
		CreatedAt:               entity.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (svc *oAuthClientSvc) PageList(ctx *gin.Context, req *dtooauthclient.PageListReq) (*dtooauthclient.PageListResp, error) {
	cond := &dao.OAuthClientCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: gincontext.GetTenantID(ctx),
		Name:     req.Name,
		Type:     req.Type,
		Status:   req.Status,
	}
	list, total, err := newOAuthClientScopeRepo().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcoauthclient.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OAuthClientGetPageListError)
	}

	items := make([]dtooauthclient.PageListItem, 0, len(list))
	for _, v := range list {
		var grantTypes []string
		_ = json.Unmarshal(v.GrantTypes, &grantTypes)

		items = append(items, dtooauthclient.PageListItem{
			OAuthClientID:           v.ID,
			AppID:                   v.AppID,
			ClientID:                v.ClientID,
			Name:                    v.Name,
			Type:                    v.Type,
			Status:                  v.Status,
			IsThirdParty:            v.IsThirdParty,
			GrantTypes:              grantTypes,
			TokenEndpointAuthMethod: v.TokenEndpointAuthMethod,
			CreatedAt:               v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &dtooauthclient.PageListResp{
		List:  items,
		Total: total,
	}, nil
}

func (svc *oAuthClientSvc) GetByClientID(ctx *gin.Context, clientID string) (*dtooauthclient.DetailResp, error) {
	entity, err := newOAuthClientScopeRepo().GetByCond(ctx, &dao.OAuthClientCond{
		ClientID: clientID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcoauthclient.GetByClientID] dao GetByCond fail, err:%v, clientID:%s", err, clientID)
		return nil, code.GetError(code.OAuthClientGetDetailError)
	}
	if entity == nil || entity.ID == 0 {
		return nil, code.GetError(code.OAuthClientNotExistError)
	}
	var redirectURIs, postLogoutRedirectURIs []string
	var grantTypes, responseTypes []string
	var allowedOrigins, defaultScopes []string
	_ = json.Unmarshal(entity.RedirectURIs, &redirectURIs)
	_ = json.Unmarshal(entity.PostLogoutRedirectURIs, &postLogoutRedirectURIs)
	_ = json.Unmarshal(entity.GrantTypes, &grantTypes)
	_ = json.Unmarshal(entity.ResponseTypes, &responseTypes)
	_ = json.Unmarshal(entity.AllowedOrigins, &allowedOrigins)
	_ = json.Unmarshal(entity.DefaultScopes, &defaultScopes)

	return &dtooauthclient.DetailResp{
		OAuthClientID:           entity.ID,
		TenantID:                entity.TenantID,
		AppID:                   entity.AppID,
		ClientID:                entity.ClientID,
		Name:                    entity.Name,
		RedirectURIs:            redirectURIs,
		PostLogoutRedirectURIs:  postLogoutRedirectURIs,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		TokenEndpointAuthMethod: entity.TokenEndpointAuthMethod,
		AllowedOrigins:          allowedOrigins,
		RequirePKCE:             entity.RequirePKCE,
		RequireAuthTime:         entity.RequireAuthTime,
		DefaultScopes:           defaultScopes,
		AccessTokenTTL:          entity.AccessTokenTTL,
		RefreshTokenTTL:         entity.RefreshTokenTTL,
		Type:                    entity.Type,
		IsThirdParty:            entity.IsThirdParty,
		Status:                  entity.Status,
		CreatedAt:               entity.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

func (svc *oAuthClientSvc) ListSecrets(ctx *gin.Context, req *dtooauthclient.SecretListReq) (*dtooauthclient.SecretListResp, error) {
	secretDao := dao.NewOAuthClientSecretDao()

	list, total, err := secretDao.GetPageListByCond(ctx, &dao.OAuthClientSecretCond{
		BaseCond:      &gormdao.BaseCond{Page: 1, PageSize: 100},
		OAuthClientID: req.OAuthClientID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcoauthclient.ListSecrets] get secrets fail, err:%v", err)
		return nil, code.GetError(code.OAuthClientSecretGetListError)
	}

	secrets := make([]dtooauthclient.SecretResp, 0, len(list))
	for _, s := range list {
		var expiresAt *string
		if s.ExpiredAt != nil {
			t := s.ExpiredAt.Format("2006-01-02 15:04:05")
			expiresAt = &t
		}
		secrets = append(secrets, dtooauthclient.SecretResp{
			ID:            uint64(s.ID),
			OAuthClientID: uint64(s.OAuthClientID),
			Name:          s.Name,
			ValuePrefix:   s.ValuePrefix,
			ExpiredAt:     expiresAt,
			CreatedAt:     s.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &dtooauthclient.SecretListResp{
		Total:   total,
		Secrets: secrets,
	}, nil
}

func (svc *oAuthClientSvc) CreateSecret(ctx *gin.Context, req *dtooauthclient.CreateSecretReq) (*dtooauthclient.CreateSecretResp, error) {
	entity, err := newOAuthClientScopeRepo().GetByID(ctx, req.OAuthClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcoauthclient.CreateSecret] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.OAuthClientSecretCreateError)
	}
	if !oauthClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.OAuthClientNotExistError)
	}

	randomBytes, err := gcrypto.GenerateRandomBytes(32)
	if err != nil {
		glog.Errorf(ctx, "[svcoauthclient.CreateSecret] generate secret fail, err:%v", err)
		return nil, code.GetError(code.OAuthClientSecretCreateError)
	}
	secretValue := hex.EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(secretValue))
	valueHash := hex.EncodeToString(hash[:])

	prefixLen := 8
	if len(secretValue) < prefixLen {
		prefixLen = len(secretValue)
	}
	valuePrefix := secretValue[:prefixLen]

	var expiresAt *time.Time
	if req.ExpiredAt != "" {
		t, err := time.Parse("2006-01-02 15:04:05", req.ExpiredAt)
		if err == nil {
			expiresAt = &t
		}
	}

	secretEntity := &model.OAuthClientSecretEntity{
		OAuthClientID: req.OAuthClientID,
		Name:          req.Name,
		ValueHash:     valueHash,
		ValuePrefix:   valuePrefix,
		ExpiredAt:     expiresAt,
		CreatedBy:     gincontext.GetUserID(ctx),
	}

	if err := dao.NewOAuthClientSecretDao().Insert(ctx, secretEntity); err != nil {
		glog.Errorf(ctx, "[svcoauthclient.CreateSecret] insert fail, err:%v", err)
		return nil, code.GetError(code.OAuthClientSecretCreateError)
	}

	return &dtooauthclient.CreateSecretResp{
		ID:          uint64(secretEntity.ID),
		Name:        secretEntity.Name,
		ValuePrefix: secretEntity.ValuePrefix,
		Secret:      secretValue,
	}, nil
}

func (svc *oAuthClientSvc) DeleteSecret(ctx *gin.Context, req *dtooauthclient.DeleteSecretReq) error {
	secretEntity, err := newOAuthClientScopeRepo().GetSecretByID(ctx, uint(req.SecretID))
	if err != nil {
		glog.Errorf(ctx, "[svcoauthclient.DeleteSecret] get secret fail, err:%v", err)
		return code.GetError(code.OAuthClientSecretDeleteError)
	}
	if secretEntity == nil || secretEntity.ID == 0 {
		return code.GetError(code.OAuthClientSecretNotExistError)
	}

	entity, err := newOAuthClientScopeRepo().GetByID(ctx, secretEntity.OAuthClientID)
	if err != nil || !oauthClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.OAuthClientSecretNotExistError)
	}

	if err := newOAuthClientScopeRepo().DeleteSecret(ctx, uint(req.SecretID), gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[svcoauthclient.DeleteSecret] delete fail, err:%v", err)
		return code.GetError(code.OAuthClientSecretDeleteError)
	}

	return nil
}
