package svctenantapplication

import (
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenantapplication"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type TenantApplicationSvc interface {
	Create(ctx *gin.Context, req *dtotenantapplication.TenantApplicationCreateReq) (*dtotenantapplication.TenantApplicationCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenantapplication.TenantApplicationDeleteReq) error
	Update(ctx *gin.Context, req *dtotenantapplication.TenantApplicationUpdateReq) error
	Detail(ctx *gin.Context, req *dtotenantapplication.TenantApplicationDetailReq) (*dtotenantapplication.TenantApplicationDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenantapplication.TenantApplicationPageListReq) (*dtotenantapplication.TenantApplicationPageListResp, error)
}

type tenantApplicationSvc struct{}

var _ TenantApplicationSvc = (*tenantApplicationSvc)(nil)

func NewTenantApplicationSvc() TenantApplicationSvc {
	return &tenantApplicationSvc{}
}

// isValidTenantApplicationStatus 校验订阅状态取值：空值表示「不指定/不修改」（Create 落默认值、Update 跳过），
// 非空必须命中白名单常量。校验归 service（AGENTS.md 硬规则 3）。
func isValidTenantApplicationStatus(status model.TenantApplicationStatus) bool {
	switch status {
	case "", model.TenantApplicationStatusEnable, model.TenantApplicationStatusDisable:
		return true
	default:
		return false
	}
}

// 平台侧跨租户运维：订阅的归属租户来自请求参数（而非调用者 token 里的租户），
// 读改删同样不校验 ctx 租户归属——与 /v1/platform/tenants/{tenantID} 同一信任模型
// （能拿到 platform-admin-web 令牌即可运维任意租户）。
func (svc *tenantApplicationSvc) Create(ctx *gin.Context, req *dtotenantapplication.TenantApplicationCreateReq) (*dtotenantapplication.TenantApplicationCreateResp, error) {
	if !isValidTenantApplicationStatus(req.Status) {
		glog.Errorf(ctx, "[svctenantapplication.Create] 非法订阅状态, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationCreateError)
	}
	// 1) 归属租户必须存在
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Create] dao tenant GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationCreateError)
	}
	if tenantEntity == nil || tenantEntity.ID == "" {
		return nil, code.GetError(code.TenantNotExistError)
	}

	// 2) 订阅必须指向真实存在的应用：否则租户侧按订阅反查应用时会静默丢弃（loadSubscribedApps），
	// 留下一条永远不生效的悬空订阅。
	app, err := dao.NewApplicationDao().GetByID(ctx, req.AppID)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Create] dao application GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationCreateError)
	}
	if app == nil || app.ID == "" {
		return nil, code.GetError(code.ApplicationNotExistError)
	}

	// 3) tenant_application 无唯一索引（幂等只能由应用层保证，见 pkg/iam/tenant.ProvisionTenantAdmin），
	// 故同一租户对同一应用只允许一条订阅，重复订阅在这里拦截。
	existing, err := dao.NewTenantApplicationDao().GetByCond(ctx, &dao.TenantApplicationCond{TenantID: req.TenantID, AppID: req.AppID})
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Create] dao tenantApplication GetByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationCreateError)
	}
	if existing != nil && existing.ID != "" {
		return nil, code.GetError(code.TenantApplicationExistError)
	}

	entity := &model.TenantApplicationEntity{
		TenantID:  req.TenantID,
		AppID:     req.AppID,
		Status:    req.Status,
		CreatedBy: gincontext.GetUserIDString(ctx),
	}
	if entity.Status == "" {
		entity.Status = model.TenantApplicationStatusEnable
	}
	// PG 下 not null JSON 列不接受 NULL：无配置时显式给默认值（与租户自建订阅路径一致）。
	if entity.Config == nil {
		entity.Config = datatypes.JSON("{}")
	}
	if entity.GrantedScope == nil {
		entity.GrantedScope = datatypes.JSON("[]")
	}
	if req.Config != "" {
		entity.Config = datatypes.JSON([]byte(req.Config))
	}
	if req.GrantedScope != "" {
		entity.GrantedScope = datatypes.JSON([]byte(req.GrantedScope))
	}
	if err := dao.NewTenantApplicationDao().Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationCreateError)
	}
	return &dtotenantapplication.TenantApplicationCreateResp{TenantAppID: entity.ID}, nil
}

func (svc *tenantApplicationSvc) Delete(ctx *gin.Context, req *dtotenantapplication.TenantApplicationDeleteReq) error {
	entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantApplicationDeleteError)
	}
	if entity == nil || entity.ID == "" {
		return code.GetError(code.TenantApplicationNotExistError)
	}
	if err := dao.NewTenantApplicationDao().Delete(ctx, req.TenantAppID, gincontext.GetUserIDString(ctx)); err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantApplicationDeleteError)
	}
	return nil
}

func (svc *tenantApplicationSvc) Update(ctx *gin.Context, req *dtotenantapplication.TenantApplicationUpdateReq) error {
	if !isValidTenantApplicationStatus(req.Status) {
		glog.Errorf(ctx, "[svctenantapplication.Update] 非法订阅状态, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.TenantApplicationUpdateError)
	}
	entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantApplicationUpdateError)
	}
	if entity == nil || entity.ID == "" {
		return code.GetError(code.TenantApplicationNotExistError)
	}
	updateMap := map[string]any{
		"updated_by": gincontext.GetUserIDString(ctx),
	}
	// status 留空表示不修改：不写该列，避免把状态覆盖为空串
	if req.Status != "" {
		updateMap["status"] = req.Status
	}
	if req.Config != "" {
		updateMap["config"] = datatypes.JSON([]byte(req.Config))
	}
	if req.GrantedScope != "" {
		updateMap["granted_scope"] = datatypes.JSON([]byte(req.GrantedScope))
	}
	if err := dao.NewTenantApplicationDao().UpdateMap(ctx, req.TenantAppID, updateMap); err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantApplicationUpdateError)
	}
	return nil
}

func (svc *tenantApplicationSvc) Detail(ctx *gin.Context, req *dtotenantapplication.TenantApplicationDetailReq) (*dtotenantapplication.TenantApplicationDetailResp, error) {
	entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationGetDetailError)
	}
	if entity == nil || entity.ID == "" {
		return nil, code.GetError(code.TenantApplicationNotExistError)
	}
	tenantNames, appNames, err := loadNameMaps(ctx, model.TenantApplicationEntityList{*entity})
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Detail] loadNameMaps fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationGetDetailError)
	}
	return &dtotenantapplication.TenantApplicationDetailResp{
		TenantAppID:  entity.ID,
		TenantID:     entity.TenantID,
		TenantName:   tenantNames[entity.TenantID],
		AppID:        entity.AppID,
		AppName:      appNames[entity.AppID],
		Status:       entity.Status,
		Config:       string(entity.Config),
		GrantedScope: string(entity.GrantedScope),
		CreatedAt:    entity.CreatedAt.Unix(),
	}, nil
}

func (svc *tenantApplicationSvc) PageList(ctx *gin.Context, req *dtotenantapplication.TenantApplicationPageListReq) (*dtotenantapplication.TenantApplicationPageListResp, error) {
	cond := &dao.TenantApplicationCond{
		BaseCond: &gormdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
		TenantID: req.TenantID, // 留空即全部租户（平台侧跨租户视角）
		Status:   req.Status,
	}
	list, total, err := dao.NewTenantApplicationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationGetPageListError)
	}
	tenantNames, appNames, err := loadNameMaps(ctx, list)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.PageList] loadNameMaps fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationGetPageListError)
	}
	items := make([]dtotenantapplication.PageListItem, 0, len(list))
	for _, v := range list {
		items = append(items, dtotenantapplication.PageListItem{
			TenantAppID: v.ID,
			TenantID:    v.TenantID,
			TenantName:  tenantNames[v.TenantID],
			AppID:       v.AppID,
			AppName:     appNames[v.AppID],
			Status:      v.Status,
			CreatedAt:   v.CreatedAt.Unix(),
			UpdatedAt:   v.UpdatedAt.Unix(),
		})
	}
	return &dtotenantapplication.TenantApplicationPageListResp{List: items, Total: total}, nil
}

// loadNameMaps 批量回填租户名与应用名（列表/详情一次查询，避免前端按行再查或 N+1）。
// 名称缺失不报错：调用方以空名返回，前端退化为展示 ID。
func loadNameMaps(ctx *gin.Context, list model.TenantApplicationEntityList) (map[string]string, map[string]string, error) {
	tenantIDs := make([]string, 0, len(list))
	appIDs := make([]string, 0, len(list))
	seenTenant := make(map[string]struct{}, len(list))
	seenApp := make(map[string]struct{}, len(list))
	for _, v := range list {
		if v.TenantID != "" {
			if _, ok := seenTenant[v.TenantID]; !ok {
				seenTenant[v.TenantID] = struct{}{}
				tenantIDs = append(tenantIDs, v.TenantID)
			}
		}
		if v.AppID != "" {
			if _, ok := seenApp[v.AppID]; !ok {
				seenApp[v.AppID] = struct{}{}
				appIDs = append(appIDs, v.AppID)
			}
		}
	}

	tenantNames := make(map[string]string, len(tenantIDs))
	if len(tenantIDs) > 0 {
		tenants, err := dao.NewTenantDao().GetListByCond(ctx, &dao.TenantCond{IDs: tenantIDs})
		if err != nil {
			return nil, nil, err
		}
		for _, t := range tenants {
			tenantNames[t.ID] = t.Name
		}
	}

	appNames := make(map[string]string, len(appIDs))
	if len(appIDs) > 0 {
		apps, err := dao.NewApplicationDao().GetListByCond(ctx, &dao.ApplicationCond{IDs: appIDs})
		if err != nil {
			return nil, nil, err
		}
		for _, a := range apps {
			appNames[a.ID] = a.Name
		}
	}
	return tenantNames, appNames, nil
}
