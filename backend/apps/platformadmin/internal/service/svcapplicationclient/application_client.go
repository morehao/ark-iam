package svcapplicationclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/svcaudit"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapplicationclient"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ApplicationClientSvc interface {
	Create(ctx *gin.Context, req *dtoapplicationclient.CreateReq) (*dtoapplicationclient.CreateResp, error)
	Delete(ctx *gin.Context, req *dtoapplicationclient.DeleteReq) error
	Update(ctx *gin.Context, req *dtoapplicationclient.UpdateReq) error
	Detail(ctx *gin.Context, req *dtoapplicationclient.DetailReq) (*dtoapplicationclient.DetailResp, error)
	PageList(ctx *gin.Context, req *dtoapplicationclient.PageListReq) (*dtoapplicationclient.PageListResp, error)
	GetByClientID(ctx *gin.Context, clientID string) (*dtoapplicationclient.DetailResp, error)
	ListSecrets(ctx *gin.Context, req *dtoapplicationclient.SecretListReq) (*dtoapplicationclient.SecretListResp, error)
	CreateSecret(ctx *gin.Context, req *dtoapplicationclient.CreateSecretReq) (*dtoapplicationclient.CreateSecretResp, error)
	DeleteSecret(ctx *gin.Context, req *dtoapplicationclient.DeleteSecretReq) error
}

type oAuthClientSvc struct{}

type applicationClientScopeRepository interface {
	GetByID(ctx context.Context, id uint) (*model.ApplicationClientEntity, error)
	GetByCond(ctx context.Context, cond gormdao.Cond) (*model.ApplicationClientEntity, error)
	GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.ApplicationClientEntityList, int64, error)
	GetSecretByID(ctx context.Context, id uint) (*model.ApplicationClientSecretEntity, error)
	DeleteSecret(ctx context.Context, id uint, userID uint) error
	Delete(ctx context.Context, id, userID uint) error
}

var newApplicationClientScopeRepo = func() applicationClientScopeRepository {
	return &applicationClientScopeDAO{}
}

type applicationClientScopeDAO struct{}

var newApplicationClientDAO = func() *dao.ApplicationClientDao {
	return dao.NewApplicationClientDao()
}

func (d *applicationClientScopeDAO) GetByID(ctx context.Context, id uint) (*model.ApplicationClientEntity, error) {
	return newApplicationClientDAO().GetByID(ctx, id)
}

func (d *applicationClientScopeDAO) GetByCond(ctx context.Context, cond gormdao.Cond) (*model.ApplicationClientEntity, error) {
	return newApplicationClientDAO().GetByCond(ctx, cond)
}

func (d *applicationClientScopeDAO) GetPageListByCond(ctx context.Context, cond gormdao.Cond) (model.ApplicationClientEntityList, int64, error) {
	return newApplicationClientDAO().GetPageListByCond(ctx, cond)
}

func (d *applicationClientScopeDAO) GetSecretByID(ctx context.Context, id uint) (*model.ApplicationClientSecretEntity, error) {
	return dao.NewApplicationClientSecretDao().GetByID(ctx, id)
}

func (d *applicationClientScopeDAO) DeleteSecret(ctx context.Context, id uint, userID uint) error {
	return dao.NewApplicationClientSecretDao().Delete(ctx, id, userID)
}

func (d *applicationClientScopeDAO) Delete(ctx context.Context, id, userID uint) error {
	return newApplicationClientDAO().Delete(ctx, id, userID)
}

func applicationClientVisibleToTenant(entity *model.ApplicationClientEntity, tenantID uint) bool {
	return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}

var _ ApplicationClientSvc = (*oAuthClientSvc)(nil)

func NewApplicationClientSvc() ApplicationClientSvc {
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

func (svc *oAuthClientSvc) Create(ctx *gin.Context, req *dtoapplicationclient.CreateReq) (*dtoapplicationclient.CreateResp, error) {
	insertEntity := &model.ApplicationClientEntity{
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

	if err := dao.NewApplicationClientDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientCreateError)
	}
	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionApplicationClientCreate,
		TenantID:   insertEntity.TenantID,
		Result:     "success",
		TargetType: "application_client",
		TargetID:   insertEntity.ID,
	})
	return &dtoapplicationclient.CreateResp{
		ApplicationClientID: insertEntity.ID,
		ClientID:            insertEntity.ClientID,
	}, nil
}

func (svc *oAuthClientSvc) Delete(ctx *gin.Context, req *dtoapplicationclient.DeleteReq) error {
	entity, err := newApplicationClientScopeRepo().GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientDeleteError)
	}
	if !applicationClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.ApplicationClientNotExistError)
	}
	if entity.IsSystem == 1 {
		return code.GetError(code.ApplicationClientSystemBuiltInErr)
	}

	userID := gincontext.GetUserID(ctx)
	if err := newApplicationClientScopeRepo().Delete(ctx, req.ApplicationClientID, userID); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientDeleteError)
	}
	return nil
}

func (svc *oAuthClientSvc) Update(ctx *gin.Context, req *dtoapplicationclient.UpdateReq) error {
	entity, err := newApplicationClientScopeRepo().GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientUpdateError)
	}
	if !applicationClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.ApplicationClientNotExistError)
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
	if err := dao.NewApplicationClientDao().UpdateMap(ctx, req.ApplicationClientID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationClientUpdateError)
	}
	return nil
}

func (svc *oAuthClientSvc) Detail(ctx *gin.Context, req *dtoapplicationclient.DetailReq) (*dtoapplicationclient.DetailResp, error) {
	entity, err := newApplicationClientScopeRepo().GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientGetDetailError)
	}
	if !applicationClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.ApplicationClientNotExistError)
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

	return &dtoapplicationclient.DetailResp{
		ApplicationClientID:     entity.ID,
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

func (svc *oAuthClientSvc) PageList(ctx *gin.Context, req *dtoapplicationclient.PageListReq) (*dtoapplicationclient.PageListResp, error) {
	cond := &dao.ApplicationClientCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: gincontext.GetTenantID(ctx),
		Name:     req.Name,
		Type:     req.Type,
		Status:   req.Status,
	}
	list, total, err := newApplicationClientScopeRepo().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientGetPageListError)
	}

	items := make([]dtoapplicationclient.PageListItem, 0, len(list))
	for _, v := range list {
		var grantTypes []string
		_ = json.Unmarshal(v.GrantTypes, &grantTypes)

		items = append(items, dtoapplicationclient.PageListItem{
			ApplicationClientID:     v.ID,
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
	return &dtoapplicationclient.PageListResp{
		List:  items,
		Total: total,
	}, nil
}

func (svc *oAuthClientSvc) GetByClientID(ctx *gin.Context, clientID string) (*dtoapplicationclient.DetailResp, error) {
	entity, err := newApplicationClientScopeRepo().GetByCond(ctx, &dao.ApplicationClientCond{
		ClientID: clientID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.GetByClientID] dao GetByCond fail, err:%v, clientID:%s", err, clientID)
		return nil, code.GetError(code.ApplicationClientGetDetailError)
	}
	if entity == nil || entity.ID == 0 {
		return nil, code.GetError(code.ApplicationClientNotExistError)
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

	return &dtoapplicationclient.DetailResp{
		ApplicationClientID:     entity.ID,
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
		var expiresAt *string
		if s.ExpiredAt != nil {
			t := s.ExpiredAt.Format("2006-01-02 15:04:05")
			expiresAt = &t
		}
		secrets = append(secrets, dtoapplicationclient.SecretResp{
			ID:                  uint64(s.ID),
			ApplicationClientID: uint64(s.ApplicationClientID),
			Name:                s.Name,
			ValuePrefix:         s.ValuePrefix,
			ExpiredAt:           expiresAt,
			CreatedAt:           s.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &dtoapplicationclient.SecretListResp{
		Total:   total,
		Secrets: secrets,
	}, nil
}

func (svc *oAuthClientSvc) CreateSecret(ctx *gin.Context, req *dtoapplicationclient.CreateSecretReq) (*dtoapplicationclient.CreateSecretResp, error) {
	entity, err := newApplicationClientScopeRepo().GetByID(ctx, req.ApplicationClientID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.CreateSecret] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationClientSecretCreateError)
	}
	if !applicationClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.ApplicationClientNotExistError)
	}

	randomBytes, err := gcrypto.GenerateRandomBytes(32)
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.CreateSecret] generate secret fail, err:%v", err)
		return nil, code.GetError(code.ApplicationClientSecretCreateError)
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

	secretEntity := &model.ApplicationClientSecretEntity{
		ApplicationClientID: req.ApplicationClientID,
		Name:                req.Name,
		ValueHash:           valueHash,
		ValuePrefix:         valuePrefix,
		ExpiredAt:           expiresAt,
		CreatedBy:           gincontext.GetUserID(ctx),
	}

	if err := dao.NewApplicationClientSecretDao().Insert(ctx, secretEntity); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.CreateSecret] insert fail, err:%v", err)
		return nil, code.GetError(code.ApplicationClientSecretCreateError)
	}

	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionApplicationClientCreateSecret,
		TenantID:   entity.TenantID,
		Result:     "success",
		TargetType: "application_client",
		TargetID:   secretEntity.ApplicationClientID,
	})

	return &dtoapplicationclient.CreateSecretResp{
		ID:          uint64(secretEntity.ID),
		Name:        secretEntity.Name,
		ValuePrefix: secretEntity.ValuePrefix,
		Secret:      secretValue,
	}, nil
}

func (svc *oAuthClientSvc) DeleteSecret(ctx *gin.Context, req *dtoapplicationclient.DeleteSecretReq) error {
	secretEntity, err := newApplicationClientScopeRepo().GetSecretByID(ctx, uint(req.SecretID))
	if err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.DeleteSecret] get secret fail, err:%v", err)
		return code.GetError(code.ApplicationClientSecretDeleteError)
	}
	if secretEntity == nil || secretEntity.ID == 0 {
		return code.GetError(code.ApplicationClientSecretNotExistError)
	}

	entity, err := newApplicationClientScopeRepo().GetByID(ctx, secretEntity.ApplicationClientID)
	if err != nil || !applicationClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.ApplicationClientSecretNotExistError)
	}

	if err := newApplicationClientScopeRepo().DeleteSecret(ctx, uint(req.SecretID), gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[svcapplicationclient.DeleteSecret] delete fail, err:%v", err)
		return code.GetError(code.ApplicationClientSecretDeleteError)
	}

	return nil
}
