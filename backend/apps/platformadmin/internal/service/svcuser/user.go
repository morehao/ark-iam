package svcuser

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/pkg/iam/object/objuser"
	"github.com/morehao/ark-iam/platformadmin/internal/dto/dtouser"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

// UserSvc 平台排查视角：跨租户用户目录只读 + 挂起/恢复 + 重置密码 + 登录日志子资源。
type UserSvc interface {
	Detail(ctx *gin.Context, req *dtouser.UserDetailReq) (*dtouser.UserDetailResp, error)
	PageList(ctx *gin.Context, req *dtouser.UserPageListReq) (*dtouser.UserPageListResp, error)
	UpdatePassword(ctx *gin.Context, req *dtouser.UserPasswordUpdateReq) error
	UpdateStatus(ctx *gin.Context, req *dtouser.UserStatusUpdateReq) error
	GetUserLoginLogByUser(ctx *gin.Context, req *dtouser.UserLoginLogByUserReq) (*dtouser.UserLoginLogPageListResp, error)
}

type userSvc struct {
}

var _ UserSvc = (*userSvc)(nil)

func NewUserSvc() UserSvc {
	return &userSvc{}
}

// loadPersonMap 批量加载自然人信息（username/email/phone 在 person 表），
// 用于用户列表/详情关联展示。查询失败仅告警并返回空 map，不阻断主流程。
func (svc *userSvc) loadPersonMap(ctx context.Context, personIDs []string) map[string]*model.PersonEntity {
	result := make(map[string]*model.PersonEntity)
	if len(personIDs) == 0 {
		return result
	}
	personDao := dao.NewPersonDao()
	for _, id := range personIDs {
		if id == "" {
			continue
		}
		person, err := personDao.GetByID(ctx, id)
		if err != nil {
			glog.Warnf(ctx, "[svcuser.loadPersonMap] person GetByID fail, personID:%s, err:%v", id, err)
			continue
		}
		if person != nil && person.ID != "" {
			result[id] = person
		}
	}
	return result
}

func (svc *userSvc) Detail(ctx *gin.Context, req *dtouser.UserDetailReq) (*dtouser.UserDetailResp, error) {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != gincontext.GetTenantIDString(ctx) {
		return nil, code.GetError(code.UserNotExistError)
	}

	var profile any
	if err := json.Unmarshal(userEntity.Profile, &profile); err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] json.Unmarshal profile fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	var customData any
	if err := json.Unmarshal(userEntity.CustomData, &customData); err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] json.Unmarshal customData fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}

	personMap := svc.loadPersonMap(ctx, []string{userEntity.PersonID})
	person := personMap[userEntity.PersonID]
	if person == nil {
		person = &model.PersonEntity{}
	}

	resp := &dtouser.UserDetailResp{
		UserID: userEntity.ID,
		UserBaseInfo: objuser.UserBaseInfo{
			TenantID:     userEntity.TenantID,
			Username:     model.DerefStr(person.Username),
			PrimaryEmail: model.DerefStr(person.PrimaryEmail),
			PrimaryPhone: model.DerefStr(person.PrimaryPhone),
			Name:         userEntity.Name,
			Avatar:       userEntity.Avatar,
			Profile:      profile,
			CustomData:   customData,
			IsSuspended:  userEntity.IsSuspended,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: userEntity.CreatedAt.Unix(),
			UpdatedAt: userEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *userSvc) PageList(ctx *gin.Context, req *dtouser.UserPageListReq) (*dtouser.UserPageListResp, error) {
	userRepo := dao.NewUserDao()
	cond := &dao.UserCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:    gincontext.GetTenantIDString(ctx),
		Name:        req.Name,
		IsSuspended: req.IsSuspended,
	}
	userEntityList, total, err := userRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetPageListError)
	}

	list := make([]dtouser.UserPageListItem, 0, len(userEntityList))
	personIDs := make([]string, 0, len(userEntityList))
	for _, v := range userEntityList {
		if v.PersonID != "" {
			personIDs = append(personIDs, v.PersonID)
		}
	}
	personMap := svc.loadPersonMap(ctx, personIDs)
	for _, v := range userEntityList {
		var profile any
		if err := json.Unmarshal(v.Profile, &profile); err != nil {
			glog.Errorf(ctx, "[svcuser.PageList] json.Unmarshal profile fail, err:%v", err)
			continue
		}
		var customData any
		if err := json.Unmarshal(v.CustomData, &customData); err != nil {
			glog.Errorf(ctx, "[svcuser.PageList] json.Unmarshal customData fail, err:%v", err)
			continue
		}
		person := personMap[v.PersonID]
		if person == nil {
			person = &model.PersonEntity{}
		}
		list = append(list, dtouser.UserPageListItem{
			UserID: v.ID,
			UserBaseInfo: objuser.UserBaseInfo{
				TenantID:     v.TenantID,
				Username:     model.DerefStr(person.Username),
				PrimaryEmail: model.DerefStr(person.PrimaryEmail),
				PrimaryPhone: model.DerefStr(person.PrimaryPhone),
				Name:         v.Name,
				Avatar:       v.Avatar,
				Profile:      profile,
				CustomData:   customData,
				IsSuspended:  v.IsSuspended,
			},
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *userSvc) UpdatePassword(ctx *gin.Context, req *dtouser.UserPasswordUpdateReq) error {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UpdatePassword] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != gincontext.GetTenantIDString(ctx) {
		return code.GetError(code.UserNotExistError)
	}
	if userEntity.PersonID == "" {
		return code.GetError(code.UserNotExistError)
	}

	password := req.Password
	if password == "" && req.PasswordEncrypted != "" {
		// 兼容旧契约：接受已哈希的密码（原样落库）
		password = req.PasswordEncrypted
	} else if password != "" {
		hash, hashErr := gcrypto.GeneratePasswordHash(password)
		if hashErr != nil {
			glog.Errorf(ctx, "[svcuser.UpdatePassword] GeneratePasswordHash fail, err:%v", hashErr)
			return code.GetError(code.PasswordHashError)
		}
		password = hash
	}
	if password == "" {
		return code.GetError(code.PasswordValidationError)
	}

	userID := gincontext.GetUserIDString(ctx)
	if err := dao.NewPersonDao().UpdateMap(ctx, userEntity.PersonID, map[string]any{
		"password_encrypted": password,
		"password_method":    "bcrypt",
		"updated_by":         userID,
	}); err != nil {
		glog.Errorf(ctx, "[svcuser.UpdatePassword] person UpdateMap fail, err:%v", err)
		return code.GetError(code.UserUpdateError)
	}
	return nil
}

func (svc *userSvc) UpdateStatus(ctx *gin.Context, req *dtouser.UserStatusUpdateReq) error {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateStatus] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != gincontext.GetTenantIDString(ctx) {
		return code.GetError(code.UserNotExistError)
	}

	userID := gincontext.GetUserIDString(ctx)
	updateMap := map[string]any{
		"is_suspended": req.IsSuspended,
		"updated_by":   userID,
	}
	if err := dao.NewUserDao().UpdateMap(ctx, req.UserID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateStatus] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	return nil
}

func (svc *userSvc) GetUserLoginLogByUser(ctx *gin.Context, req *dtouser.UserLoginLogByUserReq) (*dtouser.UserLoginLogPageListResp, error) {
	userLoginLogRepo := dao.NewUserLoginLogDao()
	cond := &dao.UserLoginLogCond{
		TenantID: gincontext.GetTenantIDString(ctx),
		UserID:   req.UserID,
	}
	userLoginLogEntityList, total, err := userLoginLogRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.GetUserLoginLogByUser] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserLoginLogGetPageListError)
	}

	list := make([]dtouser.UserLoginLogPageListItem, 0, len(userLoginLogEntityList))
	for _, v := range userLoginLogEntityList {
		list = append(list, dtouser.UserLoginLogPageListItem{
			UserLoginLogID: v.ID,
			TenantID:       v.TenantID,
			UserID:         v.UserID,
			LoginIP:        v.LoginIP,
			UserAgent:      v.UserAgent,
			LoginTime:      v.LoginTime.Unix(),
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserLoginLogPageListResp{
		List:  list,
		Total: total,
	}, nil
}
