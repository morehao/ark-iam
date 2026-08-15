package svcapikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/svcaudit"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapikey"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
)

type CreateApiKeySvc interface {
	Create(ctx *gin.Context, tenantID string, req *dtoapikey.ApiKeyCreateReq) (*dtoapikey.ApiKeyCreateResp, error)
	Revoke(ctx *gin.Context, tenantID string, req *dtoapikey.RevokeApiKeyReq) error
	Delete(ctx *gin.Context, tenantID string, req *dtoapikey.ApiKeyDeleteReq) error
	PageList(ctx *gin.Context, tenantID string, req *dtoapikey.ApiKeyPageListReq) (*dtoapikey.ApiKeyPageListResp, error)
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

func (svc *createApiKeySvc) Create(ctx *gin.Context, tenantID string, req *dtoapikey.ApiKeyCreateReq) (*dtoapikey.ApiKeyCreateResp, error) {
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
		CreatedBy: gincontext.GetUserID(ctx),
	}

	if err := svc.apiKeyDao.Insert(context.Background(), entity); err != nil {
		glog.Errorf(ctx, "[svcapikey.Create] dao Insert fail, err:%v", err)
		return nil, err
	}

	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionApiKeyCreate,
		TenantID:   tenantID,
		Result:     "success",
		TargetType: "api_key",
		TargetID:   entity.ID,
	})

	return &dtoapikey.ApiKeyCreateResp{
		ID:        entity.ID,
		Name:      entity.Name,
		Key:       rawKey,
		KeyPrefix: keyPrefix,
		ExpiredAt: expiresAtStr,
	}, nil
}

func (svc *createApiKeySvc) Revoke(ctx *gin.Context, tenantID string, req *dtoapikey.RevokeApiKeyReq) error {
	if err := svc.apiKeyDao.UpdateMap(context.Background(), req.ApiKeyID, map[string]any{"revoked_at": time.Now()}); err != nil {
		glog.Errorf(ctx, "[svcapikey.Revoke] dao UpdateMap fail, err:%v, id:%s", err, req.ApiKeyID)
		return err
	}
	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionApiKeyRevoke,
		TenantID:   tenantID,
		Result:     "success",
		TargetType: "api_key",
		TargetID:   req.ApiKeyID,
	})
	return nil
}

func (svc *createApiKeySvc) Delete(ctx *gin.Context, tenantID string, req *dtoapikey.ApiKeyDeleteReq) error {
	if err := svc.apiKeyDao.Delete(context.Background(), req.ApiKeyID, ""); err != nil {
		glog.Errorf(ctx, "[svcapikey.Delete] dao Delete fail, err:%v, id:%s", err, req.ApiKeyID)
		return err
	}
	return nil
}

func (svc *createApiKeySvc) PageList(ctx *gin.Context, tenantID string, req *dtoapikey.ApiKeyPageListReq) (*dtoapikey.ApiKeyPageListResp, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	cond := &dao.ApiKeyCond{
		BaseCond: &gormdao.BaseCond{Page: page, PageSize: pageSize},
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
