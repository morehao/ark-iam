package svcapikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoapikey"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/glog"
)

type CreateApiKeySvc interface {
	Create(ctx *gin.Context, tenantID uint, req *dtoapikey.CreateApiKeyReq) (*dtoapikey.CreateApiKeyResp, error)
	Revoke(ctx *gin.Context, tenantID uint, req *dtoapikey.RevokeApiKeyReq) error
	Delete(ctx *gin.Context, tenantID uint, req *dtoapikey.DeleteApiKeyReq) error
	PageList(ctx *gin.Context, tenantID uint, req *dtoapikey.ApiKeyPageListReq) (*dtoapikey.ApiKeyPageListResp, error)
}

type createApiKeySvc struct {
	apiKeyDao *dao.ApiKeyDao
}

var _ CreateApiKeySvc = (*createApiKeySvc)(nil)

func NewCreateApiKeySvc() CreateApiKeySvc {
	return &createApiKeySvc{
		apiKeyDao: dao.NewApiKeyDao(),
	}
}

func newCreateApiKeySvcWithDao(apiKeyDao *dao.ApiKeyDao) CreateApiKeySvc {
	return &createApiKeySvc{
		apiKeyDao: apiKeyDao,
	}
}

func (svc *createApiKeySvc) Create(ctx *gin.Context, tenantID uint, req *dtoapikey.CreateApiKeyReq) (*dtoapikey.CreateApiKeyResp, error) {
	rawKey, err := generateApiKey()
	if err != nil {
		glog.Errorf(ctx, "[svcapikey.Create] generateApiKey fail, err:%v", err)
		return nil, err
	}

	keyHash := hashApiKey(rawKey)
	keyPrefix := rawKey[:7]

	var scope json.RawMessage
	if req.Scope != "" {
		scope = json.RawMessage(req.Scope)
	} else {
		scope = json.RawMessage("{}")
	}

	var expiresAt *time.Time
	var expiresAtStr string
	if req.ExpiredAt != "" {
		t, err := time.Parse("2006-01-02T15:04:05Z", req.ExpiredAt)
		if err != nil {
			glog.Errorf(ctx, "[svcapikey.Create] parse expiresAt fail, err:%v", err)
			return nil, err
		}
		expiresAt = &t
		expiresAtStr = t.Format("2006-01-02T15:04:05Z")
	}

	entity := &model.ApiKeyEntity{
		TenantID:  tenantID,
		Name:      req.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Scope:     scope,
		ExpiredAt: expiresAt,
		CreatedBy: 0,
	}

	if err := svc.apiKeyDao.Insert(context.Background(), entity); err != nil {
		glog.Errorf(ctx, "[svcapikey.Create] dao Insert fail, err:%v", err)
		return nil, err
	}

	return &dtoapikey.CreateApiKeyResp{
		ID:        entity.ID,
		Name:      entity.Name,
		Key:       rawKey,
		KeyPrefix: keyPrefix,
		ExpiredAt: expiresAtStr,
	}, nil
}

func (svc *createApiKeySvc) Revoke(ctx *gin.Context, tenantID uint, req *dtoapikey.RevokeApiKeyReq) error {
	if err := svc.apiKeyDao.UpdateMap(context.Background(), req.ID, map[string]any{"revoked_at": time.Now()}); err != nil {
		glog.Errorf(ctx, "[svcapikey.Revoke] dao UpdateMap fail, err:%v, id:%d", err, req.ID)
		return err
	}
	return nil
}

func (svc *createApiKeySvc) Delete(ctx *gin.Context, tenantID uint, req *dtoapikey.DeleteApiKeyReq) error {
	if err := svc.apiKeyDao.Delete(context.Background(), req.ID, 0); err != nil {
		glog.Errorf(ctx, "[svcapikey.Delete] dao Delete fail, err:%v, id:%d", err, req.ID)
		return err
	}
	return nil
}

func (svc *createApiKeySvc) PageList(ctx *gin.Context, tenantID uint, req *dtoapikey.ApiKeyPageListReq) (*dtoapikey.ApiKeyPageListResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	cond := &dao.ApiKeyCond{
		BaseCond: &genericdao.BaseCond{Page: page, PageSize: pageSize},
		TenantID: tenantID,
		Name:     req.Name,
	}

	list, total, err := svc.apiKeyDao.GetPageListByCond(context.Background(), cond)
	if err != nil {
		glog.Errorf(ctx, "[svcapikey.PageList] dao GetPageListByCond fail, err:%v", err)
		return nil, err
	}

	items := make([]dtoapikey.ApiKeyPageListItem, 0, len(list))
	for _, entity := range list {
		item := dtoapikey.ApiKeyPageListItem{
			ID:        entity.ID,
			Name:      entity.Name,
			KeyPrefix: entity.KeyPrefix,
			Scope:     string(entity.Scope),
			CreatedAt: entity.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if entity.ExpiredAt != nil {
			item.ExpiredAt = entity.ExpiredAt.Format("2006-01-02T15:04:05Z")
		}
		if entity.LastUsedAt.Valid {
			item.LastUsedAt = entity.LastUsedAt.Time.Format("2006-01-02T15:04:05Z")
		}
		if entity.RevokedAt != nil {
			item.RevokedAt = entity.RevokedAt.Format("2006-01-02T15:04:05Z")
		}
		items = append(items, item)
	}

	return &dtoapikey.ApiKeyPageListResp{
		List:  items,
		Total: total,
	}, nil
}

func generateApiKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashApiKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
