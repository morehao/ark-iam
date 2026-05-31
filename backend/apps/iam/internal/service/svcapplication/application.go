package svcapplication

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
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objapplication"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type ApplicationSvc interface {
	Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error)
	Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error
	Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error
	Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error)
	PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error)
	GetByClientID(ctx *gin.Context, clientID string) (*dtoapplication.ApplicationDetailResp, error)
	ListRoles(ctx *gin.Context, req *dtoapplication.ApplicationRoleListReq) (*dtoapplication.ApplicationRoleListResp, error)
	AssignRoles(ctx *gin.Context, req *dtoapplication.AssignApplicationRolesReq) error
	RemoveRole(ctx *gin.Context, req *dtoapplication.RemoveApplicationRoleReq) error
	ListSecrets(ctx *gin.Context, req *dtoapplication.ApplicationSecretListReq) (*dtoapplication.ApplicationSecretListResp, error)
	CreateSecret(ctx *gin.Context, req *dtoapplication.CreateApplicationSecretReq) (*dtoapplication.CreateApplicationSecretResp, error)
	DeleteSecret(ctx *gin.Context, req *dtoapplication.DeleteApplicationSecretReq) error
}

type applicationSvc struct{}

type applicationRoleListReader interface {
	GetListByCond(ctx context.Context, cond genericdao.Cond) (model.ApplicationRoleEntityList, error)
}

type roleReader interface {
	GetByID(ctx context.Context, id uint) (*model.RoleEntity, error)
}

type applicationScopeRepository interface {
	GetByID(ctx context.Context, id uint) (*model.ApplicationEntity, error)
	GetByCond(ctx context.Context, cond genericdao.Cond) (*model.ApplicationEntity, error)
	GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.ApplicationEntityList, int64, error)
	GetSecretByID(ctx context.Context, id uint) (*model.ApplicationSecretEntity, error)
	DeleteSecret(ctx context.Context, id uint, userID uint) error
}

var newApplicationRoleListReader = func() applicationRoleListReader {
	return dao.NewApplicationRoleDao()
}

var newRoleReader = func() roleReader {
	return dao.NewRoleDao()
}

var newApplicationScopeRepo = func() applicationScopeRepository {
	return &applicationScopeDAO{}
}

type applicationScopeDAO struct{}

func (d *applicationScopeDAO) GetByID(ctx context.Context, id uint) (*model.ApplicationEntity, error) {
	return dao.NewApplicationDao().GetByID(ctx, id)
}

func (d *applicationScopeDAO) GetByCond(ctx context.Context, cond genericdao.Cond) (*model.ApplicationEntity, error) {
	return dao.NewApplicationDao().GetByCond(ctx, cond)
}

func (d *applicationScopeDAO) GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.ApplicationEntityList, int64, error) {
	return dao.NewApplicationDao().GetPageListByCond(ctx, cond)
}

func (d *applicationScopeDAO) GetSecretByID(ctx context.Context, id uint) (*model.ApplicationSecretEntity, error) {
	return dao.NewApplicationSecretDao().GetByID(ctx, id)
}

func (d *applicationScopeDAO) DeleteSecret(ctx context.Context, id uint, userID uint) error {
	return dao.NewApplicationSecretDao().Delete(ctx, id, userID)
}

func applicationVisibleToTenant(entity *model.ApplicationEntity, tenantID uint) bool {
	return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}

var _ ApplicationSvc = (*applicationSvc)(nil)

func NewApplicationSvc() ApplicationSvc {
	return &applicationSvc{}
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

func (svc *applicationSvc) Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error) {
	insertEntity := &model.ApplicationEntity{
		TenantID:                gincontext.GetTenantID(ctx),
		ClientID:                generateClientID(),
		Name:                    req.Name,
		Description:             req.Description,
		LogoURL:                 req.LogoURL,
		HomepageURL:             req.HomepageURL,
		Type:                    req.Type,
		IsThirdParty:            req.IsThirdParty,
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
		CreatedBy:               gincontext.GetUserID(ctx),
	}

	if err := dao.NewApplicationDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationCreateError)
	}
	return &dtoapplication.ApplicationCreateResp{
		ApplicationID: insertEntity.ID,
		ClientID:      insertEntity.ClientID,
	}, nil
}

func (svc *applicationSvc) Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error {
	appEntity, err := newApplicationScopeRepo().GetByID(ctx, req.ApplicationID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	if !applicationVisibleToTenant(appEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.ApplicationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewApplicationDao().Delete(ctx, req.ApplicationID, userID); err != nil {
		glog.Errorf(ctx, "[svcapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationDeleteError)
	}
	return nil
}

func (svc *applicationSvc) Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error {
	appEntity, err := newApplicationScopeRepo().GetByID(ctx, req.ApplicationID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	if !applicationVisibleToTenant(appEntity, gincontext.GetTenantID(ctx)) {
		return code.GetError(code.ApplicationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"name":                       req.Name,
		"description":                req.Description,
		"logo_url":                   req.LogoURL,
		"homepage_url":               req.HomepageURL,
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
	if err := dao.NewApplicationDao().UpdateMap(ctx, req.ApplicationID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ApplicationUpdateError)
	}
	return nil
}

func (svc *applicationSvc) Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error) {
	appEntity, err := newApplicationScopeRepo().GetByID(ctx, req.ApplicationID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetDetailError)
	}
	if !applicationVisibleToTenant(appEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.ApplicationNotExistError)
	}

	return &dtoapplication.ApplicationDetailResp{
		ApplicationID: appEntity.ID,
		ApplicationBaseInfo: objapplication.ApplicationBaseInfo{
			TenantID:     appEntity.TenantID,
			ClientID:     appEntity.ClientID,
			Name:         appEntity.Name,
			Description:  appEntity.Description,
			LogoURL:      appEntity.LogoURL,
			HomepageURL:  appEntity.HomepageURL,
			Type:         appEntity.Type,
			Status:       appEntity.Status,
			IsThirdParty: appEntity.IsThirdParty,
		},
	}, nil
}

func (svc *applicationSvc) PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error) {
	appRepo := newApplicationScopeRepo()
	cond := &dao.ApplicationCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: gincontext.GetTenantID(ctx),
		Name:     req.Name,
		Type:     req.Type,
		Status:   req.Status,
	}
	appEntityList, total, err := appRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationGetPageListError)
	}

	list := make([]dtoapplication.ApplicationPageListItem, 0, len(appEntityList))
	for _, v := range appEntityList {
		list = append(list, dtoapplication.ApplicationPageListItem{
			ApplicationID: v.ID,
			ApplicationBaseInfo: objapplication.ApplicationBaseInfo{
				TenantID:     v.TenantID,
				ClientID:     v.ClientID,
				Name:         v.Name,
				Description:  v.Description,
				LogoURL:      v.LogoURL,
				HomepageURL:  v.HomepageURL,
				Type:         v.Type,
				Status:       v.Status,
				IsThirdParty: v.IsThirdParty,
			},
		})
	}
	return &dtoapplication.ApplicationPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *applicationSvc) GetByClientID(ctx *gin.Context, clientID string) (*dtoapplication.ApplicationDetailResp, error) {
	appRepo := newApplicationScopeRepo()
	appEntity, err := appRepo.GetByCond(ctx, &dao.ApplicationCond{
		ClientID: clientID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.GetByClientID] dao GetByCond fail, err:%v, clientID:%s", err, clientID)
		return nil, code.GetError(code.ApplicationGetDetailError)
	}
	if appEntity == nil || appEntity.ID == 0 {
		return nil, code.GetError(code.ApplicationNotExistError)
	}
	return &dtoapplication.ApplicationDetailResp{
		ApplicationID: appEntity.ID,
		ApplicationBaseInfo: objapplication.ApplicationBaseInfo{
			TenantID:     appEntity.TenantID,
			ClientID:     appEntity.ClientID,
			Name:         appEntity.Name,
			Description:  appEntity.Description,
			LogoURL:      appEntity.LogoURL,
			HomepageURL:  appEntity.HomepageURL,
			Type:         appEntity.Type,
			Status:       appEntity.Status,
			IsThirdParty: appEntity.IsThirdParty,
		},
	}, nil
}

func (svc *applicationSvc) ListRoles(ctx *gin.Context, req *dtoapplication.ApplicationRoleListReq) (*dtoapplication.ApplicationRoleListResp, error) {
	appRoleDao := newApplicationRoleListReader()
	roleDao := newRoleReader()

	list, err := appRoleDao.GetListByCond(ctx, &dao.ApplicationRoleCond{
		ApplicationID: req.ApplicationID,
	})
	if err != nil {
		glog.Errorf(ctx, "[applicationSvc.ListRoles] get roles fail, err:%v", err)
		return nil, code.GetError(code.RoleApplicationGetListError)
	}

	roleMap := make(map[uint]*model.RoleEntity, len(list))
	for _, item := range list {
		if role, err := roleDao.GetByID(ctx, item.RoleID); err == nil && role != nil {
			roleMap[role.ID] = role
		}
	}

	roles := make([]dtoapplication.ApplicationRoleResp, 0, len(list))
	for _, item := range list {
		if role, ok := roleMap[item.RoleID]; ok {
			roles = append(roles, dtoapplication.ApplicationRoleResp{
				RoleID:        uint64(item.RoleID),
				RoleName:      role.Name,
				RoleCode:      role.Code,
				ApplicationID: uint64(item.ApplicationID),
				CreatedAt:     item.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	return &dtoapplication.ApplicationRoleListResp{
		Total: int64(len(roles)),
		Roles: roles,
	}, nil
}

func (svc *applicationSvc) AssignRoles(ctx *gin.Context, req *dtoapplication.AssignApplicationRolesReq) error {
	appRoleDao := dao.NewApplicationRoleDao()
	userID := gincontext.GetUserID(ctx)

	for _, roleID := range req.RoleIDs {
		existing, _ := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
			ApplicationID: uint(req.ApplicationID),
			RoleID:        uint(roleID),
		})
		if existing != nil && existing.ID != 0 {
			continue
		}

		entity := &model.ApplicationRoleEntity{
			TenantID:      gincontext.GetTenantID(ctx),
			ApplicationID: uint(req.ApplicationID),
			RoleID:        uint(roleID),
			CreatedBy:     userID,
		}
		if err := appRoleDao.Insert(ctx, entity); err != nil {
			glog.Errorf(ctx, "[applicationSvc.AssignRoles] insert fail, err:%v", err)
			return code.GetError(code.RoleApplicationCreateError)
		}
	}

	return nil
}

func (svc *applicationSvc) RemoveRole(ctx *gin.Context, req *dtoapplication.RemoveApplicationRoleReq) error {
	appRoleDao := dao.NewApplicationRoleDao()

	entity, err := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
		ApplicationID: uint(req.ApplicationID),
		RoleID:        uint(req.RoleID),
	})
	if err != nil || entity == nil || entity.ID == 0 {
		return code.GetError(code.RoleApplicationNotExistError)
	}

	if err := appRoleDao.Delete(ctx, entity.ID, gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[applicationSvc.RemoveRole] delete fail, err:%v", err)
		return code.GetError(code.RoleApplicationDeleteError)
	}

	return nil
}

func (svc *applicationSvc) ListSecrets(ctx *gin.Context, req *dtoapplication.ApplicationSecretListReq) (*dtoapplication.ApplicationSecretListResp, error) {
	secretDao := dao.NewApplicationSecretDao()

	list, total, err := secretDao.GetPageListByCond(ctx, &dao.ApplicationSecretCond{
		BaseCond:      &genericdao.BaseCond{Page: 1, PageSize: 100},
		ApplicationID: req.ApplicationID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.ListSecrets] get secrets fail, err:%v", err)
		return nil, code.GetError(code.ApplicationSecretGetListError)
	}

	secrets := make([]dtoapplication.ApplicationSecretResp, 0, len(list))
	for _, s := range list {
		var expiresAt *string
		if s.ExpiredAt != nil {
			t := s.ExpiredAt.Format("2006-01-02 15:04:05")
			expiresAt = &t
		}
		secrets = append(secrets, dtoapplication.ApplicationSecretResp{
			ID:            uint64(s.ID),
			ApplicationID: uint64(s.ApplicationID),
			Name:          s.Name,
			ValuePrefix:   s.ValuePrefix,
			ExpiredAt:     expiresAt,
			CreatedAt:     s.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &dtoapplication.ApplicationSecretListResp{
		Total:   total,
		Secrets: secrets,
	}, nil
}

func (svc *applicationSvc) CreateSecret(ctx *gin.Context, req *dtoapplication.CreateApplicationSecretReq) (*dtoapplication.CreateApplicationSecretResp, error) {
	appEntity, err := newApplicationScopeRepo().GetByID(ctx, req.ApplicationID)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.CreateSecret] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApplicationSecretCreateError)
	}
	if !applicationVisibleToTenant(appEntity, gincontext.GetTenantID(ctx)) {
		return nil, code.GetError(code.ApplicationNotExistError)
	}

	randomBytes, err := gcrypto.GenerateRandomBytes(32)
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.CreateSecret] generate secret fail, err:%v", err)
		return nil, code.GetError(code.ApplicationSecretCreateError)
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

	entity := &model.ApplicationSecretEntity{
		ApplicationID: req.ApplicationID,
		Name:          req.Name,
		ValueHash:     valueHash,
		ValuePrefix:   valuePrefix,
		ExpiredAt:     expiresAt,
		CreatedBy:     gincontext.GetUserID(ctx),
	}

	if err := dao.NewApplicationSecretDao().Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcapplication.CreateSecret] insert fail, err:%v", err)
		return nil, code.GetError(code.ApplicationSecretCreateError)
	}

	return &dtoapplication.CreateApplicationSecretResp{
		ID:          uint64(entity.ID),
		Name:        entity.Name,
		ValuePrefix: entity.ValuePrefix,
		Secret:      secretValue,
	}, nil
}

func (svc *applicationSvc) DeleteSecret(ctx *gin.Context, req *dtoapplication.DeleteApplicationSecretReq) error {
	appRepo := newApplicationScopeRepo()

	entity, err := appRepo.GetSecretByID(ctx, uint(req.SecretID))
	if err != nil {
		glog.Errorf(ctx, "[svcapplication.DeleteSecret] get secret fail, err:%v", err)
		return code.GetError(code.ApplicationSecretDeleteError)
	}
	if entity == nil || entity.ID == 0 {
		return code.GetError(code.ApplicationSecretNotExistError)
	}

	if err := appRepo.DeleteSecret(ctx, uint(req.SecretID), gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[svcapplication.DeleteSecret] delete fail, err:%v", err)
		return code.GetError(code.ApplicationSecretDeleteError)
	}

	return nil
}
