package svcuser

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
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
	"gorm.io/gorm"
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
	GetUserDepartmentByUser(ctx *gin.Context, req *dtouser.UserDepartmentByUserReq) (*dtouser.UserDepartmentPageListResp, error)
	AssignDepartments(ctx *gin.Context, req *dtouser.AssignDepartmentsReq) error
}

type userSvc struct {
}

type userLoginLogQueryRepository interface {
	GetPageListByCond(ctx context.Context, cond *dao.UserLoginLogCond) (model.UserLoginLogEntityList, int64, error)
}

type userQueryRepository interface {
	GetPageListByCond(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, int64, error)
}

type userObjectScopeRepository interface {
	GetByID(ctx context.Context, id uint) (*model.UserEntity, error)
	GetPageListByCond(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, int64, error)
}

type userLoginLogDetailRepository interface {
	GetByID(ctx context.Context, id uint) (*model.UserLoginLogEntity, error)
}

type userDepartmentQueryRepository interface {
	GetPageListByCond(ctx context.Context, cond *dao.UserDepartmentCond) (model.UserDepartmentEntityList, int64, error)
}

var newUserQueryRepo = func() userQueryRepository {
	return &userQueryRepoAdapter{dao: dao.NewUserDao()}
}

var newUserObjectScopeRepo = func() userObjectScopeRepository {
	return &userObjectScopeRepoAdapter{dao: dao.NewUserDao()}
}

var newUserLoginLogQueryRepo = func() userLoginLogQueryRepository {
	return &userLoginLogQueryRepoAdapter{dao: dao.NewUserLoginLogDao()}
}

var newUserLoginLogDetailRepo = func() userLoginLogDetailRepository {
	return &userLoginLogDetailRepoAdapter{dao: dao.NewUserLoginLogDao()}
}

var newUserDepartmentQueryRepo = func() userDepartmentQueryRepository {
	return &userDepartmentQueryRepoAdapter{dao: dao.NewUserDepartmentDao()}
}

type userLoginLogQueryRepoAdapter struct {
	dao *dao.UserLoginLogDao
}

type userQueryRepoAdapter struct {
	dao *dao.UserDao
}

type userObjectScopeRepoAdapter struct {
	dao *dao.UserDao
}

type userLoginLogDetailRepoAdapter struct {
	dao *dao.UserLoginLogDao
}

func (r *userQueryRepoAdapter) GetPageListByCond(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, int64, error) {
	return r.dao.GetPageListByCond(ctx, cond)
}

func (r *userObjectScopeRepoAdapter) GetByID(ctx context.Context, id uint) (*model.UserEntity, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *userObjectScopeRepoAdapter) GetPageListByCond(ctx context.Context, cond *dao.UserCond) (model.UserEntityList, int64, error) {
	return r.dao.GetPageListByCond(ctx, cond)
}

func (r *userLoginLogQueryRepoAdapter) GetPageListByCond(ctx context.Context, cond *dao.UserLoginLogCond) (model.UserLoginLogEntityList, int64, error) {
	return r.dao.GetPageListByCond(ctx, cond)
}

func (r *userLoginLogDetailRepoAdapter) GetByID(ctx context.Context, id uint) (*model.UserLoginLogEntity, error) {
	return r.dao.GetByID(ctx, id)
}

type userDepartmentQueryRepoAdapter struct {
	dao *dao.UserDepartmentDao
}

func (r *userDepartmentQueryRepoAdapter) GetPageListByCond(ctx context.Context, cond *dao.UserDepartmentCond) (model.UserDepartmentEntityList, int64, error) {
	return r.dao.GetPageListByCond(ctx, cond)
}

var iamDBFromContext = func(ctx context.Context) *gorm.DB {
	return dbclient.IamDB(ctx)
}

var _ UserSvc = (*userSvc)(nil)

func NewUserSvc() UserSvc {
	return &userSvc{}
}

// loadPersonMap 批量加载自然人信息（username/email/phone 在 person 表），
// 用于用户列表/详情关联展示。查询失败仅告警并返回空 map，不阻断主流程。
func (svc *userSvc) loadPersonMap(ctx context.Context, personIDs []uint) map[uint]*model.PersonEntity {
	result := make(map[uint]*model.PersonEntity)
	if len(personIDs) == 0 {
		return result
	}
	personDao := dao.NewPersonDao()
	for _, id := range personIDs {
		if id == 0 {
			continue
		}
		person, err := personDao.GetByID(ctx, id)
		if err != nil {
			glog.Warnf(ctx, "[svcuser.loadPersonMap] person GetByID fail, personID:%d, err:%v", id, err)
			continue
		}
		if person != nil && person.ID != 0 {
			result[id] = person
		}
	}
	return result
}

func (svc *userSvc) Create(ctx *gin.Context, req *dtouser.UserCreateReq) (*dtouser.UserCreateResp, error) {
	userDao := dao.NewUserDao()
	personDao := dao.NewPersonDao()
	tenantID := gincontext.GetTenantID(ctx)
	if req.TenantID == 0 {
		req.TenantID = tenantID
	}

	// username/primaryEmail/primaryPhone 为全局自然人标识，唯一性校验需查 person 表
	if req.Username != "" {
		existingPerson, _ := personDao.GetByCond(ctx, &dao.PersonCond{Username: req.Username})
		if existingPerson != nil && existingPerson.ID != 0 {
			return nil, code.GetError(code.UsernameAlreadyExistsError)
		}
	}

	if req.PrimaryEmail != "" {
		existingPerson, _ := personDao.GetByCond(ctx, &dao.PersonCond{PrimaryEmail: req.PrimaryEmail})
		if existingPerson != nil && existingPerson.ID != 0 {
			return nil, code.GetError(code.EmailAlreadyExistsError)
		}
	}

	if req.PrimaryPhone != "" {
		existingPerson, _ := personDao.GetByCond(ctx, &dao.PersonCond{PrimaryPhone: req.PrimaryPhone})
		if existingPerson != nil && existingPerson.ID != 0 {
			return nil, code.GetError(code.PhoneAlreadyExistsError)
		}
	}

	profileJson, err := json.Marshal(req.Profile)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Create] json.Marshal profile fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}
	customDataJson, err := json.Marshal(req.CustomData)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Create] json.Marshal customData fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}

	personID := req.PersonID
	operatorID := gincontext.GetUserID(ctx)
	now := time.Now()
	var createdUserID uint
	txErr := iamDBFromContext(ctx).Transaction(func(tx *gorm.DB) error {
		if personID == 0 {
			// 未指定已有自然人：当提供了可登录标识（username/email/phone）时自动创建 person，
			// 使该用户可登录；否则创建仅租户内可见、无登录凭证的用户记录。
			if req.Username != "" || req.PrimaryEmail != "" || req.PrimaryPhone != "" || req.Password != "" {
				if req.Username == "" && req.PrimaryEmail == "" && req.PrimaryPhone == "" {
					return code.GetError(code.AuthIdentifierRequiredError)
				}
				passwordHash := ""
				if req.Password != "" {
					hash, hashErr := gcrypto.GeneratePasswordHash(req.Password)
					if hashErr != nil {
						glog.Errorf(ctx, "[svcuser.Create] GeneratePasswordHash fail, err:%v", hashErr)
						return code.GetError(code.PasswordHashError)
					}
					passwordHash = hash
				}
				personEntity := &model.PersonEntity{
					Username:          model.StrPtr(req.Username),
					PrimaryEmail:      model.StrPtr(req.PrimaryEmail),
					PrimaryPhone:      model.StrPtr(req.PrimaryPhone),
					PasswordEncrypted: passwordHash,
					PasswordMethod:    "bcrypt",
					Name:              req.Name,
					Avatar:            req.Avatar,
					Profile:           profileJson,
					CustomData:        customDataJson,
					CreatedBy:         operatorID,
				}
				if insertErr := dao.NewPersonDao().WithTx(tx).Insert(ctx, personEntity); insertErr != nil {
					glog.Errorf(ctx, "[svcuser.Create] person Insert fail, err:%v", insertErr)
					return fmt.Errorf("person insert: %w", insertErr)
				}
				personID = personEntity.ID
			}
		}

		insertEntity := &model.UserEntity{
			TenantID:    req.TenantID,
			PersonID:    personID,
			Name:        req.Name,
			Avatar:      req.Avatar,
			Profile:     profileJson,
			CustomData:  customDataJson,
			IsSuspended: req.IsSuspended,
			IsOwner:     0,
			JoinedAt:    &now,
			CreatedBy:   operatorID,
		}

		if insertErr := userDao.WithTx(tx).Insert(ctx, insertEntity); insertErr != nil {
			glog.Errorf(ctx, "[svcuser.Create] dao Insert fail, err:%v, req:%s", insertErr, gutil.ToJsonString(req))
			return fmt.Errorf("user insert: %w", insertErr)
		}
		createdUserID = insertEntity.ID
		return nil
	})
	if txErr != nil {
		if txErr == code.GetError(code.AuthIdentifierRequiredError) || txErr == code.GetError(code.PasswordHashError) {
			return nil, txErr
		}
		glog.Errorf(ctx, "[svcuser.Create] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}

	return &dtouser.UserCreateResp{
		UserID: createdUserID,
	}, nil
}

func (svc *userSvc) Delete(ctx *gin.Context, req *dtouser.UserDeleteReq) error {
	userEntity, err := newUserObjectScopeRepo().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserDeleteError)
	}
	if userEntity == nil || userEntity.ID == 0 || userEntity.TenantID != gincontext.GetTenantID(ctx) {
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
	personDao := dao.NewPersonDao()
	userEntity, err := newUserObjectScopeRepo().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == 0 || userEntity.TenantID != gincontext.GetTenantID(ctx) {
		return code.GetError(code.UserNotExistError)
	}

	if req.Username != "" {
		existingPerson, _ := personDao.GetByCond(ctx, &dao.PersonCond{Username: req.Username})
		if existingPerson != nil && existingPerson.ID != 0 && existingPerson.ID != userEntity.PersonID {
			return code.GetError(code.UsernameAlreadyExistsError)
		}
	}

	if req.PrimaryEmail != "" {
		existingPerson, _ := personDao.GetByCond(ctx, &dao.PersonCond{PrimaryEmail: req.PrimaryEmail})
		if existingPerson != nil && existingPerson.ID != 0 && existingPerson.ID != userEntity.PersonID {
			return code.GetError(code.EmailAlreadyExistsError)
		}
	}

	if req.PrimaryPhone != "" {
		existingPerson, _ := personDao.GetByCond(ctx, &dao.PersonCond{PrimaryPhone: req.PrimaryPhone})
		if existingPerson != nil && existingPerson.ID != 0 && existingPerson.ID != userEntity.PersonID {
			return code.GetError(code.PhoneAlreadyExistsError)
		}
	}

	profileJson, err := json.Marshal(req.Profile)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Update] json.Marshal profile fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	customDataJson, err := json.Marshal(req.CustomData)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Update] json.Marshal customData fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}

	userID := gincontext.GetUserID(ctx)
	updateMap := map[string]any{
		"tenant_id":    req.TenantID,
		"name":         req.Name,
		"avatar":       req.Avatar,
		"profile":      profileJson,
		"custom_data":  customDataJson,
		"is_suspended": req.IsSuspended,
		"updated_by":   userID,
	}
	if err := dao.NewUserDao().UpdateMap(ctx, req.UserID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcuser.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	return nil
}

func (svc *userSvc) Detail(ctx *gin.Context, req *dtouser.UserDetailReq) (*dtouser.UserDetailResp, error) {
	userEntity, err := newUserObjectScopeRepo().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 || userEntity.TenantID != gincontext.GetTenantID(ctx) {
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

	personMap := svc.loadPersonMap(ctx, []uint{userEntity.PersonID})
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
	userRepo := newUserQueryRepo()
	cond := &dao.UserCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:    gincontext.GetTenantID(ctx),
		Name:        req.Name,
		IsSuspended: req.IsSuspended,
	}
	userEntityList, total, err := userRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetPageListError)
	}

	list := make([]dtouser.UserPageListItem, 0, len(userEntityList))
	personIDs := make([]uint, 0, len(userEntityList))
	for _, v := range userEntityList {
		if v.PersonID != 0 {
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
	userEntity, err := newUserObjectScopeRepo().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UpdatePassword] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == 0 || userEntity.TenantID != gincontext.GetTenantID(ctx) {
		return code.GetError(code.UserNotExistError)
	}
	if userEntity.PersonID == 0 {
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

	userID := gincontext.GetUserID(ctx)
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
	userEntity, err := newUserObjectScopeRepo().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateStatus] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == 0 || userEntity.TenantID != gincontext.GetTenantID(ctx) {
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
	tenantID := gincontext.GetTenantID(ctx)

	txErr := iamDBFromContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, deptID := range req.DepartmentIDs {
			var existing model.UserDepartmentEntity
			err := tx.WithContext(ctx).Model(&model.UserDepartmentEntity{}).
				Where("tenant_id = ? AND user_id = ? AND department_id = ?", tenantID, req.UserID, deptID).
				First(&existing).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				return err
			}
			if err == nil && existing.ID != 0 {
				continue
			}

			insertEntity := &model.UserDepartmentEntity{
				TenantID:     tenantID,
				UserID:       req.UserID,
				DepartmentID: deptID,
				IsPrimary:    0,
				CreatedBy:    userID,
			}
			if err := tx.WithContext(ctx).Create(insertEntity).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcuser.AssignDepartments] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.UserDepartmentCreateError)
	}
	return nil
}

func (svc *userSvc) DetailUserLoginLog(ctx *gin.Context, req *dtouser.UserLoginLogDetailReq) (*dtouser.UserLoginLogDetailResp, error) {
	userLoginLogEntity, err := newUserLoginLogDetailRepo().GetByID(ctx, req.UserLoginLogID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.DetailUserLoginLog] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserLoginLogGetDetailError)
	}
	if userLoginLogEntity == nil || userLoginLogEntity.ID == 0 || userLoginLogEntity.TenantID != gincontext.GetTenantID(ctx) {
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
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID: gincontext.GetTenantID(ctx),
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
	userLoginLogRepo := newUserLoginLogQueryRepo()
	cond := &dao.UserLoginLogCond{
		TenantID: gincontext.GetTenantID(ctx),
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

func (svc *userSvc) GetUserDepartmentByUser(ctx *gin.Context, req *dtouser.UserDepartmentByUserReq) (*dtouser.UserDepartmentPageListResp, error) {
	userDepartmentRepo := newUserDepartmentQueryRepo()
	cond := &dao.UserDepartmentCond{
		TenantID: gincontext.GetTenantID(ctx),
		UserID:   req.UserID,
	}
	userDepartmentEntityList, total, err := userDepartmentRepo.GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.GetUserDepartmentByUser] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserDepartmentGetPageListError)
	}

	list := make([]dtouser.UserDepartmentPageListItem, 0, len(userDepartmentEntityList))
	for _, v := range userDepartmentEntityList {
		list = append(list, dtouser.UserDepartmentPageListItem{
			UserDepartmentID: v.ID,
			TenantID:         v.TenantID,
			UserID:           v.UserID,
			DepartmentID:     v.DepartmentID,
			IsPrimary:        v.IsPrimary,
			OperatorBaseInfo: gobject.OperatorBaseInfo{
				UpdatedAt: v.UpdatedAt.Unix(),
			},
		})
	}
	return &dtouser.UserDepartmentPageListResp{
		List:  list,
		Total: total,
	}, nil
}
