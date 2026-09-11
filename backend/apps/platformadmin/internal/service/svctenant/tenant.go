package svctenant

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/audit"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/ark-iam/pkg/iam/tenant"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

type TenantSvc interface {
	Create(ctx *gin.Context, req *dtotenant.TenantCreateReq) (*dtotenant.TenantCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.TenantDeleteReq) error
	Update(ctx *gin.Context, req *dtotenant.TenantUpdateReq) error
	Detail(ctx *gin.Context, req *dtotenant.TenantDetailReq) (*dtotenant.TenantDetailResp, error)
	PageList(ctx *gin.Context, req *dtotenant.TenantPageListReq) (*dtotenant.TenantPageListResp, error)
}

type tenantSvc struct {
}

var _ TenantSvc = (*tenantSvc)(nil)

func NewTenantSvc() TenantSvc {
	return &tenantSvc{}
}

// Create 创建租户管理。
// 租户编码由服务端按统一规则自动生成（见 pkg/iam/tenant.GenerateCode），入参不接收编码；
// 编码创建后不可变更（Update 不修改 code）。
func (svc *tenantSvc) Create(ctx *gin.Context, req *dtotenant.TenantCreateReq) (*dtotenant.TenantCreateResp, error) {
	tenantCode, err := tenant.GenerateCode()
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantCreate] generate tenant code fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantCreateError)
	}
	userID := gincontext.GetUserIDString(ctx)
	tenantType := model.TenantType(req.Type)
	if tenantType != model.TenantTypeCustomer && tenantType != model.TenantTypePlatform {
		tenantType = model.TenantTypeCustomer
	}
	insertEntity := &model.TenantEntity{
		Code:      tenantCode,
		CreatedBy: userID,
		DbUser:    req.DbUser,
		Name:      req.Name,
		Status:    model.NormalizeTenantStatus(req.Status),
		Tag:       req.Tag,
		Type:      tenantType,
	}

	txErr := dbclient.IamDB(ctx).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := dao.NewTenantDao().WithTx(tx).Insert(ctx, insertEntity); err != nil {
			return err
		}
		// 每个租户创建时自动创建同名的根组织节点（组织树容器根）
		rootOrg := &model.OrganizationEntity{
			TenantID:  insertEntity.ID,
			ParentID:  "",
			Name:      req.Name,
			Status:    string(model.OrgNodeStatusActive),
			CreatedBy: userID,
		}
		if err := dao.NewOrganizationDao().WithTx(tx).Insert(ctx, rootOrg); err != nil {
			return err
		}
		// 根节点路径："/"+id，深度 1（ID 由 BeforeCreate 生成，需创建后补写）
		if err := dao.NewOrganizationDao().WithTx(tx).UpdateMap(ctx, rootOrg.ID, map[string]any{
			"org_path":  "/" + rootOrg.ID,
			"org_depth": 1,
		}); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svctenant.TenantCreate] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantCreateError)
	}
	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionTenantCreate,
		TenantID:   insertEntity.ID,
		Result:     "success",
		TargetType: "tenant",
		TargetID:   insertEntity.ID,
	})
	return &dtotenant.TenantCreateResp{
		TenantID: insertEntity.ID,
	}, nil
}

func (svc *tenantSvc) Delete(ctx *gin.Context, req *dtotenant.TenantDeleteReq) error {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantDelete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantDeleteError)
	}
	if tenantEntity == nil || tenantEntity.ID == "" {
		return code.GetError(code.TenantNotExistError)
	}

	userID := gincontext.GetUserIDString(ctx)

	if err := dao.NewTenantDao().Delete(ctx, req.TenantID, userID); err != nil {
		glog.Errorf(ctx, "[svctenant.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantDeleteError)
	}
	return nil
}

// Update 更新租户管理
func (svc *tenantSvc) Update(ctx *gin.Context, req *dtotenant.TenantUpdateReq) error {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantUpdate] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantUpdateError)
	}
	if tenantEntity == nil || tenantEntity.ID == "" {
		return code.GetError(code.TenantNotExistError)
	}

	userID := gincontext.GetUserIDString(ctx)
	tenantType := model.TenantType(req.Type)
	if tenantType != model.TenantTypeCustomer && tenantType != model.TenantTypePlatform {
		tenantType = model.TenantTypeCustomer
	}
	tenantStatus := model.NormalizeTenantStatus(req.Status)
	// 禁止挂起操作者自己所在的租户：挂起后该租户整体无法登录、本控制台随之失联，
	// 且产品内没有恢复路径（只能改库），属于不可逆自锁。
	if tenantStatus == model.TenantStatusSuspended && req.TenantID == gincontext.GetTenantIDString(ctx) {
		glog.Errorf(ctx, "[svctenant.TenantUpdate] refuse to suspend own tenant, tenantID:%s, req:%s", req.TenantID, gutil.ToJsonString(req))
		return code.GetError(code.TenantSuspendSelfForbiddenError)
	}
	updateMap := map[string]any{
		"db_user":    req.DbUser,
		"name":       req.Name,
		"status":     tenantStatus,
		"tag":        req.Tag,
		"type":       tenantType,
		"updated_by": userID,
	}
	if err := dao.NewTenantDao().UpdateMap(ctx, req.TenantID, updateMap); err != nil {
		glog.Errorf(ctx, "[svctenant.TenantUpdate] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantUpdateError)
	}
	// 由非挂起转为挂起：立即撤销该租户全部成员的 refresh token 与 SSO 会话，
	// 切断既有登录态（access token 依赖其短 TTL 自然过期）。撤销失败仅告警，
	// 不阻断挂起本身——租户状态已在库中生效，登录/签发令牌两个门禁会独立拦截。
	if tenantStatus == model.TenantStatusSuspended && tenantEntity.Status != model.TenantStatusSuspended {
		if rErr := tenant.RevokeMemberSessions(ctx.Request.Context(), req.TenantID); rErr != nil {
			glog.Errorf(ctx, "[svctenant.TenantUpdate] revoke member sessions fail, tenantID:%s, err:%v", req.TenantID, rErr)
		}
	}
	return nil
}

// Detail 根据id获取租户管理
func (svc *tenantSvc) Detail(ctx *gin.Context, req *dtotenant.TenantDetailReq) (*dtotenant.TenantDetailResp, error) {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantDetail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantGetDetailError)
	}
	if tenantEntity == nil || tenantEntity.ID == "" {
		return nil, code.GetError(code.TenantNotExistError)
	}
	resp := &dtotenant.TenantDetailResp{
		TenantID: tenantEntity.ID,
		TenantBaseInfo: objtenant.TenantBaseInfo{
			Code:   tenantEntity.Code,
			DbUser: tenantEntity.DbUser,
			Name:   tenantEntity.Name,
			Status: tenantEntity.Status,
			Tag:    tenantEntity.Tag,
			Type:   string(tenantEntity.Type),
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: tenantEntity.CreatedAt.Unix(),
			UpdatedAt: tenantEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

// PageList 分页获取租户管理列表。
// 列表需同时回传创建时间与更新时间（前端两列都展示），故两个时间字段均需赋值。
// 状态筛选走白名单校验：非法值直接报错，避免静默返回"看起来正常"的错误集合。
func (svc *tenantSvc) PageList(ctx *gin.Context, req *dtotenant.TenantPageListReq) (*dtotenant.TenantPageListResp, error) {
	switch req.Status {
	case "", model.TenantStatusActive, model.TenantStatusSuspended:
	default:
		glog.Errorf(ctx, "[svctenant.TenantPageList] invalid status filter, status:%s, req:%s", req.Status, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantPageListStatusInvalidError)
	}
	cond := &dao.TenantCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		Keyword: strings.TrimSpace(req.Name),
		Status:  req.Status,
	}
	tenantEntityList, total, err := dao.NewTenantDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantPageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantGetPageListError)
	}
	list := make([]dtotenant.TenantPageListItem, 0, len(tenantEntityList))
	for _, v := range tenantEntityList {
		list = append(list, dtotenant.TenantPageListItem{
			TenantID: v.ID,
			TenantBaseInfo: objtenant.TenantBaseInfo{
				Code:   v.Code,
				DbUser: v.DbUser,
				Name:   v.Name,
				Status: v.Status,
				Tag:    v.Tag,
				Type:   string(v.Type),
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				CreatedAt: v.CreatedAt.Unix(),
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtotenant.TenantPageListResp{
		List:  list,
		Total: total,
	}, nil
}
