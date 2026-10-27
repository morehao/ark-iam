package svcdomain

import (
	"context"
	"strings"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtodomain"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type DomainSvc interface {
	Create(ctx context.Context, req *dtodomain.DomainCreateReq) (*dtodomain.DomainCreateResp, error)
	Update(ctx context.Context, req *dtodomain.DomainUpdateReq) error
	Detail(ctx context.Context, req *dtodomain.DomainDetailReq) (*dtodomain.DomainDetailResp, error)
	PageList(ctx context.Context, req *dtodomain.DomainPageListReq) (*dtodomain.DomainPageListResp, error)
	Delete(ctx context.Context, req *dtodomain.DomainDeleteReq) error
}

type domainSvc struct{}

var _ DomainSvc = (*domainSvc)(nil)

func NewDomainSvc() DomainSvc {
	return &domainSvc{}
}

// isValidDomainVerificationStatus 校验域名验证状态：空值表示「本次不修改」，非空必须命中白名单常量。
// 校验归 service（AGENTS.md 硬规则 3）：DTO 绑定的是前端传来的原始字符串，非法值在此拦截。
func isValidDomainVerificationStatus(status model.DomainVerificationStatus) bool {
	switch status {
	case "", model.DomainVerificationUnverified, model.DomainVerificationVerified:
		return true
	default:
		return false
	}
}

func (svc *domainSvc) Create(ctx context.Context, req *dtodomain.DomainCreateReq) (*dtodomain.DomainCreateResp, error) {
	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		return nil, code.GetError(code.DomainCreateError)
	}
	if len(domain) > 256 {
		return nil, code.GetError(code.DomainCreateError)
	}

	tenantID := dbclient.TenantIDFromContext(ctx)

	repo := dao.NewDomainDao()
	existing, err := repo.GetByTenantAndDomain(ctx, tenantID, domain)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.Create] GetByTenantAndDomain fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DomainCreateError)
	}
	if existing != nil {
		return nil, code.GetError(code.DomainAlreadyExistError)
	}

	entity := &model.DomainEntity{
		TenantID:           tenantID,
		Domain:             domain,
		VerificationStatus: model.DomainVerificationUnverified,
		CreatedBy:          req.Operator.UserID,
		UpdatedBy:          req.Operator.UserID,
	}
	if err := repo.Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcdomain.Create] Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DomainCreateError)
	}
	return &dtodomain.DomainCreateResp{ID: entity.ID}, nil
}

func (svc *domainSvc) PageList(ctx context.Context, req *dtodomain.DomainPageListReq) (*dtodomain.DomainPageListResp, error) {
	tenantID := dbclient.TenantIDFromContext(ctx)

	cond := &dao.DomainCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: tenantID,
		Domain:   req.Domain,
	}

	repo := dao.NewDomainDao()
	list, total, err := repo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.PageList] GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DomainGetPageListError)
	}

	items := make([]dtodomain.DomainPageListItem, 0, len(list))
	for _, v := range list {
		items = append(items, dtodomain.DomainPageListItem{
			ID:                 v.ID,
			Domain:             v.Domain,
			VerificationStatus: v.VerificationStatus,
			CreatedAt:          v.CreatedAt.Unix(),
			UpdatedAt:          v.UpdatedAt.Unix(),
		})
	}
	return &dtodomain.DomainPageListResp{List: items, Total: total}, nil
}

func (svc *domainSvc) Delete(ctx context.Context, req *dtodomain.DomainDeleteReq) error {
	tenantID := dbclient.TenantIDFromContext(ctx)

	repo := dao.NewDomainDao()
	entity, err := repo.GetByID(ctx, req.DomainID)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.Delete] GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DomainDeleteError)
	}
	if entity == nil || entity.ID == "" || entity.TenantID != tenantID {
		return code.GetError(code.DomainNotExistError)
	}

	userID := req.Operator.UserID
	if err := repo.Delete(ctx, req.DomainID, userID); err != nil {
		glog.Errorf(ctx, "[svcdomain.Delete] Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DomainDeleteError)
	}
	return nil
}

func (svc *domainSvc) Update(ctx context.Context, req *dtodomain.DomainUpdateReq) error {
	tenantID := dbclient.TenantIDFromContext(ctx)

	repo := dao.NewDomainDao()
	entity, err := repo.GetByID(ctx, req.DomainID)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.Update] GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DomainUpdateError)
	}
	if entity == nil || entity.ID == "" || entity.TenantID != tenantID {
		return code.GetError(code.DomainNotExistError)
	}

	if !isValidDomainVerificationStatus(req.VerificationStatus) {
		glog.Errorf(ctx, "[svcdomain.Update] 非法验证状态, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.DomainUpdateError)
	}
	updateMap := map[string]any{
		"updated_by": req.Operator.UserID,
	}
	if req.Domain != "" {
		updateMap["domain"] = req.Domain
	}
	// 验证状态留空表示不修改：不写该列，避免把状态覆盖为空串
	if req.VerificationStatus != "" {
		updateMap["verification_status"] = req.VerificationStatus
	}

	if err := repo.UpdateMap(ctx, req.DomainID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcdomain.Update] UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.DomainUpdateError)
	}
	return nil
}

func (svc *domainSvc) Detail(ctx context.Context, req *dtodomain.DomainDetailReq) (*dtodomain.DomainDetailResp, error) {
	tenantID := dbclient.TenantIDFromContext(ctx)

	repo := dao.NewDomainDao()
	entity, err := repo.GetByID(ctx, req.DomainID)
	if err != nil {
		glog.Errorf(ctx, "[svcdomain.Detail] GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.DomainDetailError)
	}
	if entity == nil || entity.ID == "" || entity.TenantID != tenantID {
		return nil, code.GetError(code.DomainNotExistError)
	}

	return &dtodomain.DomainDetailResp{
		ID:                 entity.ID,
		Domain:             entity.Domain,
		VerificationStatus: entity.VerificationStatus,
		CreatedAt:          entity.CreatedAt.Unix(),
		UpdatedAt:          entity.UpdatedAt.Unix(),
	}, nil
}
