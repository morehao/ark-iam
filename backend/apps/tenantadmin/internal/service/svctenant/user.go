package svctenant

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/audit"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/core/person"
	"github.com/morehao/ark-iam/pkg/core/tenant"
	"github.com/morehao/ark-iam/pkg/core/user"
	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

// userLoginLogPageSize 登录日志按子资源整段展示，取足够大的单页容量。
const userLoginLogPageSize = 100

type UserSvc interface {
	PageList(ctx *gin.Context, req *dtotenant.UserPageListReq) (*dtotenant.UserPageListResp, error)
	Create(ctx *gin.Context, req *dtotenant.UserCreateReq) (*dtotenant.UserCreateResp, error)
	Detail(ctx *gin.Context, req *dtotenant.UserDetailReq) (*dtotenant.UserDetailResp, error)
	Update(ctx *gin.Context, req *dtotenant.UserUpdateReq) error
	ResetPassword(ctx *gin.Context, req *dtotenant.UserResetPasswordReq) (*dtotenant.UserResetPasswordResp, error)
	ListRoles(ctx *gin.Context, req *dtotenant.UserRolesListReq) (*dtotenant.UserRolesListResp, error)
	UpdateRoles(ctx *gin.Context, req *dtotenant.UserRolesUpdateReq) error
	ListLoginLogs(ctx *gin.Context, req *dtotenant.UserLoginLogListReq) (*dtotenant.UserLoginLogListResp, error)
}

type userSvc struct {
}

var _ UserSvc = (*userSvc)(nil)

func NewUserSvc() UserSvc {
	return &userSvc{}
}

// PageList 返回当前租户内的用户目录（含自然人基础信息），支持关键词（姓名/用户名/邮箱/手机）与状态过滤；
// 传 departmentID 时仅返回"恰在该部门"的用户（含 primary/secondary/leader 任一关系，不含子部门）。
func (svc *userSvc) PageList(ctx *gin.Context, req *dtotenant.UserPageListReq) (*dtotenant.UserPageListResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	cond := &dao.UserCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:    tenantID,
		UserType:    model.UserTypeMember,
		Keyword:     req.Keyword,
		IsSuspended: req.IsSuspended,
	}

	// 部门过滤：仅筛选恰在该部门的用户（member/leader 均可），不掺子部门。
	if req.DepartmentID != "" {
		relationList, err := dao.NewDepartmentUserDao().GetListByCond(ctx, &dao.DepartmentUserCond{
			TenantID:     tenantID,
			DepartmentID: req.DepartmentID,
		})
		if err != nil {
			glog.Errorf(ctx, "[svcuser.PageList] dao relation GetListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.UserGetPageListError)
		}
		userIDs := make([]string, 0, len(relationList))
		for _, r := range relationList {
			userIDs = append(userIDs, r.UserID)
		}
		if len(userIDs) == 0 {
			return &dtotenant.UserPageListResp{List: []dtotenant.UserPageListItem{}, Total: 0}, nil
		}
		cond.IDs = userIDs
	}

	userEntityList, total, err := dao.NewUserDao().GetPageListByCond(ctx, cond)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetPageListError)
	}

	userIDs := make([]string, 0, len(userEntityList))
	for _, v := range userEntityList {
		userIDs = append(userIDs, v.ID)
	}
	_, personMap := svc.loadUserPersonMaps(ctx, userIDs)

	// 主部门 + 角色数 聚合（批量，避免 N+1）
	primaryDepartmentNameMap := loadPrimaryDepartmentNameMap(ctx, tenantID, userIDs)
	roleCountMap := loadUserRoleCountMap(ctx, tenantID, userIDs)

	list := make([]dtotenant.UserPageListItem, 0, len(userEntityList))
	for _, v := range userEntityList {
		person := personMap[v.PersonID]
		if person == nil {
			person = &model.PersonEntity{}
		}
		list = append(list, dtotenant.UserPageListItem{
			UserID:                v.ID,
			TenantID:              v.TenantID,
			Username:              model.DerefStr(person.Username),
			PrimaryEmail:          model.DerefStr(person.PrimaryEmail),
			PrimaryPhone:          model.DerefStr(person.PrimaryPhone),
			Name:                  v.Name,
			Avatar:                v.Avatar,
			IsSuspended:           v.IsSuspended,
			PrimaryDepartmentName: primaryDepartmentNameMap[v.ID],
			RoleCount:             roleCountMap[v.ID],
			CreatedAt:             v.CreatedAt.Unix(),
			UpdatedAt:             v.UpdatedAt.Unix(),
		})
	}
	return &dtotenant.UserPageListResp{
		List:  list,
		Total: total,
	}, nil
}

// loadPrimaryDepartmentNameMap 批量查询用户行政归属部门名称（member 唯一行）。
func loadPrimaryDepartmentNameMap(ctx *gin.Context, tenantID string, userIDs []string) map[string]string {
	result := make(map[string]string, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}
	relationList, err := dao.NewDepartmentUserDao().GetListByCond(ctx, &dao.DepartmentUserCond{
		TenantID:     tenantID,
		RelationType: model.DeptUserRelationPrimary,
		UserIDs:      userIDs,
	})
	if err != nil {
		glog.Warnf(ctx, "[svcuser.loadPrimaryDepartmentNameMap] query member dept fail, err:%v", err)
		return result
	}
	userDeptMap := make(map[string]string, len(relationList))
	deptIDs := make([]string, 0, len(relationList))
	for _, r := range relationList {
		if _, ok := userDeptMap[r.UserID]; ok {
			continue
		}
		userDeptMap[r.UserID] = r.DepartmentID
		deptIDs = append(deptIDs, r.DepartmentID)
	}
	if len(deptIDs) == 0 {
		return result
	}
	var deptList []model.DepartmentEntity
	if err := dbclient.IamDB(ctx).Model(&model.DepartmentEntity{}).
		Where("tenant_id = ? AND id IN ?", tenantID, deptIDs).
		Find(&deptList).Error; err != nil {
		glog.Warnf(ctx, "[svcuser.loadPrimaryDepartmentNameMap] query dept name fail, err:%v", err)
		return result
	}
	deptNameMap := make(map[string]string, len(deptList))
	for _, o := range deptList {
		deptNameMap[o.ID] = o.Name
	}
	for userID, deptID := range userDeptMap {
		result[userID] = deptNameMap[deptID]
	}
	return result
}

// loadUserRoleCountMap 批量统计用户角色数（GROUP BY user_id）。
func loadUserRoleCountMap(ctx *gin.Context, tenantID string, userIDs []string) map[string]int64 {
	result := make(map[string]int64, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}
	type countRow struct {
		UserID string
		Cnt    int64
	}
	var rows []countRow
	if err := dbclient.IamDB(ctx).Model(&model.UserRoleEntity{}).
		Where("tenant_id = ? AND user_id IN ?", tenantID, userIDs).
		Select("user_id, count(*) as cnt").Group("user_id").Scan(&rows).Error; err != nil {
		glog.Warnf(ctx, "[svcuser.loadUserRoleCountMap] count user_role fail, err:%v", err)
		return result
	}
	for _, r := range rows {
		result[r.UserID] = r.Cnt
	}
	return result
}

// Create 创建租户用户：person find-or-create（见设计文档 §4.4）。
// 提供 personID 直接关联；否则按 email/phone 命中已有 person 则复用；未命中则同事务创建 person（姓名即自然人姓名）；
// 同时按 primaryDepartmentID 建立行政归属（primary，单值）、secondaryDepartmentIDs 参与部门、
// leaderDepartmentIDs 负责关系（leader）。
// 业务约束：用户必须从属于一个主部门，primaryDepartmentID 必传。
//
// 口径与平台侧建租户内置管理员完全一致（需求①/D7）：
//   - 密码不由调用方提供，统一由 pkg/credential 生成临时密码，仅本次响应返回一次；
//   - 仅新建自然人时置 must_change_password=true（复用既有自然人绝不改动其密码）；
//   - person/user/部门关系同事务写入，共用 pkg/core/user.Create。
func (svc *userSvc) Create(ctx *gin.Context, req *dtotenant.UserCreateReq) (*dtotenant.UserCreateResp, error) {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.UserCreateError); err != nil {
		return nil, err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	operatorID := gincontext.GetUserIDString(ctx)

	// 0. 用户必须从属于一个主部门（业务约束，防绕过 DTO 校验）
	if req.PrimaryDepartmentID == "" {
		return nil, code.GetError(code.UserDepartmentRequiredError)
	}
	// 0.1 邮箱或手机号至少填写一个（联系方式是用户识别/找回的必需信息）
	if req.PrimaryEmail == "" && req.PrimaryPhone == "" {
		return nil, code.GetError(code.UserContactRequiredError)
	}

	// 1. 显式 personID 时先校验存在性（err 为系统错误、nil 为业务边界，两者分开判断）
	if req.PersonID != "" {
		personEntity, err := dao.NewPersonDao().GetByID(ctx, req.PersonID)
		if err != nil {
			glog.Errorf(ctx, "[svcuser.Create] dao GetByID person fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.UserCreateError)
		}
		if personEntity == nil || personEntity.ID == "" {
			return nil, code.GetError(code.UserNotExistError)
		}
	}

	// 2. 校验归属部门均属于本租户
	deptList, err := dao.NewDepartmentDao().GetListByCond(ctx, &dao.DepartmentCond{TenantID: tenantID})
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Create] query dept fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}
	deptSet := make(map[string]bool, len(deptList))
	for _, o := range deptList {
		deptSet[o.ID] = true
	}
	if !deptSet[req.PrimaryDepartmentID] {
		return nil, code.GetError(code.DepartmentNotExistError)
	}
	for _, deptID := range req.SecondaryDepartmentIDs {
		if !deptSet[deptID] {
			return nil, code.GetError(code.DepartmentNotExistError)
		}
	}
	for _, deptID := range req.LeaderDepartmentIDs {
		if !deptSet[deptID] {
			return nil, code.GetError(code.DepartmentNotExistError)
		}
	}

	// 3. 初始临时密码：系统生成，仅新建自然人时生效（复用既有自然人时密码保持不变）
	tempPassword, err := credential.GenerateTemporaryPassword()
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Create] generate temporary password fail, err:%v", err)
		return nil, code.GetError(code.UserCreateError)
	}
	passwordHash, err := gcrypto.GeneratePasswordHash(tempPassword)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Create] GeneratePasswordHash fail, err:%v", err)
		return nil, code.GetError(code.PasswordHashError)
	}

	// 4. 事务：person find-or-create + user 主体 + 部门归属（共用 pkg/core/user.Create）
	var createdUserID string
	personCreated := false
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		createReq := &user.CreateReq{
			TenantID:               tenantID,
			PersonID:               req.PersonID,
			Name:                   req.Name,
			Avatar:                 req.Avatar,
			IsSuspended:            req.IsSuspended,
			CreatedBy:              operatorID,
			PrimaryDepartmentID:    req.PrimaryDepartmentID,
			SecondaryDepartmentIDs: req.SecondaryDepartmentIDs,
			LeaderDepartmentIDs:    req.LeaderDepartmentIDs,
		}
		if req.PersonID == "" {
			createReq.Person = &person.FindOrCreateReq{
				Username:           req.Username,
				PrimaryEmail:       req.PrimaryEmail,
				PrimaryPhone:       req.PrimaryPhone,
				PasswordEncrypted:  passwordHash,
				PasswordMethod:     model.PasswordMethodBcrypt,
				MustChangePassword: true,
				Name:               req.Name,
				Avatar:             req.Avatar,
				CreatedBy:          operatorID,
			}
		}
		createdUser, isNewPerson, createErr := user.Create(ctx, tx, createReq)
		if createErr != nil {
			return createErr
		}
		createdUserID = createdUser.ID
		personCreated = isNewPerson
		return nil
	})
	if txErr != nil {
		// 公共层哨兵错误 → 本应用错误码（err 与业务边界判定分离）
		switch {
		case errors.Is(txErr, user.ErrAlreadyInTenant):
			return nil, code.GetError(code.UserAlreadyInTenantError)
		case errors.Is(txErr, user.ErrPersonNotFound):
			return nil, code.GetError(code.UserNotExistError)
		case errors.Is(txErr, user.ErrDeptLeaderConflict):
			return nil, code.GetError(code.DepartmentUserLeaderConflictError)
		}
		glog.Errorf(ctx, "[svcuser.Create] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}

	resp := &dtotenant.UserCreateResp{UserID: createdUserID}
	if personCreated {
		resp.InitialPassword = tempPassword
	}
	// TODO(delivery): 临时密码目前只能在本响应中回显一次（系统尚无邮件/短信通道）；
	// 接入通道后改为下发给账号本人，本响应不再返回明文。
	// 见 docs/design/tenant-admin-provisioning-design-20260912.md T1/Q4。
	return resp, nil
}

// Detail 用户详情：基础信息（含自然人）+ 部门归属 + 已分配角色。
func (svc *userSvc) Detail(ctx *gin.Context, req *dtotenant.UserDetailReq) (*dtotenant.UserDetailResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return nil, code.GetError(code.UserNotExistError)
	}

	userMap, personMap := svc.loadUserPersonMaps(ctx, []string{userEntity.ID})
	u := userMap[userEntity.ID]
	if u == nil {
		u = userEntity
	}
	person := personMap[u.PersonID]
	if person == nil {
		person = &model.PersonEntity{}
	}

	resp := &dtotenant.UserDetailResp{
		UserPageListItem: dtotenant.UserPageListItem{
			UserID:       u.ID,
			TenantID:     u.TenantID,
			Username:     model.DerefStr(person.Username),
			PrimaryEmail: model.DerefStr(person.PrimaryEmail),
			PrimaryPhone: model.DerefStr(person.PrimaryPhone),
			Name:         u.Name,
			Avatar:       u.Avatar,
			IsSuspended:  u.IsSuspended,
			CreatedAt:    u.CreatedAt.Unix(),
		},
	}

	// 部门归属
	departments, err := loadUserDepartments(ctx, tenantID, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] load departments fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	resp.Departments = departments

	// 已分配角色
	roles, err := svc.listRoles(ctx, tenantID, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] list roles fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	resp.Roles = roles
	return resp, nil
}

// Update 局部更新用户（PATCH）：姓名/头像/状态 + 主/参与/负责部门更新。
func (svc *userSvc) Update(ctx *gin.Context, req *dtotenant.UserUpdateReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.UserUpdateError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return code.GetError(code.UserNotExistError)
	}
	operatorID := gincontext.GetUserIDString(ctx)

	// 主部门不可清空：显式传 "" 拒绝。
	if req.PrimaryDepartmentID != nil && *req.PrimaryDepartmentID == "" {
		return code.GetError(code.UserDepartmentRequiredError)
	}

	// 编辑联系方式（person 全局标识）：加载当前 person 以便做二选一与唯一性校验。
	var curPerson *model.PersonEntity
	if (req.Username != nil || req.PrimaryEmail != nil || req.PrimaryPhone != nil) && userEntity.PersonID != "" {
		curPerson, err = dao.NewPersonDao().GetByID(ctx, userEntity.PersonID)
		if err != nil {
			glog.Errorf(ctx, "[svcuser.Update] dao GetByID person fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.UserUpdateError)
		}
	}

	// 联系方式二选一 + 唯一性：基于变更后结果校验（传入值 + 保持的旧值）。
	if curPerson != nil {
		newEmail := model.DerefStr(curPerson.PrimaryEmail)
		if req.PrimaryEmail != nil {
			newEmail = *req.PrimaryEmail
		}
		newPhone := model.DerefStr(curPerson.PrimaryPhone)
		if req.PrimaryPhone != nil {
			newPhone = *req.PrimaryPhone
		}
		if newEmail == "" && newPhone == "" {
			return code.GetError(code.UserContactRequiredError)
		}
		if err := checkPersonContactUnique(ctx, curPerson.ID, req.Username, req.PrimaryEmail, req.PrimaryPhone); err != nil {
			return err
		}
	}

	// 校验传入的部门均属本租户。
	deptSet := make(map[string]bool)
	if (req.PrimaryDepartmentID != nil && *req.PrimaryDepartmentID != "") || req.SecondaryDepartmentIDs != nil || req.LeaderDepartmentIDs != nil {
		deptList, err := dao.NewDepartmentDao().GetListByCond(ctx, &dao.DepartmentCond{TenantID: tenantID})
		if err != nil {
			glog.Errorf(ctx, "[svcuser.Update] query dept fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.UserUpdateError)
		}
		for _, o := range deptList {
			deptSet[o.ID] = true
		}
		checkOwnership := func(id string) bool { return id == "" || deptSet[id] }
		if req.PrimaryDepartmentID != nil && !checkOwnership(*req.PrimaryDepartmentID) {
			return code.GetError(code.DepartmentNotExistError)
		}
		for _, id := range derefSlice(req.SecondaryDepartmentIDs) {
			if !deptSet[id] {
				return code.GetError(code.DepartmentNotExistError)
			}
		}
		for _, id := range derefSlice(req.LeaderDepartmentIDs) {
			if !deptSet[id] {
				return code.GetError(code.DepartmentNotExistError)
			}
		}
	}

	var txErr = dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		updateMap := map[string]any{"updated_by": operatorID}
		if req.Name != "" {
			updateMap["name"] = req.Name
		}
		if req.Avatar != "" {
			updateMap["avatar"] = req.Avatar
		}
		if req.IsSuspended != nil {
			updateMap["is_suspended"] = *req.IsSuspended
		}
		if len(updateMap) > 0 {
			if err := dao.NewUserDao().UpdateMap(ctx, req.UserID, updateMap); err != nil {
				return err
			}
		}
		if curPerson != nil {
			if err := updatePersonContact(ctx, tx, curPerson, req.Username, req.PrimaryEmail, req.PrimaryPhone); err != nil {
				return err
			}
		}
		if req.PrimaryDepartmentID != nil {
			// 替换主部门：删旧 primary（至多 1 行）后建新。
			if err := replaceDeptRelationList(ctx, tx, tenantID, req.UserID, []string{*req.PrimaryDepartmentID}, model.DeptUserRelationPrimary, operatorID); err != nil {
				return err
			}
		}
		if req.SecondaryDepartmentIDs != nil {
			if err := replaceDeptRelationList(ctx, tx, tenantID, req.UserID, *req.SecondaryDepartmentIDs, model.DeptUserRelationSecondary, operatorID); err != nil {
				return err
			}
		}
		if req.LeaderDepartmentIDs != nil {
			if err := replaceDeptRelationList(ctx, tx, tenantID, req.UserID, *req.LeaderDepartmentIDs, model.DeptUserRelationLeader, operatorID); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		if txErr == code.GetError(code.DepartmentUserLeaderConflictError) {
			return txErr
		}
		glog.Errorf(ctx, "[svcuser.Update] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	return nil
}

// derefSlice 解引用可空切片，nil 返回空切片。
func derefSlice(p *[]string) []string {
	if p == nil {
		return nil
	}
	return *p
}

// checkPersonContactUnique 校验变更后的用户名/邮箱/手机号不被其他 person 占用。
// 仅对传入（非 nil）的字段做查重，排除当前 person 自身。
func checkPersonContactUnique(ctx *gin.Context, selfPersonID string, username, email, phone *string) error {
	if username != nil && *username != "" {
		if p, err := dao.NewPersonDao().GetByCond(ctx, &dao.PersonCond{Username: *username}); err != nil {
			return err
		} else if p != nil && p.ID != "" && p.ID != selfPersonID {
			return code.GetError(code.UsernameAlreadyExistsError)
		}
	}
	if email != nil && *email != "" {
		if p, err := dao.NewPersonDao().GetByCond(ctx, &dao.PersonCond{PrimaryEmail: *email}); err != nil {
			return err
		} else if p != nil && p.ID != "" && p.ID != selfPersonID {
			return code.GetError(code.EmailAlreadyExistsError)
		}
	}
	if phone != nil && *phone != "" {
		if p, err := dao.NewPersonDao().GetByCond(ctx, &dao.PersonCond{PrimaryPhone: *phone}); err != nil {
			return err
		} else if p != nil && p.ID != "" && p.ID != selfPersonID {
			return code.GetError(code.PhoneAlreadyExistsError)
		}
	}
	return nil
}

// updatePersonContact 更新 person 的用户名/邮箱/手机号（空串视为清空）。须在事务内传入 tx。
func updatePersonContact(ctx *gin.Context, tx *gorm.DB, person *model.PersonEntity, username, email, phone *string) error {
	personDao := dao.NewPersonDao().WithTx(tx)
	updateMap := map[string]any{"updated_by": gincontext.GetUserIDString(ctx)}
	if username != nil {
		updateMap["username"] = strPtrOrNil(username)
	}
	if email != nil {
		updateMap["primary_email"] = strPtrOrNil(email)
	}
	if phone != nil {
		updateMap["primary_phone"] = strPtrOrNil(phone)
	}
	if len(updateMap) == 0 {
		return nil
	}
	return personDao.UpdateMap(ctx, person.ID, updateMap)
}

// strPtrOrNil 将指针字符串转 person 存储值：空串转 nil（清空），非空保留指针。
func strPtrOrNil(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

// ResetPassword 重置成员密码（D7：与平台侧重置内置管理员的口径完全一致）。
// 不接收新密码：由服务端生成临时密码并在响应中返回一次；置 must_change_password=true
// 使新口令首次登录必须改密，并撤销该自然人既有 SSO 会话与 refresh token（改密即全局登出）。
// 无自然人关联的用户（服务账号等）不可登录，也没有口令语义，直接拒绝。
func (svc *userSvc) ResetPassword(ctx *gin.Context, req *dtotenant.UserResetPasswordReq) (*dtotenant.UserResetPasswordResp, error) {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.UserResetPasswordError); err != nil {
		return nil, err
	}
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.ResetPassword] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != gincontext.GetTenantIDString(ctx) {
		return nil, code.GetError(code.UserNotExistError)
	}
	// 服务账号（user_type=machine）不使用口令，归属其自身的密钥管理，不在此处重置
	if userEntity.UserType != model.UserTypeMember {
		return nil, code.GetError(code.UserNotExistError)
	}
	if userEntity.PersonID == "" {
		return nil, code.GetError(code.UserNotExistError)
	}

	tempPassword, err := credential.GenerateTemporaryPassword()
	if err != nil {
		glog.Errorf(ctx, "[svcuser.ResetPassword] generate temporary password fail, err:%v", err)
		return nil, code.GetError(code.UserResetPasswordError)
	}
	hash, err := gcrypto.GeneratePasswordHash(tempPassword)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.ResetPassword] GeneratePasswordHash fail, err:%v", err)
		return nil, code.GetError(code.PasswordHashError)
	}
	if err := dao.NewPersonDao().UpdateMap(ctx, userEntity.PersonID, map[string]any{
		"password_encrypted":   hash,
		"password_method":      model.PasswordMethodBcrypt,
		"must_change_password": true,
		"updated_by":           gincontext.GetUserIDString(ctx),
	}); err != nil {
		glog.Errorf(ctx, "[svcuser.ResetPassword] person UpdateMap fail, err:%v", err)
		return nil, code.GetError(code.UserResetPasswordError)
	}

	// 改密即全局登出：旧 SSO 会话与 refresh token 立即失效
	tenant.RevokePersonSessions(ctx, userEntity.PersonID)

	audit.WriteAudit(ctx, audit.AuditEntry{
		Action:     audit.ActionTenantAdminPasswordReset,
		TenantID:   gincontext.GetTenantIDString(ctx),
		Result:     model.AuditResultSuccess,
		TargetType: model.AuditTargetTypeUser,
		TargetID:   userEntity.ID,
	})
	// TODO(delivery): 临时密码目前只能在本响应中回显一次（系统尚无邮件/短信通道）；
	// 接入通道后改为下发给账号本人，本响应不再返回明文。
	// 见 docs/design/tenant-admin-provisioning-design-20260912.md T1/Q4。
	return &dtotenant.UserResetPasswordResp{
		UserID:          userEntity.ID,
		InitialPassword: tempPassword,
	}, nil
}

// ListRoles 用户已分配角色列表。
func (svc *userSvc) ListRoles(ctx *gin.Context, req *dtotenant.UserRolesListReq) (*dtotenant.UserRolesListResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.ListRoles] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return nil, code.GetError(code.UserNotExistError)
	}
	// 服务账号角色走 /machine-users/{machineUserID}/roles 入口，真实用户接口对服务账号不开放
	if userEntity.IsMachine() {
		return nil, code.GetError(code.UserMemberOperationOnlyError)
	}
	roles, err := svc.listRoles(ctx, tenantID, req.UserID)
	if err != nil {
		return nil, err
	}
	return &dtotenant.UserRolesListResp{List: roles}, nil
}

// UpdateRoles 按应用全量替换用户角色（PUT 集合语义）。
// 角色从属于应用(role.app_id)，故授权以「用户 × 应用」为粒度：仅替换目标应用
// (role.app_id == req.AppID) 下的角色关联，其它应用的角色不受影响；
// req.AppID 为空串表示「系统/未归属应用组」。req.RoleIDs 必须全部属于当前租户
// 且归属目标应用，否则拒绝（防跨应用混授）。
func (svc *userSvc) UpdateRoles(ctx *gin.Context, req *dtotenant.UserRolesUpdateReq) error {
	// 系统管理操作：控制台管理层专用，直接调 API 的普通成员拒绝
	if err := requireSystemAdmin(ctx, code.UserRoleReplaceError); err != nil {
		return err
	}
	tenantID := gincontext.GetTenantIDString(ctx)
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateRoles] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return code.GetError(code.UserNotExistError)
	}
	// 服务账号角色走 /machine-users/{machineUserID}/roles 入口，真实用户接口对服务账号不开放
	if userEntity.IsMachine() {
		return code.GetError(code.UserMemberOperationOnlyError)
	}

	// 旧关联（事务外读取一次，供删除范围收敛与「最后一个内置管理员」保护使用）
	oldList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: req.UserID})
	if err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateRoles] dao GetListByCond old user_role fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserRoleReplaceError)
	}
	// 旧角色归属应用映射（缺失的历史角色按不属于任何组处理，不参与删除/保护）
	oldRoleAppMap, err := svc.roleAppMapOf(ctx, tenantID, oldList)
	if err != nil {
		return err
	}

	// 校验新角色均属于本租户且归属目标应用
	if len(req.RoleIDs) > 0 {
		roleList, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: req.RoleIDs})
		if err != nil {
			glog.Errorf(ctx, "[svcuser.UpdateRoles] dao GetListByCond roles fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.UserUpdateError)
		}
		if len(roleList) != len(req.RoleIDs) {
			return code.GetError(code.RoleNotExistError)
		}
		for i := range roleList {
			if roleList[i].AppID != req.AppID {
				return code.GetError(code.RoleNotExistError)
			}
		}
	}

	// 计算替换后的有效角色全集 =（其它应用现有角色）+（本应用新列表），用于管理员兜底保护
	effectiveRoleIDs := make([]string, 0, len(oldList)+len(req.RoleIDs))
	kept := make(map[string]struct{}, len(oldList))
	for _, ur := range oldList {
		if oldRoleAppMap[ur.RoleID] == req.AppID {
			continue // 本应用旧关联将被替换移除
		}
		if _, ok := kept[ur.RoleID]; ok {
			continue
		}
		kept[ur.RoleID] = struct{}{}
		effectiveRoleIDs = append(effectiveRoleIDs, ur.RoleID)
	}
	effectiveRoleIDs = append(effectiveRoleIDs, req.RoleIDs...)

	// 内置管理员保护：禁止移除「最后一个内置管理员角色持有者」的系统管理能力，防止平台锁死
	if keepLastAdmin, err := svc.hasOtherSystemAdminHolder(ctx, tenantID, req.UserID, effectiveRoleIDs); err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateRoles] check other admin holder fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserRoleReplaceError)
	} else if keepLastAdmin {
		return code.GetError(code.UserRoleRemoveLastAdminForbiddenError)
	}

	operator := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 仅删除目标应用下的旧关联，其它应用保持不变
		for _, ur := range oldList {
			if oldRoleAppMap[ur.RoleID] != req.AppID {
				continue
			}
			if err := dao.NewUserRoleDao().WithTx(tx).Delete(ctx, ur.ID, operator); err != nil {
				return err
			}
		}
		// 插入新关联
		for _, roleID := range req.RoleIDs {
			entity := &model.UserRoleEntity{
				TenantID:  tenantID,
				UserID:    req.UserID,
				RoleID:    roleID,
				CreatedBy: operator,
			}
			if err := dao.NewUserRoleDao().WithTx(tx).Insert(ctx, entity); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		glog.Errorf(ctx, "[svcuser.UpdateRoles] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return code.GetError(code.UserRoleReplaceError)
	}
	return nil
}

// roleAppMapOf 收集一批 user_role 关联所指向角色的归属应用（缺失角色忽略，不入映射）。
func (svc *userSvc) roleAppMapOf(ctx *gin.Context, tenantID string, urList []model.UserRoleEntity) (map[string]string, error) {
	result := make(map[string]string, len(urList))
	if len(urList) == 0 {
		return result, nil
	}
	roleIDs := make([]string, 0, len(urList))
	for _, ur := range urList {
		roleIDs = append(roleIDs, ur.RoleID)
	}
	roleList, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: roleIDs})
	if err != nil {
		glog.Errorf(ctx, "[svcuser.roleAppMapOf] dao GetListByCond roles fail, err:%v", err)
		return nil, code.GetError(code.UserRoleReplaceError)
	}
	for i := range roleList {
		result[roleList[i].ID] = roleList[i].AppID
	}
	return result, nil
}

// hasOtherSystemAdminHolder 判断本次「按应用全量替换」是否会移除目标用户持有的
// 「最后一个内置管理员角色」。newEffectiveRoleIDs 为替换后的有效角色全集
// （其它应用现有角色 + 本应用新列表），而非仅本应用的新列表。
// 返回 true 表示应拒绝该操作（防止平台系统管理能力永久锁死）。
// 规则：目标用户当前持有 ≥1 个内置管理员角色（source=builtin 且 admin_type=admin），且替换后的
// 有效集合不包含任何内置管理员角色，且当前租户内除目标用户外没有其他用户仍持有内置管理员角色 → 需保留，返回 true。
func (svc *userSvc) hasOtherSystemAdminHolder(ctx *gin.Context, tenantID, targetUserID string, newEffectiveRoleIDs []string) (bool, error) {
	// 1. 目标用户当前角色
	urList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: targetUserID})
	if err != nil {
		return false, err
	}
	if len(urList) == 0 {
		return false, nil
	}
	currentRoleIDs := make([]string, 0, len(urList))
	for _, r := range urList {
		currentRoleIDs = append(currentRoleIDs, r.RoleID)
	}
	// 2. 目标用户当前持有的内置管理员角色
	sysRoleIDs, err := svc.filterBuiltinSystemRoles(ctx, tenantID, currentRoleIDs)
	if err != nil {
		return false, err
	}
	if len(sysRoleIDs) == 0 {
		return false, nil // 目标用户本就不具备内置系统管理能力
	}
	// 3. 替换后的有效集合中是否还包含内置管理员角色
	newSysRoleIDs, err := svc.filterBuiltinSystemRoles(ctx, tenantID, newEffectiveRoleIDs)
	if err != nil {
		return false, err
	}
	if len(newSysRoleIDs) > 0 {
		return false, nil // 有效集合仍保留系统管理能力
	}
	// 4. 是否还有其他用户持有任一内置管理员角色
	allSysRoles, err := svc.listTenantBuiltinSystemRoles(ctx, tenantID)
	if err != nil {
		return false, err
	}
	if len(allSysRoles) > 0 {
		sysRoleSet := make(map[string]struct{}, len(allSysRoles))
		for _, id := range allSysRoles {
			sysRoleSet[id] = struct{}{}
		}
		others, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID})
		if err != nil {
			return false, err
		}
		for _, ur := range others {
			if ur.UserID == targetUserID {
				continue
			}
			if _, ok := sysRoleSet[ur.RoleID]; ok {
				return false, nil // 其他用户仍持有管理员角色，允许释放目标用户
			}
		}
	}
	return true, nil
}

// filterBuiltinSystemRoles 从 roleIDs 中筛出「内置 + 系统管理」的角色 ID（内置管理员：source=builtin && admin_type=admin）。
func (svc *userSvc) filterBuiltinSystemRoles(ctx *gin.Context, tenantID string, roleIDs []string) ([]string, error) {
	result := make([]string, 0)
	if len(roleIDs) == 0 {
		return result, nil
	}
	roles, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: roleIDs})
	if err != nil {
		return nil, err
	}
	for i := range roles {
		r := &roles[i]
		if r.IsBuiltinAdmin() {
			result = append(result, r.ID)
		}
	}
	return result, nil
}

// listTenantBuiltinSystemRoles 返回当前租户内全部内置管理员角色 ID。
func (svc *userSvc) listTenantBuiltinSystemRoles(ctx *gin.Context, tenantID string) ([]string, error) {
	roles, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{
		TenantID:  tenantID,
		Source:    model.RoleSourceBuiltin,
		AdminType: model.SysAdminTypeAdmin,
	})
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(roles))
	for _, r := range roles {
		result = append(result, r.ID)
	}
	return result, nil
}

// listRoles 查询用户已分配角色（含角色基础信息）。
func (svc *userSvc) listRoles(ctx *gin.Context, tenantID, userID string) ([]dtotenant.UserRoleItem, error) {
	urList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: userID})
	if err != nil {
		glog.Errorf(ctx, "[svcuser.listRoles] query user_role fail, err:%v", err)
		return nil, code.GetError(code.UserRoleGetPageListError)
	}
	if len(urList) == 0 {
		return []dtotenant.UserRoleItem{}, nil
	}
	roleIDs := make([]string, 0, len(urList))
	for _, r := range urList {
		roleIDs = append(roleIDs, r.RoleID)
	}
	roleList, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: roleIDs})
	if err != nil {
		glog.Errorf(ctx, "[svcuser.listRoles] query role fail, err:%v", err)
		return nil, code.GetError(code.UserRoleGetPageListError)
	}
	roleMap := make(map[string]*model.RoleEntity, len(roleList))
	for i := range roleList {
		roleMap[roleList[i].ID] = &roleList[i]
	}
	appNameMap, err := tenantAppNameMap(ctx)
	if err != nil {
		return nil, code.GetError(code.UserRoleGetPageListError)
	}
	list := make([]dtotenant.UserRoleItem, 0, len(urList))
	for _, r := range urList {
		role := roleMap[r.RoleID]
		if role == nil {
			continue
		}
		list = append(list, dtotenant.UserRoleItem{
			RoleID:      role.ID,
			AppID:       role.AppID,
			AppName:     appNameMap[role.AppID],
			Name:        role.Name,
			Description: role.Description,
		})
	}
	return list, nil
}

// loadUserPersonMaps 批量加载用户与其关联自然人（IN 查询，避免 N+1）。
func (svc *userSvc) loadUserPersonMaps(ctx *gin.Context, userIDs []string) (map[string]*model.UserEntity, map[string]*model.PersonEntity) {
	userMap := make(map[string]*model.UserEntity)
	personMap := make(map[string]*model.PersonEntity)
	if len(userIDs) == 0 {
		return userMap, personMap
	}
	userList, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{IDs: userIDs})
	if err != nil {
		glog.Warnf(ctx, "[svcuser.loadUserPersonMaps] user GetListByCond fail, err:%v", err)
		return userMap, personMap
	}
	personIDs := make([]string, 0, len(userList))
	for i := range userList {
		userMap[userList[i].ID] = &userList[i]
		if userList[i].PersonID != "" {
			personIDs = append(personIDs, userList[i].PersonID)
		}
	}
	if len(personIDs) > 0 {
		personList, err := dao.NewPersonDao().GetListByCond(ctx, &dao.PersonCond{IDs: personIDs})
		if err != nil {
			glog.Warnf(ctx, "[svcuser.loadUserPersonMaps] person GetListByCond fail, err:%v", err)
			return userMap, personMap
		}
		for i := range personList {
			personMap[personList[i].ID] = &personList[i]
		}
	}
	return userMap, personMap
}

// ListLoginLogs 返回租户内某用户的登录日志（只读）。
// 先校验用户属于当前租户，再以「租户 + 用户」双重条件查询，避免跨租户读取。
func (svc *userSvc) ListLoginLogs(ctx *gin.Context, req *dtotenant.UserLoginLogListReq) (*dtotenant.UserLoginLogListResp, error) {
	if _, err := resolveTenantUser(ctx, req.UserID); err != nil {
		return nil, err
	}

	entityList, total, err := dao.NewUserLoginLogDao().GetPageListByCond(ctx, &dao.UserLoginLogCond{
		BaseCond: &gormdao.BaseCond{Page: 1, PageSize: userLoginLogPageSize},
		TenantID: gincontext.GetTenantIDString(ctx),
		UserID:   req.UserID,
	})
	if err != nil {
		glog.Errorf(ctx, "[svcuser.ListLoginLogs] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserLoginLogGetPageListError)
	}

	list := make([]dtotenant.UserLoginLogItem, 0, len(entityList))
	for _, v := range entityList {
		list = append(list, dtotenant.UserLoginLogItem{
			UserLoginLogID: v.ID,
			LoginIP:        v.LoginIP,
			UserAgent:      v.UserAgent,
			LoginTime:      v.LoginTime.Unix(),
		})
	}
	return &dtotenant.UserLoginLogListResp{List: list, Total: total}, nil
}
