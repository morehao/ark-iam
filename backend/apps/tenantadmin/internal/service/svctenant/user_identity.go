package svctenant

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

// userIdentityPageSize 用户身份列表不分页展示，取足够大的单页容量。
const userIdentityPageSize = 100

// UserIdentitySvc 租户内用户的第三方身份子资源。
// 身份实体按自然人（person_id）归属，没有自己的 tenant_id，因此每个操作都要先借道
// “该自然人在本租户内是否有用户”完成租户可见性校验，避免跨租户读写。
type UserIdentitySvc interface {
	ListByUser(ctx *gin.Context, req *dtotenant.UserIdentityListReq) (*dtotenant.UserIdentityListResp, error)
	Create(ctx *gin.Context, req *dtotenant.UserIdentityCreateReq) (*dtotenant.UserIdentityCreateResp, error)
	Delete(ctx *gin.Context, req *dtotenant.UserIdentityDeleteReq) error
}

type userIdentitySvc struct{}

var _ UserIdentitySvc = (*userIdentitySvc)(nil)

func NewUserIdentitySvc() UserIdentitySvc {
	return &userIdentitySvc{}
}

func (svc *userIdentitySvc) ListByUser(ctx *gin.Context, req *dtotenant.UserIdentityListReq) (*dtotenant.UserIdentityListResp, error) {
	userEntity, err := resolveTenantUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if userEntity.PersonID == "" {
		return nil, code.GetError(code.UserIdentityNotExistError)
	}

	entityList, total, err := dao.NewUserIdentityDao().GetPageListByCond(ctx, &dao.UserIdentityCond{
		BaseCond: &gormdao.BaseCond{Page: 1, PageSize: userIdentityPageSize},
		PersonID: userEntity.PersonID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcuserIdentity.ListByUser] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityGetPageListError)
	}

	list := make([]dtotenant.UserIdentityItem, 0, len(entityList))
	for _, v := range entityList {
		var detail any
		if uErr := json.Unmarshal(v.Detail, &detail); uErr != nil {
			// 明细反序列化失败不阻断列表：置空明细并告警，避免单条脏数据隐藏整个身份列表。
			glog.Warnf(ctx, "[svcuserIdentity.ListByUser] json.Unmarshal detail fail, identityID:%s, err:%v", v.ID, uErr)
		}
		list = append(list, dtotenant.UserIdentityItem{
			UserIdentityID: v.ID,
			Issuer:         v.Issuer,
			IdentityID:     v.ExternalSubject,
			Detail:         detail,
			CreatedAt:      v.CreatedAt.Unix(),
			UpdatedAt:      v.UpdatedAt.Unix(),
		})
	}
	return &dtotenant.UserIdentityListResp{List: list, Total: total}, nil
}

func (svc *userIdentitySvc) Create(ctx *gin.Context, req *dtotenant.UserIdentityCreateReq) (*dtotenant.UserIdentityCreateResp, error) {
	userEntity, err := resolveTenantUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if userEntity.PersonID == "" {
		return nil, code.GetError(code.UserIdentityCreateError)
	}

	detailJSON, mErr := json.Marshal(req.Detail)
	if mErr != nil {
		glog.Errorf(ctx, "[svcuserIdentity.Create] json.Marshal detail fail, err:%v, req:%s", mErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityCreateError)
	}

	entity := &model.UserIdentityEntity{
		PersonID:        userEntity.PersonID,
		Issuer:          req.Issuer,
		ExternalSubject: req.IdentityID,
		Detail:          detailJSON,
		CreatedBy:       gincontext.GetUserIDString(ctx),
	}
	// 同一外部主体（issuer + external_subject）全局唯一，由部分唯一索引兜底，
	// 重复绑定在此以创建失败返回。
	if err := dao.NewUserIdentityDao().Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcuserIdentity.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserIdentityCreateError)
	}
	return &dtotenant.UserIdentityCreateResp{UserIdentityID: entity.ID}, nil
}

func (svc *userIdentitySvc) Delete(ctx *gin.Context, req *dtotenant.UserIdentityDeleteReq) error {
	// 先校验 path 上的用户在不在本租户，再要求身份确实归属该用户的自然人：
	// 两道校验合起来才能保证「租户内、且与该用户匹配」，不允许拿本租户其他用户的身份ID解绑。
	userEntity, err := resolveTenantUser(ctx, req.UserID)
	if err != nil {
		return err
	}
	entity, err := dao.NewUserIdentityDao().GetByID(ctx, req.UserIdentityID)
	if err != nil {
		glog.Errorf(ctx, "[svcuserIdentity.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityDeleteError)
	}
	if entity == nil || entity.ID == "" || entity.PersonID == "" || entity.PersonID != userEntity.PersonID {
		return code.GetError(code.UserIdentityNotExistError)
	}
	if err := dao.NewUserIdentityDao().Delete(ctx, req.UserIdentityID, gincontext.GetUserIDString(ctx)); err != nil {
		glog.Errorf(ctx, "[svcuserIdentity.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserIdentityDeleteError)
	}
	return nil
}

// resolveTenantUser 校验用户存在且属于当前登录租户，返回该租户内的用户实体。
// 系统错误与业务边界（不属于本租户）分开处理：前者记日志并返回功能级错误码。
func resolveTenantUser(ctx *gin.Context, userID string) (*model.UserEntity, error) {
	userEntity, err := dao.NewUserDao().GetByID(ctx, userID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.resolveTenantUser] dao GetByID fail, err:%v, userID:%s", err, userID)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != gincontext.GetTenantIDString(ctx) {
		return nil, code.GetError(code.UserNotExistError)
	}
	return userEntity, nil
}
