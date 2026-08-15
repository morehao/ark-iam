package svctenant

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/gctx"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/ark-iam/pkg/iam/svcaudit"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TenantSvc interface {
	Create(ctx *gin.Context, req *dtotenant.TenantCreateReq) (*dtotenant.TenantCreateResp, error)
	CreateTenantAsOwner(ctx *gin.Context, req *dtotenant.TenantCreateAsOwnerReq) (*dtotenant.TenantCreateAsOwnerResp, error)
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

// generateTenantCode 生成全局唯一、非空的租户编码，用于避免空 code 撞到唯一索引。
func generateTenantCode() string {
	return "tenant-" + uuid.NewString()
}

// Create 创建租户管理
func (svc *tenantSvc) Create(ctx *gin.Context, req *dtotenant.TenantCreateReq) (*dtotenant.TenantCreateResp, error) {
	tenantCode := req.Code
	if tenantCode == "" {
		// 保证 code 非空且唯一，避免撞租户表唯一索引（MySQL 仅允许一条空字符串）
		tenantCode = generateTenantCode()
	}
	tenantType := model.TenantType(req.Type)
	if tenantType != model.TenantTypeCustomer && tenantType != model.TenantTypePlatform {
		tenantType = model.TenantTypeCustomer
	}
	insertEntity := &model.TenantEntity{
		Code:        tenantCode,
		DbUser:      req.DbUser,
		IsSuspended: req.IsSuspended,
		Name:        req.Name,
		Tag:         req.Tag,
		Type:        tenantType,
	}

	userID := gctx.GetUserID(ctx)
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
	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionTenantCreate,
		TenantID:   insertEntity.ID,
		Result:     "success",
		TargetType: "tenant",
		TargetID:   insertEntity.ID,
	})
	return &dtotenant.TenantCreateResp{
		TenantID: insertEntity.ID,
	}, nil
}

// CreateTenantAsOwner 0租户自然人自助创建租户并成为租户 owner
func (svc *tenantSvc) CreateTenantAsOwner(ctx *gin.Context, req *dtotenant.TenantCreateAsOwnerReq) (*dtotenant.TenantCreateAsOwnerResp, error) {
	userID := gctx.GetUserID(ctx)
	now := time.Now()

	// 授权闸门：仅允许 0 租户自然人，且目标应用的 tenant_policy.allowPersonCreateTenant 为 true
	if err := svc.checkCreateTenantAsOwnerGate(ctx, req); err != nil {
		return nil, err
	}

	var tenantID string
	txErr := dbclient.IamDB(ctx).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tenant := &model.TenantEntity{
			Code:      generateTenantCode(),
			Name:      req.Name,
			Type:      model.TenantTypeCustomer,
			CreatedBy: userID,
		}
		if err := dao.NewTenantDao().WithTx(tx).Insert(ctx, tenant); err != nil {
			return err
		}

		user := &model.UserEntity{
			TenantID:   tenant.ID,
			PersonID:   req.PersonID,
			Name:       req.Name,
			Profile:    json.RawMessage("{}"),
			CustomData: json.RawMessage("{}"),
			IsOwner:    true,
			JoinedAt:   &now,
			CreatedBy:  req.PersonID,
		}
		if err := dao.NewUserDao().WithTx(tx).Insert(ctx, user); err != nil {
			return err
		}

		if req.AppID != "" {
			app := &model.TenantApplicationEntity{
				TenantID:     tenant.ID,
				AppID:        req.AppID,
				Status:       "enable",
				Config:       datatypes.JSON("{}"),
				GrantedScope: datatypes.JSON("[]"),
				CreatedBy:    userID,
			}
			if err := dao.NewTenantApplicationDao().WithTx(tx).Insert(ctx, app); err != nil {
				return err
			}
		}

		// 每个租户创建时自动创建同名的根组织节点（组织树容器根）
		rootOrg := &model.OrganizationEntity{
			TenantID:  tenant.ID,
			ParentID:  "",
			Name:      req.Name,
			Status:    string(model.OrgNodeStatusActive),
			CreatedBy: req.PersonID,
		}
		if err := dao.NewOrganizationDao().WithTx(tx).Insert(ctx, rootOrg); err != nil {
			return err
		}
		if err := dao.NewOrganizationDao().WithTx(tx).UpdateMap(ctx, rootOrg.ID, map[string]any{
			"org_path":  "/" + rootOrg.ID,
			"org_depth": 1,
		}); err != nil {
			return err
		}

		tenantID = tenant.ID
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svctenant.CreateTenantAsOwner] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.TenantCreateError)
	}

	svcaudit.WriteAudit(ctx, svcaudit.AuditEntry{
		Action:     svcaudit.ActionTenantCreate,
		TenantID:   tenantID,
		Result:     "success",
		TargetType: "tenant",
		TargetID:   tenantID,
	})
	return &dtotenant.TenantCreateAsOwnerResp{
		TenantID: tenantID,
	}, nil
}

// checkCreateTenantAsOwnerGate 校验自助创建租户的授权闸门：
//  1. 自然人必须处于 0 租户状态（在真实租户下不存在 user 记录）
//  2. 必须提供有效应用 AppID，且该应用的 tenant_policy.allowPersonCreateTenant 为 true
func (svc *tenantSvc) checkCreateTenantAsOwnerGate(ctx *gin.Context, req *dtotenant.TenantCreateAsOwnerReq) error {
	users, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{PersonID: req.PersonID})
	if err != nil {
		glog.Errorf(ctx, "[svctenant.CreateTenantAsOwner] query users by person fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantCreateAsOwnerForbiddenError)
	}
	for _, u := range users {
		if u.TenantID != "" {
			glog.Errorf(ctx, "[svctenant.CreateTenantAsOwner] person already has a tenant, req:%s", gutil.ToJsonString(req))
			return code.GetError(code.TenantCreateAsOwnerForbiddenError)
		}
	}

	if req.AppID == "" {
		glog.Errorf(ctx, "[svctenant.CreateTenantAsOwner] appID is required for self-service tenant creation, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.TenantCreateAsOwnerForbiddenError)
	}

	app, err := dao.NewApplicationDao().GetByID(ctx, req.AppID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.CreateTenantAsOwner] query application fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantCreateAsOwnerForbiddenError)
	}
	if app == nil || app.ID == "" {
		glog.Errorf(ctx, "[svctenant.CreateTenantAsOwner] application not found, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.TenantCreateAsOwnerForbiddenError)
	}

	var policy model.TenantPolicy
	if len(app.TenantPolicy) == 0 {
		glog.Errorf(ctx, "[svctenant.CreateTenantAsOwner] application has empty tenant policy, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.TenantCreateAsOwnerForbiddenError)
	}
	if err := json.Unmarshal(app.TenantPolicy, &policy); err != nil || policy.AllowPersonCreateTenant == nil || !*policy.AllowPersonCreateTenant {
		glog.Errorf(ctx, "[svctenant.CreateTenantAsOwner] application policy does not allow person create tenant, req:%s", gutil.ToJsonString(req))
		return code.GetError(code.TenantCreateAsOwnerForbiddenError)
	}
	return nil
}

// Delete 删除租户管理
func (svc *tenantSvc) Delete(ctx *gin.Context, req *dtotenant.TenantDeleteReq) error {
	tenantEntity, err := dao.NewTenantDao().GetByID(ctx, req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svctenant.TenantDelete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantDeleteError)
	}
	if tenantEntity == nil || tenantEntity.ID == "" {
		return code.GetError(code.TenantNotExistError)
	}

	userID := gctx.GetUserID(ctx)

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

	userID := gctx.GetUserID(ctx)
	tenantType := model.TenantType(req.Type)
	if tenantType != model.TenantTypeCustomer && tenantType != model.TenantTypePlatform {
		tenantType = model.TenantTypeCustomer
	}
	updateMap := map[string]any{
		"db_user":      req.DbUser,
		"is_suspended": req.IsSuspended,
		"name":         req.Name,
		"tag":          req.Tag,
		"type":         tenantType,
		"updated_by":   userID,
	}
	if err := dao.NewTenantDao().UpdateMap(ctx, req.TenantID, updateMap); err != nil {
		glog.Errorf(ctx, "[svctenant.TenantUpdate] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.TenantUpdateError)
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
			Code:        tenantEntity.Code,
			DbUser:      tenantEntity.DbUser,
			IsSuspended: tenantEntity.IsSuspended,
			Name:        tenantEntity.Name,
			Tag:         tenantEntity.Tag,
			Type:        string(tenantEntity.Type),
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: tenantEntity.CreatedAt.Unix(),
			UpdatedAt: tenantEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

// PageList 分页获取租户管理列表
func (svc *tenantSvc) PageList(ctx *gin.Context, req *dtotenant.TenantPageListReq) (*dtotenant.TenantPageListResp, error) {
	cond := &dao.TenantCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
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
				Code:        v.Code,
				DbUser:      v.DbUser,
				IsSuspended: v.IsSuspended,
				Name:        v.Name,
				Tag:         v.Tag,
				Type:        string(v.Type),
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtotenant.TenantPageListResp{
		List:  list,
		Total: total,
	}, nil
}
