package svcuser

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objuser"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
)

type UserSvc interface {
	Create(ctx *gin.Context, req *dtouser.UserCreateReq) (*dtouser.UserCreateResp, error)
	Delete(ctx *gin.Context, req *dtouser.UserDeleteReq) error
	Update(ctx *gin.Context, req *dtouser.UserUpdateReq) error
	Detail(ctx *gin.Context, req *dtouser.UserDetailReq) (*dtouser.UserDetailResp, error)
	PageList(ctx *gin.Context, req *dtouser.UserPageListReq) (*dtouser.UserPageListResp, error)
	UpdatePassword(ctx *gin.Context, req *dtouser.UserPasswordUpdateReq) error
	UpdateStatus(ctx *gin.Context, req *dtouser.UserStatusUpdateReq) error
	DetailUserLoginLog(ctx *gin.Context, req *dtouser.UserLoginLogDetailReq) (*dtouser.UserLoginLogDetailResp, error)
	PageListUserLoginLog(ctx *gin.Context, req *dtouser.UserLoginLogPageListReq) (*dtouser.UserLoginLogPageListResp, error)
	GetUserLoginLogByUser(ctx *gin.Context, req *dtouser.UserLoginLogByUserReq) (*dtouser.UserLoginLogPageListResp, error)
	CreateUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationCreateReq) (*dtouser.UserDepartmentRelationCreateResp, error)
	DeleteUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationDeleteReq) error
	UpdateUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationUpdateReq) error
	DetailUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationDetailReq) (*dtouser.UserDepartmentRelationDetailResp, error)
	PageListUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationPageListReq) (*dtouser.UserDepartmentRelationPageListResp, error)
	GetUserDepartmentRelationByUser(ctx *gin.Context, req *dtouser.UserDepartmentRelationByUserReq) (*dtouser.UserDepartmentRelationPageListResp, error)
	AssignDepartments(ctx *gin.Context, req *dtouser.AssignDepartmentsReq) error
}

type userSvc struct {
}

var _ UserSvc = (*userSvc)(nil)

func NewUserSvc() UserSvc {
	return &userSvc{}
}

func (svc *userSvc) Create(ctx *gin.Context, req *dtouser.UserCreateReq) (*dtouser.UserCreateResp, error) {
	userDao := dao.NewUserDao()

	if req.Username != "" {
		existingUser, _ := userDao.GetByCond(ctx, &dao.UserCond{
			TenantID: req.TenantID,
			Username: req.Username,
		})
		if existingUser != nil && existingUser.ID != 0 {
			return nil, code.GetError(code.UsernameAlreadyExistsError)
		}
	}

	if req.PrimaryEmail != "" {
		existingUser, _ := userDao.GetByCond(ctx, &dao.UserCond{
			TenantID:    req.TenantID,
			PrimaryEmail: req.PrimaryEmail,
		})
		if existingUser != nil && existingUser.ID != 0 {
			return nil, code.GetError(code.EmailAlreadyExistsError)
		}
	}

	if req.PrimaryPhone != "" {
		existingUser, _ := userDao.GetByCond(ctx, &dao.UserCond{
			TenantID:    req.TenantID,
			PrimaryPhone: req.PrimaryPhone,
		})
		if existingUser != nil && existingUser.ID != 0 {
			return nil, code.GetError(code.PhoneAlreadyExistsError)
		}
	}

	profileJson, err := json.Marshal(req.Profile)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Create] json.Marshal profile fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}
	identitiesJson, err := json.Marshal(req.Identities)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Create] json.Marshal identities fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}
	customDataJson, err := json.Marshal(req.CustomData)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Create] json.Marshal customData fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}

	insertEntity := &model.UserEntity{
		TenantID:          req.TenantID,
		Username:          req.Username,
		PrimaryEmail:      req.PrimaryEmail,
		PrimaryPhone:      req.PrimaryPhone,
		PasswordEncrypted: req.PasswordEncrypted,
		PasswordMethod:    req.PasswordMethod,
		Name:              req.Name,
		Avatar:            req.Avatar,
		Profile:           profileJson,
		ApplicationID:     req.ApplicationID,
		Identities:        identitiesJson,
		CustomData:        customDataJson,
		IsSuspended:       req.IsSuspended,
		CreatedBy:         gincontext.GetUserID(ctx),
	}

	if err := userDao.Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcuser.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}
	return &dtouser.UserCreateResp{
		UserID: insertEntity.ID,
	}, nil
}

func (svc *userSvc) Delete(ctx *gin.Context, req *dtouser.UserDeleteReq) error {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDeleteError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewUserDao().Delete(ctx, req.UserID, userID); err != nil {
		glog.Errorf(ctx, "[svcuser.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDeleteError)
	}
	return nil
}

func (svc *userSvc) Update(ctx *gin.Context, req *dtouser.UserUpdateReq) error {
	userDao := dao.NewUserDao()
	userEntity, err := userDao.GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	if req.Username != "" && req.Username != userEntity.Username {
		existingUser, _ := userDao.GetByCond(ctx, &dao.UserCond{
			TenantID: req.TenantID,
			Username: req.Username,
		})
		if existingUser != nil && existingUser.ID != 0 && existingUser.ID != req.UserID {
			return code.GetError(code.UsernameAlreadyExistsError)
		}
	}

	if req.PrimaryEmail != "" && req.PrimaryEmail != userEntity.PrimaryEmail {
		existingUser, _ := userDao.GetByCond(ctx, &dao.UserCond{
			TenantID:    req.TenantID,
			PrimaryEmail: req.PrimaryEmail,
		})
		if existingUser != nil && existingUser.ID != 0 && existingUser.ID != req.UserID {
			return code.GetError(code.EmailAlreadyExistsError)
		}
	}

	if req.PrimaryPhone != "" && req.PrimaryPhone != userEntity.PrimaryPhone {
		existingUser, _ := userDao.GetByCond(ctx, &dao.UserCond{
			TenantID:    req.TenantID,
			PrimaryPhone: req.PrimaryPhone,
		})
		if existingUser != nil && existingUser.ID != 0 && existingUser.ID != req.UserID {
			return code.GetError(code.PhoneAlreadyExistsError)
		}
	}

	profileJson, err := json.Marshal(req.Profile)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Update] json.Marshal profile fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	identitiesJson, err := json.Marshal(req.Identities)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Update] json.Marshal identities fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	customDataJson, err := json.Marshal(req.CustomData)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Update] json.Marshal customData fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":          req.TenantID,
		"username":           req.Username,
		"primary_email":      req.PrimaryEmail,
		"primary_phone":      req.PrimaryPhone,
		"password_encrypted": req.PasswordEncrypted,
		"password_method":    req.PasswordMethod,
		"name":               req.Name,
		"avatar":             req.Avatar,
		"profile":            profileJson,
		"application_id":     req.ApplicationID,
		"identities":         identitiesJson,
		"custom_data":        customDataJson,
		"is_suspended":       req.IsSuspended,
		"updated_by":         userID,
	}
	if err := dao.NewUserDao().UpdateMap(ctx, req.UserID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcuser.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	return nil
}

func (svc *userSvc) Detail(ctx *gin.Context, req *dtouser.UserDetailReq) (*dtouser.UserDetailResp, error) {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	var profile any
	if err := json.Unmarshal(userEntity.Profile, &profile); err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] json.Unmarshal profile fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	var identities any
	if err := json.Unmarshal(userEntity.Identities, &identities); err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] json.Unmarshal identities fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	var customData any
	if err := json.Unmarshal(userEntity.CustomData, &customData); err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] json.Unmarshal customData fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}

	resp := &dtouser.UserDetailResp{
		UserID: userEntity.ID,
		UserBaseInfo: objuser.UserBaseInfo{
			TenantID:          userEntity.TenantID,
			Username:          userEntity.Username,
			PrimaryEmail:      userEntity.PrimaryEmail,
			PrimaryPhone:      userEntity.PrimaryPhone,
			PasswordEncrypted: userEntity.PasswordEncrypted,
			PasswordMethod:    userEntity.PasswordMethod,
			Name:              userEntity.Name,
			Avatar:            userEntity.Avatar,
			Profile:           profile,
			ApplicationID:     userEntity.ApplicationID,
			Identities:        identities,
			CustomData:        customData,
			IsSuspended:       userEntity.IsSuspended,
		},
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: userEntity.CreatedAt.Unix(),
			UpdatedAt: userEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *userSvc) PageList(ctx *gin.Context, req *dtouser.UserPageListReq) (*dtouser.UserPageListResp, error) {
	cond := &dao.UserCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:     req.TenantID,
		Username:     req.Username,
		PrimaryEmail: req.PrimaryEmail,
		PrimaryPhone: req.PrimaryPhone,
		Name:         req.Name,
		IsSuspended:  req.IsSuspended,
	}
	userEntityList, total, err := dao.NewUserDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetPageListError)
	}

	list := make([]dtouser.UserPageListItem, 0, len(userEntityList))
	for _, v := range userEntityList {
		var profile any
		if err := json.Unmarshal(v.Profile, &profile); err != nil {
			glog.Errorf(ctx, "[svcuser.PageList] json.Unmarshal profile fail, err:%v", err)
			continue
		}
		var identities any
		if err := json.Unmarshal(v.Identities, &identities); err != nil {
			glog.Errorf(ctx, "[svcuser.PageList] json.Unmarshal identities fail, err:%v", err)
			continue
		}
		var customData any
		if err := json.Unmarshal(v.CustomData, &customData); err != nil {
			glog.Errorf(ctx, "[svcuser.PageList] json.Unmarshal customData fail, err:%v", err)
			continue
		}
		list = append(list, dtouser.UserPageListItem{
			UserID: v.ID,
			UserBaseInfo: objuser.UserBaseInfo{
				TenantID:          v.TenantID,
				Username:          v.Username,
				PrimaryEmail:      v.PrimaryEmail,
				PrimaryPhone:      v.PrimaryPhone,
				PasswordEncrypted: v.PasswordEncrypted,
				PasswordMethod:    v.PasswordMethod,
				Name:              v.Name,
				Avatar:            v.Avatar,
				Profile:           profile,
				ApplicationID:     v.ApplicationID,
				Identities:        identities,
				CustomData:        customData,
				IsSuspended:       v.IsSuspended,
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
	if userEntity == nil || userEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"password_encrypted": req.PasswordEncrypted,
		"password_method":    req.PasswordMethod,
		"updated_by":         userID,
	}
	if err := dao.NewUserDao().UpdateMap(ctx, req.UserID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcuser.UpdatePassword] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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
	if userEntity == nil || userEntity.ID == 0 {
		return code.GetError(code.UserNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
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

func (svc *userSvc) AssignDepartments(ctx *gin.Context, req *dtouser.AssignDepartmentsReq) error {
	userID := gincontext.GetUserID(ctx)

	for _, deptID := range req.DepartmentIDs {
		insertEntity := &model.UserDepartmentRelationEntity{
			TenantID:     0,
			UserID:       req.UserID,
			DepartmentID: deptID,
			IsPrimary:    0,
			CreatedBy:    userID,
		}
		if err := dao.NewUserDepartmentRelationDao().Insert(ctx, insertEntity); err != nil {
			glog.Errorf(ctx, "[svcuser.AssignDepartments] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.UserDepartmentRelationCreateError)
		}
	}
	return nil
}

func (svc *userSvc) DetailUserLoginLog(ctx *gin.Context, req *dtouser.UserLoginLogDetailReq) (*dtouser.UserLoginLogDetailResp, error) {
	userLoginLogEntity, err := dao.NewUserLoginLogDao().GetByID(ctx, req.UserLoginLogID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.DetailUserLoginLog] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserLoginLogGetDetailError)
	}
	if userLoginLogEntity == nil || userLoginLogEntity.ID == 0 {
		return nil, code.GetError(code.UserLoginLogNotExistError)
	}

	resp := &dtouser.UserLoginLogDetailResp{
		UserLoginLogID: userLoginLogEntity.ID,
		TenantID:       userLoginLogEntity.TenantID,
		UserID:         userLoginLogEntity.UserID,
		LoginIP:        userLoginLogEntity.LoginIP,
		UserAgent:      userLoginLogEntity.UserAgent,
		LoginTime:      userLoginLogEntity.LoginTime.Unix(),
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: userLoginLogEntity.CreatedAt.Unix(),
			UpdatedAt: userLoginLogEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *userSvc) PageListUserLoginLog(ctx *gin.Context, req *dtouser.UserLoginLogPageListReq) (*dtouser.UserLoginLogPageListResp, error) {
	cond := &dao.UserLoginLogCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: req.TenantID,
		UserID:   req.UserID,
		LoginIP:  req.LoginIP,
	}
	userLoginLogEntityList, total, err := dao.NewUserLoginLogDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.PageListUserLoginLog] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
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

func (svc *userSvc) GetUserLoginLogByUser(ctx *gin.Context, req *dtouser.UserLoginLogByUserReq) (*dtouser.UserLoginLogPageListResp, error) {
	cond := &dao.UserLoginLogCond{
		UserID: req.UserID,
	}
	userLoginLogEntityList, total, err := dao.NewUserLoginLogDao().GetPageListByCond(ctx, cond)
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

func (svc *userSvc) CreateUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationCreateReq) (*dtouser.UserDepartmentRelationCreateResp, error) {
	insertEntity := &model.UserDepartmentRelationEntity{
		TenantID:     req.TenantID,
		UserID:       req.UserID,
		DepartmentID: req.DepartmentID,
		IsPrimary:    req.IsPrimary,
		CreatedBy:    gincontext.GetUserID(ctx),
	}

	if err := dao.NewUserDepartmentRelationDao().Insert(ctx, insertEntity); err != nil {
		glog.Errorf(ctx, "[svcuser.CreateUserDepartmentRelation] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserDepartmentRelationCreateError)
	}
	return &dtouser.UserDepartmentRelationCreateResp{
		UserDepartmentRelationID: insertEntity.ID,
	}, nil
}

func (svc *userSvc) DeleteUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationDeleteReq) error {
	userDepartmentRelationEntity, err := dao.NewUserDepartmentRelationDao().GetByID(ctx, req.UserDepartmentRelationID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.DeleteUserDepartmentRelation] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDepartmentRelationDeleteError)
	}
	if userDepartmentRelationEntity == nil || userDepartmentRelationEntity.ID == 0 {
		return code.GetError(code.UserDepartmentRelationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	if err := dao.NewUserDepartmentRelationDao().Delete(ctx, req.UserDepartmentRelationID, userID); err != nil {
		glog.Errorf(ctx, "[svcuser.DeleteUserDepartmentRelation] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDepartmentRelationDeleteError)
	}
	return nil
}

func (svc *userSvc) UpdateUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationUpdateReq) error {
	userDepartmentRelationEntity, err := dao.NewUserDepartmentRelationDao().GetByID(ctx, req.UserDepartmentRelationID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateUserDepartmentRelation] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDepartmentRelationUpdateError)
	}
	if userDepartmentRelationEntity == nil || userDepartmentRelationEntity.ID == 0 {
		return code.GetError(code.UserDepartmentRelationNotExistError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":     req.TenantID,
		"user_id":       req.UserID,
		"department_id": req.DepartmentID,
		"is_primary":    req.IsPrimary,
		"updated_by":    userID,
	}
	if err := dao.NewUserDepartmentRelationDao().UpdateMap(ctx, req.UserDepartmentRelationID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateUserDepartmentRelation] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDepartmentRelationUpdateError)
	}
	return nil
}

func (svc *userSvc) DetailUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationDetailReq) (*dtouser.UserDepartmentRelationDetailResp, error) {
	userDepartmentRelationEntity, err := dao.NewUserDepartmentRelationDao().GetByID(ctx, req.UserDepartmentRelationID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.DetailUserDepartmentRelation] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserDepartmentRelationGetDetailError)
	}
	if userDepartmentRelationEntity == nil || userDepartmentRelationEntity.ID == 0 {
		return nil, code.GetError(code.UserDepartmentRelationNotExistError)
	}

	resp := &dtouser.UserDepartmentRelationDetailResp{
		UserDepartmentRelationID: userDepartmentRelationEntity.ID,
		TenantID:                 userDepartmentRelationEntity.TenantID,
		UserID:                   userDepartmentRelationEntity.UserID,
		DepartmentID:             userDepartmentRelationEntity.DepartmentID,
		IsPrimary:                userDepartmentRelationEntity.IsPrimary,
		OperatorBaseInfo: gobject.OperatorBaseInfo{
			CreatedAt: userDepartmentRelationEntity.CreatedAt.Unix(),
			UpdatedAt: userDepartmentRelationEntity.UpdatedAt.Unix(),
		},
	}
	return resp, nil
}

func (svc *userSvc) PageListUserDepartmentRelation(ctx *gin.Context, req *dtouser.UserDepartmentRelationPageListReq) (*dtouser.UserDepartmentRelationPageListResp, error) {
	cond := &dao.UserDepartmentRelationCond{
		BaseCond: &genericdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:     req.TenantID,
		UserID:       req.UserID,
		DepartmentID: req.DepartmentID,
		IsPrimary:    req.IsPrimary,
	}
	userDepartmentRelationEntityList, total, err := dao.NewUserDepartmentRelationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.PageListUserDepartmentRelation] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserDepartmentRelationGetPageListError)
	}

	list := make([]dtouser.UserDepartmentRelationPageListItem, 0, len(userDepartmentRelationEntityList))
	for _, v := range userDepartmentRelationEntityList {
		list = append(list, dtouser.UserDepartmentRelationPageListItem{
			UserDepartmentRelationID: v.ID,
			TenantID:                 v.TenantID,
			UserID:                   v.UserID,
			DepartmentID:             v.DepartmentID,
			IsPrimary:                v.IsPrimary,
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserDepartmentRelationPageListResp{
		List:  list,
		Total: total,
	}, nil
}

func (svc *userSvc) GetUserDepartmentRelationByUser(ctx *gin.Context, req *dtouser.UserDepartmentRelationByUserReq) (*dtouser.UserDepartmentRelationPageListResp, error) {
	cond := &dao.UserDepartmentRelationCond{
		UserID: req.UserID,
	}
	userDepartmentRelationEntityList, total, err := dao.NewUserDepartmentRelationDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.GetUserDepartmentRelationByUser] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserDepartmentRelationGetPageListError)
	}

	list := make([]dtouser.UserDepartmentRelationPageListItem, 0, len(userDepartmentRelationEntityList))
	for _, v := range userDepartmentRelationEntityList {
		list = append(list, dtouser.UserDepartmentRelationPageListItem{
			UserDepartmentRelationID: v.ID,
			TenantID:                 v.TenantID,
			UserID:                   v.UserID,
			DepartmentID:             v.DepartmentID,
			IsPrimary:                v.IsPrimary,
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserDepartmentRelationPageListResp{
		List:  list,
		Total: total,
	}, nil
}
