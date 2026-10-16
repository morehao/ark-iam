package svcaudit

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtoaudit"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

const (
	// auditLogMaxPageSize 单页上限：审计行含 text 详情，放大页容易把主库拖慢（方案 R6）。
	auditLogMaxPageSize = 100
	// auditLogDefaultPageSize 未传 pageSize 时的默认值。
	auditLogDefaultPageSize = 20
	// auditLogDefaultWindow 未传 startTime 时的默认时间窗（近 7 天），避免无界扫描全表。
	auditLogDefaultWindow = 7 * 24 * time.Hour
)

// auditLogOrderBy 审计列表固定按时间倒序；同一秒内按 id 倒序，保证 OFFSET 翻页稳定。
const auditLogOrderBy = "created_at DESC, id DESC"

type AuditLogSvc interface {
	PageList(ctx *gin.Context, req *dtoaudit.AuditLogPageListReq) (*dtoaudit.AuditLogPageListResp, error)
}

type auditLogSvc struct {
}

var _ AuditLogSvc = (*auditLogSvc)(nil)

func NewAuditLogSvc() AuditLogSvc {
	return &auditLogSvc{}
}

// isValidAuditAction / isValidAuditResult 白名单校验枚举入参（空串=不筛选，由调用方放行）。
// 请求来自前端（原始字符串），非法值必须在此拦截，不能静默当作"无匹配"。
func isValidAuditAction(action model.AuditAction) bool {
	switch action {
	case model.AuditActionLogin,
		model.AuditActionLogout,
		model.AuditActionTenantSwitch,
		model.AuditActionTenantCreate,
		model.AuditActionTenantAdminPasswordReset,
		model.AuditActionApplicationCreate,
		model.AuditActionApplicationClientCreate,
		model.AuditActionApplicationClientCreateSecret,
		model.AuditActionApiKeyCreate,
		model.AuditActionApiKeyRevoke,
		model.AuditActionInstallationInitialize:
		return true
	default:
		return false
	}
}

func isValidAuditResult(result model.AuditResult) bool {
	switch result {
	case model.AuditResultSuccess, model.AuditResultFailure:
		return true
	default:
		return false
	}
}

// PageList 平台视角分页查询审计日志（只读）。
//
// 隔离口径（关键）：audit_log 有 tenant_id 且**不在** tenantScopeSkipTables，
// 若不显式声明作用域，插件会注入"当前租户"，平台视图将看不到别家审计；
// 而平台审计查询的语义本就是跨租户，因此显式声明 dbclient.CrossTenantContext——
// 不用"ctx 碰巧没有作用域"表达跨租户（那样在 fail-closed 下会直接报错）。
// 过滤到具体租户时仍是"显式跨租户作用域 + tenant_id 条件"，二者不是同一件事。
func (svc *auditLogSvc) PageList(ctx *gin.Context, req *dtoaudit.AuditLogPageListReq) (*dtoaudit.AuditLogPageListResp, error) {
	if req.Action != "" && !isValidAuditAction(req.Action) {
		glog.Errorf(ctx, "[svcaudit.PageList] 非法审计动作, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.AuditLogPageListParamInvalidError)
	}
	if req.Result != "" && !isValidAuditResult(req.Result) {
		glog.Errorf(ctx, "[svcaudit.PageList] 非法审计结果, req:%s", gutil.ToJsonString(req))
		return nil, code.GetError(code.AuditLogPageListParamInvalidError)
	}

	// 分页兜底：GetPageListByCond 在 page/pageSize 非正数时**不分页、返回全部**，
	// 审计表只增不减，绝不能让它无界返回，因此这里补齐默认值并压到单页上限。
	page, pageSize := req.Page, req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = auditLogDefaultPageSize
	}
	if pageSize > auditLogMaxPageSize {
		pageSize = auditLogMaxPageSize
	}

	// 时间窗兜底：不传 startTime 时默认近 7 天（R6）。
	startTime := req.StartTime
	if startTime <= 0 {
		startTime = time.Now().Add(-auditLogDefaultWindow).Unix()
	}

	cond := &dao.AuditLogCond{
		BaseCond: &gormdao.BaseCond{
			Page:     page,
			PageSize: pageSize,
			// 时间区间复用 gormdao.BaseCond 已有的 created_at >= / <= 注入
			// （同一谓词只保留一份实现，避免在 AuditLogCond 再写一份重复条件）。
			CreatedAtStart: startTime,
			CreatedAtEnd:   req.EndTime,
			OrderField:     auditLogOrderBy,
		},
		TenantID: req.TenantID,
		PersonID: req.ActorPersonID,
		Action:   req.Action,
		Result:   req.Result,
	}
	entityList, total, err := dao.NewAuditLogDao().GetPageListByCond(dbclient.CrossTenantContext(ctx), cond)
	if err != nil {
		glog.Errorf(ctx, "[svcaudit.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.AuditLogGetPageListError)
	}

	list := make([]dtoaudit.AuditLogPageListItem, 0, len(entityList))
	for _, v := range entityList {
		list = append(list, dtoaudit.AuditLogPageListItem{
			AuditLogID:    v.ID,
			ActorPersonID: v.ActorPersonID,
			ActorUserID:   v.ActorUserID,
			TenantID:      v.TenantID,
			ClientID:      v.ClientID,
			Action:        v.Action,
			TargetType:    v.TargetType,
			TargetID:      v.TargetID,
			Result:        v.Result,
			IP:            v.IP,
			UserAgent:     v.UserAgent,
			Detail:        v.Detail,
			CreatedAt:     v.CreatedAt.Unix(),
		})
	}
	return &dtoaudit.AuditLogPageListResp{
		List:  list,
		Total: total,
	}, nil
}
