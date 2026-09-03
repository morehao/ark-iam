package svctenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/apikey"
	"github.com/morehao/ark-iam/pkg/iam/audit"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

// ApiKeySvc 租户端 API 密钥领域服务。
// 密钥一律归属服务账号（开发者模式，面向服务端集成）；创建/列表/吊销/删除为系统管理操作
// （服务账号不可登录，不存在"本人自管"通道）。鉴权时按归属服务账号注入身份上下文。
type ApiKeySvc interface {
	Create(ctx *gin.Context, req *dtotenant.ApiKeyCreateReq) (*dtotenant.ApiKeyCreateResp, error)
	PageList(ctx *gin.Context, req *dtotenant.ApiKeyPageListReq) (*dtotenant.ApiKeyPageListResp, error)
	Revoke(ctx *gin.Context, req *dtotenant.ApiKeyRevokeReq) error
	Delete(ctx *gin.Context, req *dtotenant.ApiKeyDeleteReq) error
}

type apiKeySvc struct{}

var _ ApiKeySvc = (*apiKeySvc)(nil)

func NewApiKeySvc() ApiKeySvc {
	return &apiKeySvc{}
}

// requireSystemAdmin 校验当前操作者具备系统管理能力，opErr 用于系统错误的兜底返回。
func (svc *apiKeySvc) requireSystemAdmin(ctx *gin.Context, opErr int) error {
	ok, err := HasSystemAdminCapability(ctx)
	if err != nil {
		glog.Errorf(ctx, "[svcapikey.requireSystemAdmin] resolve admin level fail, err:%v", err)
		return code.GetError(opErr)
	}
	if !ok {
		return code.GetError(code.UserSystemAdminRequiredError)
	}
	return nil
}

// loadOwnerServiceAccount 加载指定租户内作为密钥归属的服务账号（不存在/跨租户/非服务账号均按不存在处理）。
func (svc *apiKeySvc) loadOwnerServiceAccount(ctx *gin.Context, tenantID, machineUserID string) (*model.UserEntity, error) {
	entity, err := dao.NewUserDao().GetByID(ctx, machineUserID)
	if err != nil {
		glog.Errorf(ctx, "[svcapikey.loadOwnerServiceAccount] dao GetByID fail, err:%v, id:%s", err, machineUserID)
		return nil, code.GetError(code.ApiKeyOwnerNotExistError)
	}
	if entity == nil || entity.TenantID != tenantID || !entity.IsMachine() {
		return nil, code.GetError(code.ApiKeyOwnerNotExistError)
	}
	return entity, nil
}

func (svc *apiKeySvc) Create(ctx *gin.Context, req *dtotenant.ApiKeyCreateReq) (*dtotenant.ApiKeyCreateResp, error) {
	if err := svc.requireSystemAdmin(ctx, code.ApiKeyCreateError); err != nil {
		return nil, err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	operatorID := gincontext.GetUserIDString(ctx)

	// 密钥归属服务账号（DTO 已校验 machineUserID 必填）
	owner, err := svc.loadOwnerServiceAccount(ctx, tenantID, req.MachineUserID)
	if err != nil {
		return nil, err
	}

	rawKey, err := apikey.Generate()
	if err != nil {
		glog.Errorf(ctx, "[svcapikey.Create] apikey.Generate fail, err:%v", err)
		return nil, code.GetError(code.ApiKeyCreateError)
	}
	var expiresAt *time.Time
	var expiresAtUnix int64
	if req.ExpiredAt > 0 {
		t := time.Unix(req.ExpiredAt, 0)
		expiresAt = &t
		expiresAtUnix = req.ExpiredAt
	}
	entity := &model.ApiKeyEntity{
		TenantID:    tenantID,
		OwnerUserID: owner.ID,
		Name:        req.Name,
		KeyHash:     apikey.Hash(rawKey),
		KeyPrefix:   apikey.Prefix(rawKey),
		Scope:       json.RawMessage(`{}`),
		ExpiredAt:   expiresAt,
		CreatedBy:   operatorID,
	}
	if err := dao.NewApiKeyDao().Insert(context.Background(), entity); err != nil {
		glog.Errorf(ctx, "[svcapikey.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApiKeyCreateError)
	}
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionApiKeyCreate,
		TenantID:   tenantID,
		Result:     "success",
		TargetType: "api_key",
		TargetID:   entity.ID,
	})
	return &dtotenant.ApiKeyCreateResp{
		ID:        entity.ID,
		Name:      entity.Name,
		Key:       rawKey,
		KeyPrefix: entity.KeyPrefix,
		ExpiredAt: expiresAtUnix,
	}, nil
}

// PageList 服务账号密钥分页（管理视角，需系统管理能力）：
// machineUserID 为空 = 租户全部密钥；指定 = 该服务账号名下密钥。
func (svc *apiKeySvc) PageList(ctx *gin.Context, req *dtotenant.ApiKeyPageListReq) (*dtotenant.ApiKeyPageListResp, error) {
	if err := svc.requireSystemAdmin(ctx, code.ApiKeyGetPageListError); err != nil {
		return nil, err
	}
	tenantID := gincontext.GetTenantIDString(ctx)

	cond := &dao.ApiKeyCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: tenantID,
		Name:     req.Name,
	}
	if req.MachineUserID != "" {
		if _, err := svc.loadOwnerServiceAccount(ctx, tenantID, req.MachineUserID); err != nil {
			return nil, err
		}
		cond.OwnerUserID = req.MachineUserID
	}

	list, total, err := dao.NewApiKeyDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcapikey.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ApiKeyGetPageListError)
	}
	userMap, err := svc.loadUserMap(ctx, tenantID, list)
	if err != nil {
		return nil, err
	}
	respList := make([]dtotenant.ApiKeyPageListItem, 0, len(list))
	for _, v := range list {
		item := dtotenant.ApiKeyPageListItem{
			KeyID:       v.ID,
			Name:        v.Name,
			KeyPrefix:   v.KeyPrefix,
			OwnerUserID: v.OwnerUserID,
			ExpiredAt:   timePtrUnix(v.ExpiredAt),
			LastUsedAt:  sqlTimePtrUnix(v.LastUsedAt),
			RevokedAt:   timePtrUnix(v.RevokedAt),
			CreatedAt:   v.CreatedAt.Unix(),
		}
		if owner, ok := userMap[v.OwnerUserID]; ok {
			item.OwnerType = string(owner.UserType)
			item.OwnerName = owner.Name
		}
		if creator, ok := userMap[v.CreatedBy]; ok {
			item.CreatedBy = v.CreatedBy
			item.CreatorName = creator.Name
		}
		respList = append(respList, item)
	}
	return &dtotenant.ApiKeyPageListResp{List: respList, Total: total}, nil
}

func (svc *apiKeySvc) Revoke(ctx *gin.Context, req *dtotenant.ApiKeyRevokeReq) error {
	tenantID := gincontext.GetTenantIDString(ctx)
	key, err := svc.loadKey(ctx, tenantID, req.ApiKeyID)
	if err != nil {
		return err
	}
	if err := svc.requireSystemAdmin(ctx, code.ApiKeyRevokeError); err != nil {
		return err
	}
	if err := dao.NewApiKeyDao().UpdateMap(ctx, key.ID, map[string]any{"revoked_at": time.Now()}); err != nil {
		glog.Errorf(ctx, "[svcapikey.Revoke] dao UpdateMap fail, err:%v, id:%s", err, key.ID)
		return code.GetError(code.ApiKeyRevokeError)
	}
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionApiKeyRevoke,
		TenantID:   tenantID,
		Result:     "success",
		TargetType: "api_key",
		TargetID:   key.ID,
	})
	return nil
}

func (svc *apiKeySvc) Delete(ctx *gin.Context, req *dtotenant.ApiKeyDeleteReq) error {
	tenantID := gincontext.GetTenantIDString(ctx)
	key, err := svc.loadKey(ctx, tenantID, req.ApiKeyID)
	if err != nil {
		return err
	}
	if err := svc.requireSystemAdmin(ctx, code.ApiKeyDeleteError); err != nil {
		return err
	}
	if err := dao.NewApiKeyDao().Delete(context.Background(), key.ID, gincontext.GetUserIDString(ctx)); err != nil {
		glog.Errorf(ctx, "[svcapikey.Delete] dao Delete fail, err:%v, id:%s", err, key.ID)
		return code.GetError(code.ApiKeyDeleteError)
	}
	return nil
}

func (svc *apiKeySvc) loadKey(ctx *gin.Context, tenantID, keyID string) (*model.ApiKeyEntity, error) {
	entity, err := dao.NewApiKeyDao().GetByID(ctx, keyID)
	if err != nil {
		glog.Errorf(ctx, "[svcapikey.loadKey] dao GetByID fail, err:%v, id:%s", err, keyID)
		return nil, code.GetError(code.ApiKeyRevokeError)
	}
	if entity == nil || entity.TenantID != tenantID {
		return nil, code.GetError(code.ApiKeyNotExistError)
	}
	return entity, nil
}

// loadUserMap 汇总某批密钥中的归属服务账号与创建人，批量加载租户内用户。
func (svc *apiKeySvc) loadUserMap(ctx *gin.Context, tenantID string, list model.ApiKeyEntityList) (map[string]*model.UserEntity, error) {
	idSet := make(map[string]struct{})
	for _, v := range list {
		if v.OwnerUserID != "" {
			idSet[v.OwnerUserID] = struct{}{}
		}
		if v.CreatedBy != "" {
			idSet[v.CreatedBy] = struct{}{}
		}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return map[string]*model.UserEntity{}, nil
	}
	userList, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{TenantID: tenantID, IDs: ids})
	if err != nil {
		glog.Errorf(ctx, "[svcapikey.loadUserMap] dao user GetListByCond fail, err:%v", err)
		return nil, code.GetError(code.ApiKeyGetPageListError)
	}
	m := make(map[string]*model.UserEntity, len(userList))
	for i := range userList {
		m[userList[i].ID] = &userList[i]
	}
	return m, nil
}

// timePtrUnix *time.Time → *int64(unix秒)。
func timePtrUnix(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	v := t.Unix()
	return &v
}

// sqlTimePtrUnix sql.NullTime → *int64(unix秒,无效为空)。
func sqlTimePtrUnix(t sql.NullTime) *int64 {
	if !t.Valid {
		return nil
	}
	v := t.Time.Unix()
	return &v
}
