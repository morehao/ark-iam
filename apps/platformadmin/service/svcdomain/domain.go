package svcdomain

import (
	"context"
	"strings"
	"time"

	"github.com/morehao/ark-iam/apps/platformadmin/dto/dtodomain"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/tenantdomain"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

// DomainSvc 域名管理（平台端）。动作端点（verify/tls-ready/activate/disable/retire/enable/release）
// 编排 pkg/core/tenantdomain 状态机；Create/PageList/Detail/Delete 为平台代租户管理入口。
// 不提供通用 PUT 改 domain/role/usage/status（改名=退役+新登记，状态只走动作端点）。
type DomainSvc interface {
	Create(ctx context.Context, req *dtodomain.DomainCreateReq) (*dtodomain.DomainCreateResp, error)
	PageList(ctx context.Context, req *dtodomain.DomainPageListReq) (*dtodomain.DomainPageListResp, error)
	Detail(ctx context.Context, req *dtodomain.DomainDetailReq) (*dtodomain.DomainDetailResp, error)
	Delete(ctx context.Context, req *dtodomain.DomainDeleteReq) error

	Verify(ctx context.Context, req *dtodomain.DomainVerifyReq) (*dtodomain.DomainActionResp, error)
	TLSReady(ctx context.Context, req *dtodomain.DomainTLSReadyReq) (*dtodomain.DomainActionResp, error)
	Activate(ctx context.Context, req *dtodomain.DomainActivateReq) (*dtodomain.DomainActionResp, error)
	Disable(ctx context.Context, req *dtodomain.DomainDisableReq) (*dtodomain.DomainActionResp, error)
	Enable(ctx context.Context, req *dtodomain.DomainEnableReq) (*dtodomain.DomainActionResp, error)
	Retire(ctx context.Context, req *dtodomain.DomainRetireReq) (*dtodomain.DomainActionResp, error)
	Release(ctx context.Context, req *dtodomain.DomainReleaseReq) (*dtodomain.DomainActionResp, error)
}

type domainSvc struct {
	db       gormdao.DBGetter
	basePath string // 部署挂载前缀（空或 /iam），派生 issuer 用；从部署配置注入（P0 占位空）
}

var _ DomainSvc = (*domainSvc)(nil)

func NewDomainSvc(db gormdao.DBGetter) DomainSvc {
	if db == nil {
		db = dbclient.IamDB
	}
	return &domainSvc{db: db}
}

// newTx 构造 tenantdomain.Tx（在调用方事务内）。
func (svc *domainSvc) tx(db *gorm.DB, targetTenantID, operator string) *tenantdomain.Tx {
	return &tenantdomain.Tx{DB: db, TenantID: targetTenantID, Operator: operator, BasePath: svc.basePath}
}

// mapDomainErr 把领域哨兵映射为功能级错误码（101207–101213 / 通用域错误）。
func mapDomainErr(err error, fallback int) error {
	switch {
	case err == nil:
		return nil
	case err == tenantdomain.ErrDomainNotExist:
		return code.GetError(code.DomainNotExistError)
	case err == tenantdomain.ErrTenantNotExist, err == tenantdomain.ErrTenantStatusInvalid:
		return code.GetError(code.DomainTenantSuspendedError)
	case err == tenantdomain.ErrInvalidTransition:
		return code.GetError(code.DomainStatusInvalidError)
	case err == tenantdomain.ErrPrimaryUsageInvalid:
		return code.GetError(code.DomainStatusInvalidError)
	case err == tenantdomain.ErrAlreadyActivePrimary, err == tenantdomain.ErrConflict:
		return code.GetError(code.DomainStatusInvalidError)
	case err == tenantdomain.ErrTargetMismatch:
		return code.GetError(code.DomainIssuerTierMismatchError)
	case err == tenantdomain.ErrTargetNotActive:
		return code.GetError(code.DomainStatusInvalidError)
	case err == tenantdomain.ErrReferencedByAlias:
		return code.GetError(code.DomainReservedError)
	case err == tenantdomain.ErrReleaseNotDue:
		return code.GetError(code.DomainStatusInvalidError)
	default:
		return code.GetError(fallback)
	}
}

// runAction 在事务内执行一个域名动作，提交后清 host 缓存（返回的 hosts）。
func (svc *domainSvc) runAction(ctx context.Context, targetTenantID, operator string,
	fn func(tx *tenantdomain.Tx) ([]string, error)) error {
	var hosts []string
	txErr := svc.db(ctx).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		h, err := fn(svc.tx(tx, targetTenantID, operator))
		if err != nil {
			return err
		}
		hosts = h
		return nil
	})
	if txErr != nil {
		if mapped := mapDomainErr(txErr, code.DomainUpdateError); mapped != nil {
			// 已知领域错误按映射返回；未知系统错误记日志并返回 fallback
			if isDomainSentinel(txErr) {
				return mapped
			}
		}
		glog.Errorf(ctx, "[svcdomain] action tx fail, err:%v", txErr)
		return mapDomainErr(txErr, code.DomainUpdateError)
	}
	invalidateHosts(ctx, hosts)
	return nil
}

func isDomainSentinel(err error) bool {
	switch err {
	case tenantdomain.ErrDomainNotExist, tenantdomain.ErrTenantNotExist, tenantdomain.ErrTenantStatusInvalid,
		tenantdomain.ErrInvalidTransition, tenantdomain.ErrPrimaryUsageInvalid, tenantdomain.ErrAlreadyActivePrimary,
		tenantdomain.ErrConflict, tenantdomain.ErrTargetMismatch, tenantdomain.ErrTargetNotActive,
		tenantdomain.ErrReferencedByAlias, tenantdomain.ErrReleaseNotDue:
		return true
	}
	return false
}

// invalidateHosts 提交后删除 host 路由缓存（W2 提供实现；P0 占位：无缓存则跳过）。
func invalidateHosts(ctx context.Context, hosts []string) {
	// W2 host 缓存落地后在此注入 dbclient.RedisCli.Del("domain_route:"+host)。
	// 现在缓存未接入，无需操作；保留钩子避免后续遗漏（停签门禁读 DB，不依赖缓存）。
	_ = ctx
	_ = hosts
}

// tenantFromCtx 取本请求作用域的租户（平台代租户操作时用 req.TenantID）。
func targetTenant(ctx context.Context, tenantID string) string {
	if tenantID != "" {
		return tenantID
	}
	return dbclient.TenantIDFromContext(ctx)
}

func normalizeHost(h string) string {
	return strings.ToLower(strings.TrimSpace(h))
}

// Create 创建 draft 域名（平台代租户）。校验角色/用途；D alias/email 必须带 target（解析为同租户 primary）。
func (svc *domainSvc) Create(ctx context.Context, req *dtodomain.DomainCreateReq) (*dtodomain.DomainCreateResp, error) {
	tenantID := normalizeTenant(req.TenantID)
	if tenantID == "" {
		return nil, code.GetError(code.DomainTenantSuspendedError)
	}
	domain := normalizeHost(req.Domain)
	if domain == "" || len(domain) > 256 {
		return nil, code.GetError(code.DomainFormatInvalidError)
	}
	if req.DomainRole != model.DomainRolePrimary && req.DomainRole != model.DomainRoleAlias {
		return nil, code.GetError(code.DomainRoleInvalidError)
	}
	if !validUsage(req.DomainUsage) {
		return nil, code.GetError(code.DomainStatusInvalidError)
	}
	if req.DomainRole == model.DomainRolePrimary && req.DomainUsage == model.DomainUsageEmail {
		return nil, code.GetError(code.DomainStatusInvalidError) // primary 必须 login/both
	}

	// 目标租户必须存在且未删除（iam_tenant 是 skip 表，CrossTenant 查）。
	var tn model.TenantEntity
	if err := svc.db(ctx).WithContext(dbclient.CrossTenantContext(ctx)).
		Where("id = ? AND deleted_at IS NULL", tenantID).First(&tn).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, code.GetError(code.DomainTenantSuspendedError)
		}
		glog.Errorf(ctx, "[svcdomain.Create] load tenant fail, err:%v", err)
		return nil, code.GetError(code.DomainCreateError)
	}

	tenantCtx := dbclient.ExplicitTenantContext(ctx, tenantID)
	repo := dao.NewDomainDao(dao.WithDBGetter(svc.db))

	// 解析 target（alias/email 必填）：targetDomain 是 host，服务端解析为同租户 primary 行 id。
	targetDomainID := ""
	if req.DomainRole == model.DomainRoleAlias {
		if req.TargetDomain == "" {
			return nil, code.GetError(code.DomainIssuerTierMismatchError)
		}
		primary, err := repo.GetByTenantAndDomain(tenantCtx, tenantID, normalizeHost(req.TargetDomain))
		if err != nil {
			glog.Errorf(ctx, "[svcdomain.Create] resolve target fail, err:%v", err)
			return nil, code.GetError(code.DomainCreateError)
		}
		if primary == nil || primary.DomainRole != model.DomainRolePrimary {
			return nil, code.GetError(code.DomainIssuerTierMismatchError)
		}
		targetDomainID = primary.ID
	}

	method := req.VerificationMethod
	if method == "" {
		method = model.DomainVerificationManual
	}
	entity := &model.DomainEntity{
		TenantID:           tenantID,
		Domain:             domain,
		DomainRole:         req.DomainRole,
		DomainUsage:        req.DomainUsage,
		TargetDomainID:     targetDomainID,
		Status:             model.DomainStatusDraft,
		VerificationMethod: method,
		CreatedBy:          req.Operator.UserID,
		UpdatedBy:          req.Operator.UserID,
	}
	if err := repo.Insert(tenantCtx, entity); err != nil {
		glog.Errorf(ctx, "[svcdomain.Create] insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DomainCreateError)
	}
	return &dtodomain.DomainCreateResp{ID: entity.ID}, nil
}

func validUsage(u model.DomainUsage) bool {
	return u == model.DomainUsageLogin || u == model.DomainUsageEmail || u == model.DomainUsageBoth
}

func normalizeTenant(t string) string { return strings.TrimSpace(t) }

func (svc *domainSvc) PageList(ctx context.Context, req *dtodomain.DomainPageListReq) (*dtodomain.DomainPageListResp, error) {
	tenantID := dbclient.TenantIDFromContext(ctx)
	if req.TenantID != "" {
		tenantID = req.TenantID
	}
	cond := &dao.DomainCond{
		BaseCond:    &gormdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
		TenantID:    tenantID,
		Domain:      req.Domain,
		Status:      req.Status,
		DomainRole:  req.DomainRole,
		DomainUsage: req.DomainUsage,
	}
	repo := dao.NewDomainDao(dao.WithDBGetter(svc.db))
	scopeCtx := ctx
	if req.TenantID != "" {
		scopeCtx = dbclient.ExplicitTenantContext(ctx, req.TenantID)
	}
	list, total, err := repo.GetPageListByCond(scopeCtx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.PageList] GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DomainGetPageListError)
	}
	items := make([]dtodomain.DomainPageListItem, 0, len(list))
	for _, v := range list {
		items = append(items, dtodomain.DomainPageListItem{
			ID: v.ID, TenantID: v.TenantID, Domain: v.Domain, DomainRole: v.DomainRole,
			DomainUsage: v.DomainUsage, Status: v.Status,
			CreatedAt: v.CreatedAt.Unix(), UpdatedAt: v.UpdatedAt.Unix(),
		})
	}
	return &dtodomain.DomainPageListResp{List: items, Total: total}, nil
}

func (svc *domainSvc) Detail(ctx context.Context, req *dtodomain.DomainDetailReq) (*dtodomain.DomainDetailResp, error) {
	scopeCtx, entity, err := svc.load(ctx, req.DomainID)
	if err != nil {
		return nil, err
	}
	_ = scopeCtx
	resp := &dtodomain.DomainDetailResp{
		ID: entity.ID, TenantID: entity.TenantID, Domain: entity.Domain, DomainRole: entity.DomainRole,
		DomainUsage: entity.DomainUsage, TargetDomainID: entity.TargetDomainID, Status: entity.Status,
		VerificationMethod: entity.VerificationMethod, RetireReason: entity.RetireReason,
		ReleaseApproval: entity.ReleaseApproval,
		CreatedAt:       entity.CreatedAt.Unix(), UpdatedAt: entity.UpdatedAt.Unix(),
	}
	if entity.VerifiedAt != nil {
		resp.VerifiedAt = ptr(entity.VerifiedAt.Unix())
	}
	if entity.TLSReadyAt != nil {
		resp.TLSReadyAt = ptr(entity.TLSReadyAt.Unix())
	}
	if entity.RetiredAt != nil {
		resp.RetiredAt = ptr(entity.RetiredAt.Unix())
	}
	if entity.ReleasedAt != nil {
		resp.ReleasedAt = ptr(entity.ReleasedAt.Unix())
	}
	return resp, nil
}

func ptr(v int64) *int64 { return &v }

// load 加载域名行：优先按请求租户作用域；平台管理员在 AllScope 下按 ID 反查（跨租户）。
func (svc *domainSvc) load(ctx context.Context, domainID string) (context.Context, *model.DomainEntity, error) {
	tenantID := dbclient.TenantIDFromContext(ctx)
	if tenantID != "" {
		scopeCtx := dbclient.ExplicitTenantContext(ctx, tenantID)
		entity, err := dao.NewDomainDao(dao.WithDBGetter(svc.db)).GetByID(scopeCtx, domainID)
		if err != nil {
			glog.Errorf(ctx, "[svcdomain.load] GetByID fail, err:%v", err)
			return nil, nil, code.GetError(code.DomainDetailError)
		}
		if entity == nil {
			return nil, nil, code.GetError(code.DomainNotExistError)
		}
		return scopeCtx, entity, nil
	}
	// 平台 AllScope：CrossTenant 反查，取到后以其 tenant_id 重建作用域。
	crossCtx := dbclient.CrossTenantContext(ctx)
	entity, err := dao.NewDomainDao(dao.WithDBGetter(svc.db)).GetByID(crossCtx, domainID)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.load] cross GetByID fail, err:%v", err)
		return nil, nil, code.GetError(code.DomainDetailError)
	}
	if entity == nil {
		return nil, nil, code.GetError(code.DomainNotExistError)
	}
	return dbclient.ExplicitTenantContext(ctx, entity.TenantID), entity, nil
}

func (svc *domainSvc) Delete(ctx context.Context, req *dtodomain.DomainDeleteReq) error {
	scopeCtx, entity, err := svc.load(ctx, req.DomainID)
	if err != nil {
		return err
	}
	// 取消/删除未上线域名，交 tenantdomain.CancelDelete（校验未 active + 无未 released alias 引用）。
	return svc.runActionOnLoaded(ctx, scopeCtx, entity, req.Operator.UserID, func(tx *tenantdomain.Tx) ([]string, error) {
		return tx.CancelDelete(ctx, entity.ID, time.Now())
	}, code.DomainDeleteError)
}

// runActionOnLoaded 在已解析出 tenant 作用域后跑动作。
func (svc *domainSvc) runActionOnLoaded(ctx context.Context, scopeCtx context.Context, entity *model.DomainEntity,
	operator string, fn func(tx *tenantdomain.Tx) ([]string, error), fallback int) error {
	var hosts []string
	txErr := svc.db(ctx).WithContext(scopeCtx).Transaction(func(tx *gorm.DB) error {
		h, e := fn(svc.tx(tx, entity.TenantID, operator))
		if e != nil {
			return e
		}
		hosts = h
		return nil
	})
	if txErr != nil {
		if !isDomainSentinel(txErr) {
			glog.Errorf(ctx, "[svcdomain] action tx fail, err:%v", txErr)
		}
		return mapDomainErr(txErr, fallback)
	}
	invalidateHosts(ctx, hosts)
	return nil
}

func actionResp(domainID string) (*dtodomain.DomainActionResp, error) {
	return &dtodomain.DomainActionResp{DomainID: domainID}, nil
}

func (svc *domainSvc) Verify(ctx context.Context, req *dtodomain.DomainVerifyReq) (*dtodomain.DomainActionResp, error) {
	scopeCtx, entity, err := svc.load(ctx, req.DomainID)
	if err != nil {
		return nil, err
	}
	method := req.Method
	if method == "" {
		method = model.DomainVerificationManual
	}
	err = svc.runActionOnLoaded(ctx, scopeCtx, entity, req.Operator.UserID, func(tx *tenantdomain.Tx) ([]string, error) {
		return tx.Verify(ctx, entity.ID, method, req.Token, time.Now())
	}, code.DomainUpdateError)
	if err != nil {
		return nil, err
	}
	return actionResp(entity.ID)
}

func (svc *domainSvc) TLSReady(ctx context.Context, req *dtodomain.DomainTLSReadyReq) (*dtodomain.DomainActionResp, error) {
	scopeCtx, entity, err := svc.load(ctx, req.DomainID)
	if err != nil {
		return nil, err
	}
	err = svc.runActionOnLoaded(ctx, scopeCtx, entity, req.Operator.UserID, func(tx *tenantdomain.Tx) ([]string, error) {
		return tx.TLSReady(ctx, entity.ID, time.Now())
	}, code.DomainUpdateError)
	if err != nil {
		return nil, err
	}
	return actionResp(entity.ID)
}

func (svc *domainSvc) Activate(ctx context.Context, req *dtodomain.DomainActivateReq) (*dtodomain.DomainActionResp, error) {
	scopeCtx, entity, err := svc.load(ctx, req.DomainID)
	if err != nil {
		return nil, err
	}
	err = svc.runActionOnLoaded(ctx, scopeCtx, entity, req.Operator.UserID, func(tx *tenantdomain.Tx) ([]string, error) {
		return tx.Activate(ctx, entity.ID)
	}, code.DomainUpdateError)
	if err != nil {
		return nil, err
	}
	return actionResp(entity.ID)
}

func (svc *domainSvc) Disable(ctx context.Context, req *dtodomain.DomainDisableReq) (*dtodomain.DomainActionResp, error) {
	scopeCtx, entity, err := svc.load(ctx, req.DomainID)
	if err != nil {
		return nil, err
	}
	err = svc.runActionOnLoaded(ctx, scopeCtx, entity, req.Operator.UserID, func(tx *tenantdomain.Tx) ([]string, error) {
		return tx.Disable(ctx, entity.ID)
	}, code.DomainUpdateError)
	if err != nil {
		return nil, err
	}
	return actionResp(entity.ID)
}

func (svc *domainSvc) Enable(ctx context.Context, req *dtodomain.DomainEnableReq) (*dtodomain.DomainActionResp, error) {
	scopeCtx, entity, err := svc.load(ctx, req.DomainID)
	if err != nil {
		return nil, err
	}
	err = svc.runActionOnLoaded(ctx, scopeCtx, entity, req.Operator.UserID, func(tx *tenantdomain.Tx) ([]string, error) {
		return tx.Enable(ctx, entity.ID)
	}, code.DomainUpdateError)
	if err != nil {
		return nil, err
	}
	return actionResp(entity.ID)
}

func (svc *domainSvc) Retire(ctx context.Context, req *dtodomain.DomainRetireReq) (*dtodomain.DomainActionResp, error) {
	scopeCtx, entity, err := svc.load(ctx, req.DomainID)
	if err != nil {
		return nil, err
	}
	err = svc.runActionOnLoaded(ctx, scopeCtx, entity, req.Operator.UserID, func(tx *tenantdomain.Tx) ([]string, error) {
		return tx.Retire(ctx, entity.ID, req.Reason, time.Now())
	}, code.DomainUpdateError)
	if err != nil {
		return nil, err
	}
	return actionResp(entity.ID)
}

func (svc *domainSvc) Release(ctx context.Context, req *dtodomain.DomainReleaseReq) (*dtodomain.DomainActionResp, error) {
	scopeCtx, entity, err := svc.load(ctx, req.DomainID)
	if err != nil {
		return nil, err
	}
	err = svc.runActionOnLoaded(ctx, scopeCtx, entity, req.Operator.UserID, func(tx *tenantdomain.Tx) ([]string, error) {
		return tx.Release(ctx, entity.ID, req.ApprovalTicket, time.Now())
	}, code.DomainUpdateError)
	if err != nil {
		return nil, err
	}
	return actionResp(entity.ID)
}
