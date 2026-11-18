package svcconnector

import (
	"context"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtoconnector"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/connector"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

// ConnectorSvc 连接器「管理面」服务（平台级全局对象，跨租户运维）。
// 运行时（Authorize/Callback）仍在 apps/auth/svcauth；领域不变式复用 pkg/core/connector。
type ConnectorSvc interface {
	Create(ctx context.Context, req *dtoconnector.ConnectorCreateReq) (*dtoconnector.ConnectorCreateResp, error)
	Delete(ctx context.Context, req *dtoconnector.ConnectorDeleteReq) error
	Update(ctx context.Context, req *dtoconnector.ConnectorUpdateReq) error
	Detail(ctx context.Context, req *dtoconnector.ConnectorDetailReq) (*dtoconnector.ConnectorDetailResp, error)
	PageList(ctx context.Context, req *dtoconnector.ConnectorPageListReq) (*dtoconnector.ConnectorPageListResp, error)
	ListFactories(ctx context.Context, req *dtoconnector.ConnectorFactoryListReq) (*dtoconnector.ConnectorFactoryListResp, error)
	TestConnector(ctx context.Context, req *dtoconnector.ConnectorIDReq) (*dtoconnector.ConnectorTestResp, error)
}

type connectorSvc struct {
	db gormdao.DBGetter
}

var _ ConnectorSvc = (*connectorSvc)(nil)

func NewConnectorSvc(db gormdao.DBGetter) ConnectorSvc {
	if db == nil {
		db = dbclient.IamDB
	}
	return &connectorSvc{db: db}
}

func (svc *connectorSvc) connectorDao() *dao.ConnectorDao {
	return dao.NewConnectorDao(dao.WithDBGetter(svc.db))
}

func (svc *connectorSvc) tenantDao() *dao.TenantDao {
	return dao.NewTenantDao(dao.WithDBGetter(svc.db))
}

// Create 平台侧创建连接器（跨租户）。
// 校验：状态合法性、四个开关枚举归一化、auto-create 且要求默认落租户时必须存在有效租户。
func (svc *connectorSvc) Create(ctx context.Context, req *dtoconnector.ConnectorCreateReq) (*dtoconnector.ConnectorCreateResp, error) {
	if !connector.IsValidStatus(req.Status) {
		glog.Errorf(ctx, "[svcconnector.Create] 非法连接器状态, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorCreateError)
	}
	baseInfo := &objauth.ConnectorBaseInfo{
		DefaultTenantID:     req.DefaultTenantID,
		Name:                req.Name,
		DisplayName:         req.DisplayName,
		Protocol:            req.Protocol,
		Provider:            req.Provider,
		Status:              req.Status,
		AllowAutoCreateUser: req.AllowAutoCreateUser,
		AllowAccountLink:    req.AllowAccountLink,
		SyncProfile:         req.SyncProfile,
		EnableTokenStorage:  req.EnableTokenStorage,
		Config:              req.Config,
		ClaimMapping:        req.ClaimMapping,
		DomainPolicy:        req.DomainPolicy,
	}
	if !connector.NormalizeSwitches(baseInfo) {
		glog.Errorf(ctx, "[svcconnector.Create] 非法连接器开关值, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorCreateError)
	}

	// 平台侧跨租户运维：连接器全局无归属租户，作用域显式声明为「全租户」。
	crossCtx := dbclient.CrossTenantContext(ctx)

	// 自动开户必须能确定落哪个租户：开启 auto-create 时 default_tenant_id 必填且租户须存在。
	if baseInfo.AllowAutoCreateUser == model.ConnectorAutoCreateUserFlagEnable {
		if baseInfo.DefaultTenantID == "" {
			return nil, code.GetError(code.ConnectorDefaultTenantRequiredError)
		}
		tenantEntity, err := svc.tenantDao().GetByID(crossCtx, baseInfo.DefaultTenantID)
		if err != nil {
			glog.Errorf(ctx, "[svcconnector.Create] dao tenant GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.ConnectorCreateError)
		}
		if tenantEntity == nil || tenantEntity.ID == "" {
			return nil, code.GetError(code.ConnectorDefaultTenantRequiredError)
		}
	} else if baseInfo.DefaultTenantID != "" {
		// 未开启自动开户但显式指定了默认落租户：仍校验其存在，避免悬空引用。
		tenantEntity, err := svc.tenantDao().GetByID(crossCtx, baseInfo.DefaultTenantID)
		if err != nil {
			glog.Errorf(ctx, "[svcconnector.Create] dao tenant GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.ConnectorCreateError)
		}
		if tenantEntity == nil || tenantEntity.ID == "" {
			return nil, code.GetError(code.ConnectorDefaultTenantRequiredError)
		}
	}

	entity := &model.ConnectorEntity{
		DefaultTenantID:     baseInfo.DefaultTenantID,
		Name:                baseInfo.Name,
		DisplayName:         baseInfo.DisplayName,
		Protocol:            baseInfo.Protocol,
		Provider:            baseInfo.Provider,
		Status:              baseInfo.Status,
		AllowAutoCreateUser: baseInfo.AllowAutoCreateUser,
		AllowAccountLink:    baseInfo.AllowAccountLink,
		SyncProfile:         baseInfo.SyncProfile,
		EnableTokenStorage:  baseInfo.EnableTokenStorage,
		Config:              baseInfo.Config,
		ClaimMapping:        baseInfo.ClaimMapping,
		DomainPolicy:        baseInfo.DomainPolicy,
		CreatedBy:           req.Operator.UserID,
	}
	if entity.Status == "" {
		entity.Status = model.ConnectorStatusEnable
	}
	if err := svc.connectorDao().Insert(crossCtx, entity); err != nil {
		glog.Errorf(ctx, "[svcconnector.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorCreateError)
	}
	return &dtoconnector.ConnectorCreateResp{ConnectorID: entity.ID}, nil
}

// Delete 删除连接器（跨租户，无租户归属校验）。
func (svc *connectorSvc) Delete(ctx context.Context, req *dtoconnector.ConnectorDeleteReq) error {
	crossCtx := dbclient.CrossTenantContext(ctx)
	entity, err := svc.connectorDao().GetByID(crossCtx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcconnector.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorDeleteError)
	}
	if entity == nil || entity.ID == "" {
		return code.GetError(code.ConnectorNotExistError)
	}
	if err := svc.connectorDao().Delete(crossCtx, req.ConnectorID, req.Operator.UserID); err != nil {
		glog.Errorf(ctx, "[svcconnector.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorDeleteError)
	}
	return nil
}

// Update 修改连接器（跨租户，允许编辑 default_tenant_id）。
// config 走 PreserveSecret（保留库内已存 secret）；JSON 列用结构化 Updates + 显式字段。
func (svc *connectorSvc) Update(ctx context.Context, req *dtoconnector.ConnectorUpdateReq) error {
	if !connector.IsValidStatus(req.Status) {
		glog.Errorf(ctx, "[svcconnector.Update] 非法连接器状态, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}
	baseInfo := &objauth.ConnectorBaseInfo{
		DefaultTenantID:     req.DefaultTenantID,
		Name:                req.Name,
		DisplayName:         req.DisplayName,
		Protocol:            req.Protocol,
		Provider:            req.Provider,
		Status:              req.Status,
		AllowAutoCreateUser: req.AllowAutoCreateUser,
		AllowAccountLink:    req.AllowAccountLink,
		SyncProfile:         req.SyncProfile,
		EnableTokenStorage:  req.EnableTokenStorage,
		Config:              req.Config,
		ClaimMapping:        req.ClaimMapping,
		DomainPolicy:        req.DomainPolicy,
	}
	if !connector.NormalizeSwitches(baseInfo) {
		glog.Errorf(ctx, "[svcconnector.Update] 非法连接器开关值, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}

	crossCtx := dbclient.CrossTenantContext(ctx)
	existing, err := svc.connectorDao().GetByID(crossCtx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcconnector.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}
	if existing == nil || existing.ID == "" {
		return code.GetError(code.ConnectorNotExistError)
	}

	// auto-create 且要求落租户：与 Create 同口径校验（开启时 default_tenant_id 必填且存在）。
	if baseInfo.AllowAutoCreateUser == model.ConnectorAutoCreateUserFlagEnable {
		if baseInfo.DefaultTenantID == "" {
			return code.GetError(code.ConnectorDefaultTenantRequiredError)
		}
	}
	if baseInfo.DefaultTenantID != "" {
		tenantEntity, terr := svc.tenantDao().GetByID(crossCtx, baseInfo.DefaultTenantID)
		if terr != nil {
			glog.Errorf(ctx, "[svcconnector.Update] dao tenant GetByID fail, err:%v, req:%s", terr, gutil.ToJsonString(req))
			return code.GetError(code.ConnectorUpdateError)
		}
		if tenantEntity == nil || tenantEntity.ID == "" {
			return code.GetError(code.ConnectorDefaultTenantRequiredError)
		}
	}

	status := baseInfo.Status
	if status == "" {
		status = existing.Status
	}
	entity := &model.ConnectorEntity{
		DefaultTenantID:     baseInfo.DefaultTenantID,
		Name:                baseInfo.Name,
		DisplayName:         baseInfo.DisplayName,
		Protocol:            baseInfo.Protocol,
		Provider:            baseInfo.Provider,
		Status:              status,
		AllowAutoCreateUser: baseInfo.AllowAutoCreateUser,
		AllowAccountLink:    baseInfo.AllowAccountLink,
		SyncProfile:         baseInfo.SyncProfile,
		EnableTokenStorage:  baseInfo.EnableTokenStorage,
		Config:              connector.PreserveSecret(baseInfo.Config, existing.Config),
		ClaimMapping:        baseInfo.ClaimMapping,
		DomainPolicy:        baseInfo.DomainPolicy,
		UpdatedBy:           req.Operator.UserID,
	}
	fields := []string{
		"default_tenant_id", "name", "display_name", "protocol", "provider", "status",
		"allow_auto_create_user", "allow_account_link", "sync_profile", "enable_token_storage",
		"config", "claim_mapping", "domain_policy", "updated_by",
	}
	if err := dao.UpdateFields(crossCtx, svc.connectorDao().Dao, req.ConnectorID, entity, fields...); err != nil {
		glog.Errorf(ctx, "[svcconnector.Update] dao UpdateFields fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.ConnectorUpdateError)
	}
	return nil
}

// Detail 连接器详情（跨租户，无租户归属校验）。
func (svc *connectorSvc) Detail(ctx context.Context, req *dtoconnector.ConnectorDetailReq) (*dtoconnector.ConnectorDetailResp, error) {
	crossCtx := dbclient.CrossTenantContext(ctx)
	entity, err := svc.connectorDao().GetByID(crossCtx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcconnector.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if entity == nil || entity.ID == "" {
		return nil, code.GetError(code.ConnectorNotExistError)
	}
	tenantNames, err := svc.loadTenantNames(crossCtx, model.ConnectorEntityList{*entity})
	if err != nil {
		glog.Errorf(ctx, "[svcconnector.Detail] loadTenantNames fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	return &dtoconnector.ConnectorDetailResp{
		ConnectorID:        entity.ID,
		ConnectorBaseInfo:  toBaseInfo(*entity),
		DefaultTenantName:  tenantNames[entity.DefaultTenantID],
		OperatorBaseInfo: gobject.OperatorBaseInfo{CreatedAt: entity.CreatedAt.Unix(), UpdatedAt: entity.UpdatedAt.Unix()},
	}, nil
}

// PageList 连接器列表（跨租户视角，defaultTenantID 留空即全部）。
func (svc *connectorSvc) PageList(ctx context.Context, req *dtoconnector.ConnectorPageListReq) (*dtoconnector.ConnectorPageListResp, error) {
	crossCtx := dbclient.CrossTenantContext(ctx)
	cond := &dao.ConnectorCond{
		BaseCond:        &gormdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
		DefaultTenantID: req.DefaultTenantID,
		Protocol:        req.Protocol,
		Provider:        req.Provider,
		Status:          req.Status,
		Name:            req.Name,
		DisplayName:     req.DisplayName,
	}
	list, total, err := svc.connectorDao().GetPageListByCond(crossCtx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcconnector.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetPageListError)
	}
	tenantNames, err := svc.loadTenantNames(crossCtx, list)
	if err != nil {
		glog.Errorf(ctx, "[svcconnector.PageList] loadTenantNames fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetPageListError)
	}
	items := make([]dtoconnector.ConnectorPageListItem, 0, len(list))
	for _, v := range list {
		items = append(items, dtoconnector.ConnectorPageListItem{
			ConnectorID:       v.ID,
			ConnectorBaseInfo: toBaseInfo(v),
			DefaultTenantName: tenantNames[v.DefaultTenantID],
			OperatorBaseInfo:  gobject.OperatorBaseInfo{CreatedAt: v.CreatedAt.Unix(), UpdatedAt: v.UpdatedAt.Unix()},
		})
	}
	return &dtoconnector.ConnectorPageListResp{List: items, Total: total}, nil
}

// ListFactories 返回内置连接器工厂定义（按 protocol/provider 过滤）。
func (svc *connectorSvc) ListFactories(ctx context.Context, req *dtoconnector.ConnectorFactoryListReq) (*dtoconnector.ConnectorFactoryListResp, error) {
	factories := connector.SelectFactories(req.Protocol, req.Provider)
	items := make([]dtoconnector.ConnectorFactoryResp, 0, len(factories))
	for _, f := range factories {
		items = append(items, dtoconnector.ConnectorFactoryResp{
			FactoryID:     f.FactoryID,
			Protocol:      f.Protocol,
			Provider:      f.Provider,
			DisplayName:   f.DisplayName,
			IsStandard:    f.IsStandard,
			DefaultScopes: f.DefaultScopes,
			Capabilities:  f.Capabilities,
			ConfigSchema:  f.ConfigSchema,
		})
	}
	return &dtoconnector.ConnectorFactoryListResp{List: items}, nil
}

// TestConnector 对连接器做连通性测试（选择驱动后调用 TestConnection）。
func (svc *connectorSvc) TestConnector(ctx context.Context, req *dtoconnector.ConnectorIDReq) (*dtoconnector.ConnectorTestResp, error) {
	crossCtx := dbclient.CrossTenantContext(ctx)
	entity, err := svc.connectorDao().GetByID(crossCtx, req.ConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcconnector.TestConnector] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	if entity == nil || entity.ID == "" {
		return nil, code.GetError(code.ConnectorNotExistError)
	}
	driver, config, err := connector.SelectDriverForConnector(connector.DefaultDriverRegistry(), entity)
	if err != nil {
		glog.Errorf(ctx, "[svcconnector.TestConnector] select driver fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	out, err := driver.TestConnection(ctx, &connector.TestInput{Config: config})
	if err != nil {
		glog.Errorf(ctx, "[svcconnector.TestConnector] driver TestConnection fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.ConnectorGetDetailError)
	}
	return &dtoconnector.ConnectorTestResp{Success: out.Success, Message: out.Message}, nil
}

// toBaseInfo 把实体转为前端展示的 ConnectorBaseInfo：config 脱敏（clientSecret 不回传明文）。
func toBaseInfo(entity model.ConnectorEntity) objauth.ConnectorBaseInfo {
	return objauth.ConnectorBaseInfo{
		DefaultTenantID:     entity.DefaultTenantID,
		Name:                entity.Name,
		DisplayName:         entity.DisplayName,
		Protocol:            entity.Protocol,
		Provider:            entity.Provider,
		Status:              entity.Status,
		AllowAutoCreateUser: entity.AllowAutoCreateUser,
		AllowAccountLink:    entity.AllowAccountLink,
		SyncProfile:         entity.SyncProfile,
		EnableTokenStorage:  entity.EnableTokenStorage,
		Config:              connector.SanitizeConfig(entity.Config),
		ClaimMapping:        entity.ClaimMapping,
		DomainPolicy:        entity.DomainPolicy,
	}
}

// loadTenantNames 批量回填默认落租户名称（一次查询，避免 N+1）。
func (svc *connectorSvc) loadTenantNames(ctx context.Context, list model.ConnectorEntityList) (map[string]string, error) {
	names := make(map[string]string, len(list))
	ids := make([]string, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, v := range list {
		if v.DefaultTenantID == "" {
			continue
		}
		if _, ok := seen[v.DefaultTenantID]; ok {
			continue
		}
		seen[v.DefaultTenantID] = struct{}{}
		ids = append(ids, v.DefaultTenantID)
	}
	if len(ids) == 0 {
		return names, nil
	}
	tenants, err := svc.tenantDao().GetListByCond(ctx, &dao.TenantCond{IDs: ids})
	if err != nil {
		return nil, err
	}
	for _, t := range tenants {
		names[t.ID] = t.Name
	}
	return names, nil
}
