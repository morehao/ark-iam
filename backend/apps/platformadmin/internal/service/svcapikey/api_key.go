package svcapikey

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoapikey"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
)

// ApiKeySupervisionSvc 平台监督视角的 API Key 服务。
// 平台侧仅保留跨租户只读监督：密钥归属主体（真实用户本人/服务账号）、租户与创建人信息用于泄漏风险排查；
// 创建/吊销/删除等写动作归属租户自服务控制台（/v1/tenant/*）。
type ApiKeySupervisionSvc interface {
	PageListSupervision(ctx *gin.Context, req *dtoapikey.ApiKeySupervisionPageListReq) (*dtoapikey.ApiKeySupervisionPageListResp, error)
}

type apiKeySupervisionSvc struct {
	apiKeyDao *dao.ApiKeyDao
}

var _ ApiKeySupervisionSvc = (*apiKeySupervisionSvc)(nil)

func NewApiKeySupervisionSvc() ApiKeySupervisionSvc {
	return &apiKeySupervisionSvc{
		apiKeyDao: dao.NewApiKeyDao(),
	}
}

func newApiKeySupervisionSvcWithDao(apiKeyDao *dao.ApiKeyDao) ApiKeySupervisionSvc {
	return &apiKeySupervisionSvc{
		apiKeyDao: apiKeyDao,
	}
}

// PageListSupervision 平台排查视角：跨租户只读检索全部 API Key（忽略当前上下文租户）。
// 明文密钥永不可见（仅前缀）；本接口不提供吊销/删除等写动作（平台级应急吊销暂未纳入）。
func (svc *apiKeySupervisionSvc) PageListSupervision(ctx *gin.Context, req *dtoapikey.ApiKeySupervisionPageListReq) (*dtoapikey.ApiKeySupervisionPageListResp, error) {
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

	tenantIDs, userIDs := collectSupervisionRelationIDs(list)
	tenantNameMap := loadTenantNameMap(ctx, tenantIDs)
	userMap := loadUserNameMap(ctx, userIDs)

	items := make([]dtoapikey.ApiKeySupervisionItem, 0, len(list))
	for _, entity := range list {
		item := dtoapikey.ApiKeySupervisionItem{
			ID:         entity.ID,
			TenantID:   entity.TenantID,
			TenantName: tenantNameMap[entity.TenantID],
			Name:       entity.Name,
			KeyPrefix:  entity.KeyPrefix,
			Scope:      string(entity.Scope),
			CreatedAt:  entity.CreatedAt.Unix(),
		}
		if owner, ok := userMap[entity.OwnerUserID]; ok {
			item.OwnerUserID = owner.ID
			item.OwnerType = string(owner.UserType)
			item.OwnerName = owner.Name
		}
		if creator, ok := userMap[entity.CreatedBy]; ok {
			item.CreatedBy = creator.ID
			item.CreatorName = creator.Name
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

// collectSupervisionRelationIDs 去重收集列表内出现的租户 ID 与关联用户（归属者/创建人）ID。
func collectSupervisionRelationIDs(list model.ApiKeyEntityList) (tenantIDs, userIDs []string) {
	tenantSeen := make(map[string]struct{}, len(list))
	userSeen := make(map[string]struct{}, len(list))
	for _, entity := range list {
		if entity.TenantID != "" {
			if _, ok := tenantSeen[entity.TenantID]; !ok {
				tenantSeen[entity.TenantID] = struct{}{}
				tenantIDs = append(tenantIDs, entity.TenantID)
			}
		}
		for _, id := range []string{entity.OwnerUserID, entity.CreatedBy} {
			if id == "" {
				continue
			}
			if _, ok := userSeen[id]; !ok {
				userSeen[id] = struct{}{}
				userIDs = append(userIDs, id)
			}
		}
	}
	return tenantIDs, userIDs
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

// loadUserNameMap 按 ID 批量加载关联用户（租户 user，含服务账号）；失败仅告警并置空。
func loadUserNameMap(ctx *gin.Context, userIDs []string) map[string]*model.UserEntity {
	out := make(map[string]*model.UserEntity, len(userIDs))
	if len(userIDs) == 0 {
		return out
	}
	userList, _, err := dao.NewUserDao().GetPageListByCond(ctx, &dao.UserCond{
		BaseCond: &gormdao.BaseCond{Page: 1, PageSize: len(userIDs)},
		IDs:      userIDs,
	})
	if err != nil {
		glog.Warnf(ctx, "[svcapikey.PageListSupervision] user GetPageListByCond fail, err:%v", err)
		return out
	}
	for i := range userList {
		out[userList[i].ID] = &userList[i]
	}
	return out
}
