package svcapikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/audit"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapikey"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
)

type CreateApiKeySvc interface {
	Create(ctx *gin.Context, req *dtoapikey.ApiKeyCreateReq) (*dtoapikey.ApiKeyCreateResp, error)
	Revoke(ctx *gin.Context, req *dtoapikey.RevokeApiKeyReq) error
	Delete(ctx *gin.Context, req *dtoapikey.ApiKeyDeleteReq) error
	PageList(ctx *gin.Context, req *dtoapikey.ApiKeyPageListReq) (*dtoapikey.ApiKeyPageListResp, error)
	PageListSupervision(ctx *gin.Context, req *dtoapikey.ApiKeySupervisionPageListReq) (*dtoapikey.ApiKeySupervisionPageListResp, error)
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

func (svc *createApiKeySvc) Create(ctx *gin.Context, req *dtoapikey.ApiKeyCreateReq) (*dtoapikey.ApiKeyCreateResp, error) {
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
	var expiresAtUnix int64
	if req.ExpiredAt > 0 {
		t := time.Unix(req.ExpiredAt, 0)
		expiresAt = &t
		expiresAtUnix = req.ExpiredAt
	}

	entity := &model.ApiKeyEntity{
		TenantID:  gincontext.GetTenantIDString(ctx),
		Name:      req.Name,
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		Scope:     scope,
		ExpiredAt: expiresAt,
		CreatedBy: gincontext.GetUserIDString(ctx),
	}

	if err := svc.apiKeyDao.Insert(context.Background(), entity); err != nil {
		glog.Errorf(ctx, "[svcapikey.Create] dao Insert fail, err:%v", err)
		return nil, err
	}

	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionApiKeyCreate,
		TenantID:   gincontext.GetTenantIDString(ctx),
		Result:     "success",
		TargetType: "api_key",
		TargetID:   entity.ID,
	})

	return &dtoapikey.ApiKeyCreateResp{
		ID:        entity.ID,
		Name:      entity.Name,
		Key:       rawKey,
		KeyPrefix: keyPrefix,
		ExpiredAt: expiresAtUnix,
	}, nil
}

func (svc *createApiKeySvc) Revoke(ctx *gin.Context, req *dtoapikey.RevokeApiKeyReq) error {
	if err := svc.apiKeyDao.UpdateMap(context.Background(), req.ApiKeyID, map[string]any{"revoked_at": time.Now()}); err != nil {
		glog.Errorf(ctx, "[svcapikey.Revoke] dao UpdateMap fail, err:%v, id:%s", err, req.ApiKeyID)
		return err
	}
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionApiKeyRevoke,
		TenantID:   gincontext.GetTenantIDString(ctx),
		Result:     "success",
		TargetType: "api_key",
		TargetID:   req.ApiKeyID,
	})
	return nil
}

func (svc *createApiKeySvc) Delete(ctx *gin.Context, req *dtoapikey.ApiKeyDeleteReq) error {
	if err := svc.apiKeyDao.Delete(context.Background(), req.ApiKeyID, ""); err != nil {
		glog.Errorf(ctx, "[svcapikey.Delete] dao Delete fail, err:%v, id:%s", err, req.ApiKeyID)
		return err
	}
	return nil
}

func (svc *createApiKeySvc) PageList(ctx *gin.Context, req *dtoapikey.ApiKeyPageListReq) (*dtoapikey.ApiKeyPageListResp, error) {
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
		TenantID: gincontext.GetTenantIDString(ctx),
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
			CreatedAt: entity.CreatedAt.Unix(),
		}
		if entity.ExpiredAt != nil {
			item.ExpiredAt = entity.ExpiredAt.Unix()
		}
		if entity.LastUsedAt.Valid {
			item.LastUsedAt = entity.LastUsedAt.Time.Unix()
		}
		if entity.RevokedAt != nil {
			item.RevokedAt = entity.RevokedAt.Unix()
		}
		items = append(items, item)
	}

	return &dtoapikey.ApiKeyPageListResp{
		List:  items,
		Total: total,
	}, nil
}

// PageListSupervision 平台排查视角：跨租户只读检索全部 API Key（忽略当前上下文租户），
// 用于泄漏风险监督（僵尸 key/长期未用/超长有效期等由前端结合字段呈现）。
// 明文密钥永不可见（仅前缀）；本接口不提供吊销/删除等写动作（平台级应急吊销暂未纳入）。
func (svc *createApiKeySvc) PageListSupervision(ctx *gin.Context, req *dtoapikey.ApiKeySupervisionPageListReq) (*dtoapikey.ApiKeySupervisionPageListResp, error) {
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
		TenantID: req.TenantID,
		Name:     req.Name,
	}
	list, total, err := svc.apiKeyDao.GetPageListByCond(context.Background(), cond)
	if err != nil {
		glog.Errorf(ctx, "[svcapikey.PageListSupervision] dao GetPageListByCond fail, err:%v", err)
		return nil, err
	}

	tenantIDList, creatorIDList := collectSupervisionOwnerIDs(list)
	tenantNameMap := loadTenantNameMap(ctx, tenantIDList)
	creatorNameMap := loadCreatorNameMap(ctx, creatorIDList)

	items := make([]dtoapikey.ApiKeySupervisionItem, 0, len(list))
	for _, entity := range list {
		item := dtoapikey.ApiKeySupervisionItem{
			ID:          entity.ID,
			TenantID:    entity.TenantID,
			TenantName:  tenantNameMap[entity.TenantID],
			CreatedBy:   entity.CreatedBy,
			CreatorName: creatorNameMap[entity.CreatedBy],
			Name:        entity.Name,
			KeyPrefix:   entity.KeyPrefix,
			Scope:       string(entity.Scope),
			CreatedAt:   entity.CreatedAt.Unix(),
		}
		if entity.ExpiredAt != nil {
			item.ExpiredAt = entity.ExpiredAt.Unix()
		}
		if entity.LastUsedAt.Valid {
			item.LastUsedAt = entity.LastUsedAt.Time.Unix()
		}
		if entity.RevokedAt != nil {
			item.RevokedAt = entity.RevokedAt.Unix()
		}
		items = append(items, item)
	}

	return &dtoapikey.ApiKeySupervisionPageListResp{
		List:  items,
		Total: total,
	}, nil
}

// collectSupervisionOwnerIDs 去重收集列表内出现的租户 ID 与创建人（user）ID。
func collectSupervisionOwnerIDs(list model.ApiKeyEntityList) (tenantIDList, creatorIDList []string) {
	tenantSeen := make(map[string]struct{}, len(list))
	creatorSeen := make(map[string]struct{}, len(list))
	for _, entity := range list {
		if entity.TenantID != "" {
			if _, ok := tenantSeen[entity.TenantID]; !ok {
				tenantSeen[entity.TenantID] = struct{}{}
				tenantIDList = append(tenantIDList, entity.TenantID)
			}
		}
		if entity.CreatedBy != "" {
			if _, ok := creatorSeen[entity.CreatedBy]; !ok {
				creatorSeen[entity.CreatedBy] = struct{}{}
				creatorIDList = append(creatorIDList, entity.CreatedBy)
			}
		}
	}
	return tenantIDList, creatorIDList
}

// loadTenantNameMap 按 ID 加载租户名；单个查询失败仅告警并置空，不中断列表返回。
func loadTenantNameMap(ctx *gin.Context, tenantIDs []string) map[string]string {
	out := make(map[string]string, len(tenantIDs))
	for _, id := range tenantIDs {
		tenant, err := dao.NewTenantDao().GetByID(ctx, id)
		if err != nil || tenant == nil {
			glog.Warnf(ctx, "[svcapikey.PageListSupervision] tenant GetByID fail, tenantID:%s, err:%v", id, err)
			out[id] = ""
			continue
		}
		out[id] = tenant.Name
	}
	return out
}

// loadCreatorNameMap 按 ID 批量加载创建人（租户 user）名；失败仅告警并置空。
func loadCreatorNameMap(ctx *gin.Context, creatorIDs []string) map[string]string {
	out := make(map[string]string, len(creatorIDs))
	if len(creatorIDs) == 0 {
		return out
	}
	userList, _, err := dao.NewUserDao().GetPageListByCond(ctx, &dao.UserCond{
		BaseCond: &gormdao.BaseCond{Page: 1, PageSize: len(creatorIDs)},
		IDs:      creatorIDs,
	})
	if err != nil {
		glog.Warnf(ctx, "[svcapikey.PageListSupervision] user GetPageListByCond fail, err:%v", err)
		return out
	}
	for _, user := range userList {
		out[user.ID] = user.Name
	}
	return out
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
