package svctenantapplication

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/tenant"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
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
// （能拿到 platform_admin_web 令牌即可运维任意租户）。
func (svc *tenantApplicationSvc) Create(ctx *gin.Context, req *dtotenantapplication.TenantApplicationCreateReq) (*dtotenantapplication.TenantApplicationCreateResp, error) {
	if !isValidTenantApplicationStatus(req.Status) {
		glog.Errorf(ctx, "[svctenantapplication.Create] 非法订阅状态, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationCreateError)
	}
	// 平台侧跨租户运维：订阅归属租户来自请求参数，调用方 ctx 携带的是平台租户，
	// 因此这里必须显式声明「全租户」作用域——跨租户可见性不能来自「ctx 恰好没有作用域」。
	crossCtx := dbclient.CrossTenantContext(ctx)
	// 1) 归属租户必须存在
	tenantEntity, err := dao.NewTenantDao().GetByID(crossCtx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Create] dao tenant GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationCreateError)
	}
	if tenantEntity == nil || tenantEntity.ID == "" {
		return nil, code.GetError(code.TenantNotExistError)
	}

	// 2) 订阅必须指向真实存在的应用：否则租户侧按订阅反查应用时会静默丢弃（loadSubscribedApps），
	// 留下一条永远不生效的悬空订阅。
	app, err := dao.NewApplicationDao().GetByID(crossCtx, req.AppID)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Create] dao application GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationCreateError)
	}
	if app == nil || app.ID == "" {
		return nil, code.GetError(code.ApplicationNotExistError)
	}

	// 3) tenant_application 无唯一索引（幂等只能由应用层保证，见 pkg/core/tenant.ProvisionTenantAdmin），
	// 故同一租户对同一应用只允许一条订阅，重复订阅在这里拦截。
	existing, err := dao.NewTenantApplicationDao().GetByCond(crossCtx, &dao.TenantApplicationCond{TenantID: req.TenantID, AppID: req.AppID})
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
	// 4) 订阅落行即代表该应用在此租户内可用，故同事务把应用角色模板物化到该租户：
	// 缺少物化时租户下没有该应用的角色、用户 ID token 的 groups 为空，下游策略集为空即 Deny
	// （见 docs/design/application-integration-guide.md §3.4）。
	txErr := dbclient.IamDB(crossCtx).Transaction(func(tx *gorm.DB) error {
		if err := dao.NewTenantApplicationDao().WithTx(tx).Insert(crossCtx, entity); err != nil {
			return err
		}
		return tenant.SyncAppRoleTemplate(crossCtx, tx, &tenant.SyncAppRoleTemplateReq{
			TenantID:  req.TenantID,
			AppID:     req.AppID,
			CreatedBy: gincontext.GetUserIDString(ctx),
		})
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svctenantapplication.Create] insert subscription/sync role template fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationCreateError)
	}
	return &dtotenantapplication.TenantApplicationCreateResp{TenantAppID: entity.ID}, nil
}

// Delete 删除租户应用订阅。
//
// 内置应用的订阅禁删：种子为平台租户写入 platform_admin 订阅、ProvisionTenantAdmin 为每个租户
// 写入 tenant_admin 订阅——它们由系统开通，删除会让对应控制台当场失去菜单（平台侧整栈失联、
// 租户侧无法自救）。判定口径取「订阅的应用 source=builtin」，与 svcapplication.Delete 的
// 内置应用禁删同源；需要下线时改 status=disable（Update 不受限）。
func (svc *tenantApplicationSvc) Delete(ctx *gin.Context, req *dtotenantapplication.TenantApplicationDeleteReq) error {
	// 平台侧跨租户运维：订阅归属租户来自请求参数，调用方 ctx 携带的是平台租户，
	// 因此这里必须显式声明「全租户」作用域——跨租户可见性不能来自「ctx 恰好没有作用域」。
	crossCtx := dbclient.CrossTenantContext(ctx)
	entity, err := dao.NewTenantApplicationDao().GetByID(crossCtx, req.TenantAppID)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantApplicationDeleteError)
	}
	if entity == nil || entity.ID == "" {
		return code.GetError(code.TenantApplicationNotExistError)
	}
	app, err := dao.NewApplicationDao().GetByID(crossCtx, entity.AppID)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Delete] dao application GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantApplicationDeleteError)
	}
	// app 为空表示应用已被删除（悬空订阅）：不属内置，放行删除以清理脏数据。
	if app != nil && app.Source.IsBuiltin() {
		return code.GetError(code.TenantApplicationBuiltInErr)
	}
	if err := dao.NewTenantApplicationDao().Delete(crossCtx, req.TenantAppID, gincontext.GetUserIDString(ctx)); err != nil {
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
	// 平台侧跨租户运维：订阅归属租户来自请求参数，调用方 ctx 携带的是平台租户，
	// 因此这里必须显式声明「全租户」作用域——跨租户可见性不能来自「ctx 恰好没有作用域」。
	crossCtx := dbclient.CrossTenantContext(ctx)
	entity, err := dao.NewTenantApplicationDao().GetByID(crossCtx, req.TenantAppID)
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
	if err := dao.NewTenantApplicationDao().UpdateMap(crossCtx, req.TenantAppID, updateMap); err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantApplicationUpdateError)
	}
	return nil
}

func (svc *tenantApplicationSvc) Detail(ctx *gin.Context, req *dtotenantapplication.TenantApplicationDetailReq) (*dtotenantapplication.TenantApplicationDetailResp, error) {
	// 平台侧跨租户运维：订阅归属租户来自请求参数，调用方 ctx 携带的是平台租户，
	// 因此这里必须显式声明「全租户」作用域——跨租户可见性不能来自「ctx 恰好没有作用域」。
	crossCtx := dbclient.CrossTenantContext(ctx)
	entity, err := dao.NewTenantApplicationDao().GetByID(crossCtx, req.TenantAppID)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationGetDetailError)
	}
	if entity == nil || entity.ID == "" {
		return nil, code.GetError(code.TenantApplicationNotExistError)
	}
	tenantNames, appNames, appSources, err := loadRefs(crossCtx, model.TenantApplicationEntityList{*entity})
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.Detail] loadRefs fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationGetDetailError)
	}
	return &dtotenantapplication.TenantApplicationDetailResp{
		TenantAppID:  entity.ID,
		TenantID:     entity.TenantID,
		TenantName:   tenantNames[entity.TenantID],
		AppID:        entity.AppID,
		AppName:      appNames[entity.AppID],
		AppSource:    appSources[entity.AppID],
		Status:       entity.Status,
		Config:       string(entity.Config),
		GrantedScope: string(entity.GrantedScope),
		CreatedAt:    entity.CreatedAt.Unix(),
	}, nil
}

func (svc *tenantApplicationSvc) PageList(ctx *gin.Context, req *dtotenantapplication.TenantApplicationPageListReq) (*dtotenantapplication.TenantApplicationPageListResp, error) {
	// 平台侧跨租户运维：订阅归属租户来自请求参数，调用方 ctx 携带的是平台租户，
	// 因此这里必须显式声明「全租户」作用域——跨租户可见性不能来自「ctx 恰好没有作用域」。
	crossCtx := dbclient.CrossTenantContext(ctx)
	cond := &dao.TenantApplicationCond{
		BaseCond: &gormdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
		TenantID: req.TenantID, // 留空即全部租户（平台侧跨租户视角）
		Status:   req.Status,
	}
	list, total, err := dao.NewTenantApplicationDao().GetPageListByCond(crossCtx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantApplicationGetPageListError)
	}
	tenantNames, appNames, appSources, err := loadRefs(crossCtx, list)
	if err != nil {
		glog.Errorf(ctx, "[svctenantapplication.PageList] loadRefs fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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
			AppSource:   appSources[v.AppID],
			Status:      v.Status,
			CreatedAt:   v.CreatedAt.Unix(),
			UpdatedAt:   v.UpdatedAt.Unix(),
		})
	}
	return &dtotenantapplication.TenantApplicationPageListResp{List: items, Total: total}, nil
}

// loadRefs 批量回填租户名与应用名/来源（列表/详情一次查询，避免前端按行再查或 N+1）。
// 名称缺失不报错：调用方以空名返回，前端退化为展示 ID；应用来源供前端判断内置订阅（禁删）。
func loadRefs(ctx context.Context, list model.TenantApplicationEntityList) (map[string]string, map[string]string, map[string]model.AppSource, error) {
	// 平台侧跨租户运维：订阅归属租户来自请求参数，调用方 ctx 携带的是平台租户，
	// 因此这里必须显式声明「全租户」作用域——跨租户可见性不能来自「ctx 恰好没有作用域」。
	crossCtx := dbclient.CrossTenantContext(ctx)
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
		tenants, err := dao.NewTenantDao().GetListByCond(crossCtx, &dao.TenantCond{IDs: tenantIDs})
		if err != nil {
			return nil, nil, nil, err
		}
		for _, t := range tenants {
			tenantNames[t.ID] = t.Name
		}
	}

	appNames := make(map[string]string, len(appIDs))
	appSources := make(map[string]model.AppSource, len(appIDs))
	if len(appIDs) > 0 {
		apps, err := dao.NewApplicationDao().GetListByCond(crossCtx, &dao.ApplicationCond{IDs: appIDs})
		if err != nil {
			return nil, nil, nil, err
		}
		for _, a := range apps {
			appNames[a.ID] = a.Name
			appSources[a.ID] = a.Source
		}
	}
	return tenantNames, appNames, appSources, nil
}
