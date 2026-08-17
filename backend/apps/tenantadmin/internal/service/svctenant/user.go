package svctenant

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/dao"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"github.com/morehao/ark-iam/tenantadmin/internal/dto/dtotenant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"github.com/morehao/golib/gutil"
	"gorm.io/gorm"
)

type UserSvc interface {
	PageList(ctx *gin.Context, req *dtotenant.UserPageListReq) (*dtotenant.UserPageListResp, error)
	Create(ctx *gin.Context, req *dtotenant.UserCreateReq) (*dtotenant.UserCreateResp, error)
	Detail(ctx *gin.Context, req *dtotenant.UserDetailReq) (*dtotenant.UserDetailResp, error)
	Update(ctx *gin.Context, req *dtotenant.UserUpdateReq) error
	ResetPassword(ctx *gin.Context, req *dtotenant.UserResetPasswordReq) error
	ListRoles(ctx *gin.Context, req *dtotenant.UserRolesListReq) (*dtotenant.UserRolesListResp, error)
	UpdateRoles(ctx *gin.Context, req *dtotenant.UserRolesUpdateReq) error
}

type userSvc struct {
}

var _ UserSvc = (*userSvc)(nil)

func NewUserSvc() UserSvc {
	return &userSvc{}
}

// PageList 返回当前租户内的用户目录（含自然人基础信息），支持关键词（姓名/用户名/邮箱/手机）与状态过滤。
func (svc *userSvc) PageList(ctx *gin.Context, req *dtotenant.UserPageListReq) (*dtotenant.UserPageListResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	cond := &dao.UserCond{
		BaseCond: &gormdao.BaseCond{
			Page:     req.Page,
			PageSize: req.PageSize,
		},
		TenantID:    tenantID,
		Keyword:     req.Keyword,
		IsSuspended: req.IsSuspended,
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

	// 主组织 + 角色数 聚合（批量，避免 N+1）
	primaryOrgNameMap := loadPrimaryOrgNameMap(ctx, tenantID, userIDs)
	roleCountMap := loadUserRoleCountMap(ctx, tenantID, userIDs)

	list := make([]dtotenant.UserPageListItem, 0, len(userEntityList))
	for _, v := range userEntityList {
		person := personMap[v.PersonID]
		if person == nil {
			person = &model.PersonEntity{}
		}
		list = append(list, dtotenant.UserPageListItem{
			UserID:         v.ID,
			TenantID:       v.TenantID,
			Username:       model.DerefStr(person.Username),
			PrimaryEmail:   model.DerefStr(person.PrimaryEmail),
			PrimaryPhone:   model.DerefStr(person.PrimaryPhone),
			Name:           v.Name,
			Avatar:         v.Avatar,
			IsSuspended:    v.IsSuspended,
			PrimaryOrgName: primaryOrgNameMap[v.ID],
			RoleCount:      roleCountMap[v.ID],
			CreatedAt:      v.CreatedAt.Unix(),
		})
	}
	return &dtotenant.UserPageListResp{
		List:  list,
		Total: total,
	}, nil
}

// loadPrimaryOrgNameMap 批量查询用户主组织名称（member 关系 + is_primary 唯一行）。
func loadPrimaryOrgNameMap(ctx *gin.Context, tenantID string, userIDs []string) map[string]string {
	result := make(map[string]string, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}
	primary := true
	relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:     tenantID,
		RelationType: string(model.OrgUserRelationMember),
		IsPrimary:    &primary,
	})
	if err != nil {
		glog.Warnf(ctx, "[svcuser.loadPrimaryOrgNameMap] query primary org fail, err:%v", err)
		return result
	}
	userOrgMap := make(map[string]string, len(relationList))
	orgIDs := make([]string, 0, len(relationList))
	for _, r := range relationList {
		if _, ok := userOrgMap[r.UserID]; ok {
			continue
		}
		userOrgMap[r.UserID] = r.OrganizationID
		orgIDs = append(orgIDs, r.OrganizationID)
	}
	if len(orgIDs) == 0 {
		return result
	}
	var orgList []model.OrganizationEntity
	if err := dbclient.IamDB(ctx).Model(&model.OrganizationEntity{}).
		Where("tenant_id = ? AND id IN ?", tenantID, orgIDs).
		Find(&orgList).Error; err != nil {
		glog.Warnf(ctx, "[svcuser.loadPrimaryOrgNameMap] query org name fail, err:%v", err)
		return result
	}
	orgNameMap := make(map[string]string, len(orgList))
	for _, o := range orgList {
		orgNameMap[o.ID] = o.Name
	}
	for userID, orgID := range userOrgMap {
		result[userID] = orgNameMap[orgID]
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
// 同时按 organizationIDs 建立组织归属（首个为主组织）。
// 业务约束：用户必须从属于至少一个部门，organizationIDs 必传。
func (svc *userSvc) Create(ctx *gin.Context, req *dtotenant.UserCreateReq) (*dtotenant.UserCreateResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	operatorID := gincontext.GetUserIDString(ctx)

	// 0. 用户必须从属于至少一个部门（业务约束，防绕过 DTO 校验）
	if len(req.OrganizationIDs) == 0 {
		return nil, code.GetError(code.UserOrganizationRequiredError)
	}

	// 1. 解析 personID（find-or-create：默认以姓名创建自然人，命中已有全局身份则复用）
	personID := req.PersonID
	createPerson := false
	if personID != "" {
		person, err := dao.NewPersonDao().GetByID(ctx, personID)
		if err != nil || person == nil || person.ID == "" {
			return nil, code.GetError(code.UserNotExistError)
		}
	} else {
		personID = svc.findExistingPersonID(ctx, req)
		if personID == "" {
			createPerson = true
		}
	}

	// 2. 同一自然人在本租户内只能有一条 user
	if personID != "" {
		existing, err := dao.NewUserDao().GetListByCond(ctx, &dao.UserCond{TenantID: tenantID, PersonID: personID})
		if err != nil {
			glog.Errorf(ctx, "[svcuser.Create] query user by person fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.UserCreateError)
		}
		if len(existing) > 0 {
			return nil, code.GetError(code.UserAlreadyInTenantError)
		}
	}

	// 3. 校验归属组织均属于本租户
	orgSet := make(map[string]bool)
	if len(req.OrganizationIDs) > 0 {
		orgList, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{TenantID: tenantID})
		if err != nil {
			glog.Errorf(ctx, "[svcuser.Create] query org fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.UserCreateError)
		}
		for _, o := range orgList {
			orgSet[o.ID] = true
		}
		for _, orgID := range req.OrganizationIDs {
			if !orgSet[orgID] {
				return nil, code.GetError(code.OrganizationNotExistError)
			}
		}
	}

	// 4. 事务：创建 person（如需）+ 创建 user + 建立组织归属
	var createdUserID string
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		if createPerson {
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
				Profile:           json.RawMessage(`{}`),
				CustomData:        json.RawMessage(`{}`),
				CreatedBy:         operatorID,
			}
			if insertErr := dao.NewPersonDao().WithTx(tx).Insert(ctx, personEntity); insertErr != nil {
				glog.Errorf(ctx, "[svcuser.Create] person Insert fail, err:%v", insertErr)
				return fmt.Errorf("person insert: %w", insertErr)
			}
			personID = personEntity.ID
		}

		now := time.Now()
		insertEntity := &model.UserEntity{
			TenantID:    tenantID,
			PersonID:    personID,
			Name:        req.Name,
			Avatar:      req.Avatar,
			Profile:     json.RawMessage(`{}`),
			CustomData:  json.RawMessage(`{}`),
			IsSuspended: req.IsSuspended,
			IsOwner:     false,
			JoinedAt:    &now,
			CreatedBy:   operatorID,
		}
		if insertErr := dao.NewUserDao().WithTx(tx).Insert(ctx, insertEntity); insertErr != nil {
			glog.Errorf(ctx, "[svcuser.Create] dao Insert fail, err:%v, req:%s", insertErr, gutil.ToJsonString(req))
			return fmt.Errorf("user insert: %w", insertErr)
		}
		createdUserID = insertEntity.ID

		// 建立组织归属（member 关系，首个为主组织）
		for i, orgID := range req.OrganizationIDs {
			relation := &model.OrganizationUserEntity{
				TenantID:       tenantID,
				OrganizationID: orgID,
				UserID:         insertEntity.ID,
				RelationType:   string(model.OrgUserRelationMember),
				IsPrimary:      i == 0,
				CreatedBy:      operatorID,
			}
			if insertErr := dao.NewOrganizationUserDao().WithTx(tx).Insert(ctx, relation); insertErr != nil {
				glog.Errorf(ctx, "[svcuser.Create] org relation Insert fail, err:%v", insertErr)
				return fmt.Errorf("org relation insert: %w", insertErr)
			}
		}
		return nil
	})
	if txErr != nil {
		if txErr == code.GetError(code.PasswordHashError) {
			return nil, txErr
		}
		glog.Errorf(ctx, "[svcuser.Create] transaction fail, err:%v, req:%s", txErr, gutil.ToJsonString(req))
		return nil, code.GetError(code.UserCreateError)
	}

	return &dtotenant.UserCreateResp{
		UserID: createdUserID,
	}, nil
}

// Detail 用户详情：基础信息（含自然人）+ 组织归属 + 已分配角色。
func (svc *userSvc) Detail(ctx *gin.Context, req *dtotenant.UserDetailReq) (*dtotenant.UserDetailResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil || userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
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

	// 组织归属
	organizations, err := loadUserOrganizations(ctx, tenantID, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] load organizations fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	resp.Organizations = organizations

	// 已分配角色
	roles, err := svc.listRoles(ctx, tenantID, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.Detail] list roles fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	resp.Roles = roles
	return resp, nil
}

// Update 局部更新用户（PATCH）：姓名/头像/状态。
func (svc *userSvc) Update(ctx *gin.Context, req *dtotenant.UserUpdateReq) error {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil || userEntity == nil || userEntity.ID == "" || userEntity.TenantID != gincontext.GetTenantIDString(ctx) {
		return code.GetError(code.UserNotExistError)
	}

	updateMap := map[string]any{"updated_by": gincontext.GetUserIDString(ctx)}
	if req.Name != "" {
		updateMap["name"] = req.Name
	}
	if req.Avatar != "" {
		updateMap["avatar"] = req.Avatar
	}
	if req.IsSuspended != nil {
		updateMap["is_suspended"] = *req.IsSuspended
	}
	if err := dao.NewUserDao().UpdateMap(ctx, req.UserID, updateMap); err != nil {
		glog.Errorf(ctx, "[svcuser.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	return nil
}

// ResetPassword 重置密码：更新关联 person 的密码哈希（无自然人关联的用户不可登录，直接拒绝）。
func (svc *userSvc) ResetPassword(ctx *gin.Context, req *dtotenant.UserResetPasswordReq) error {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil || userEntity == nil || userEntity.ID == "" || userEntity.TenantID != gincontext.GetTenantIDString(ctx) {
		return code.GetError(code.UserNotExistError)
	}
	if userEntity.PersonID == "" {
		return code.GetError(code.UserNotExistError)
	}

	hash, hashErr := gcrypto.GeneratePasswordHash(req.Password)
	if hashErr != nil {
		glog.Errorf(ctx, "[svcuser.ResetPassword] GeneratePasswordHash fail, err:%v", hashErr)
		return code.GetError(code.PasswordHashError)
	}
	if err := dao.NewPersonDao().UpdateMap(ctx, userEntity.PersonID, map[string]any{
		"password_encrypted": hash,
		"password_method":    "bcrypt",
		"updated_by":         gincontext.GetUserIDString(ctx),
	}); err != nil {
		glog.Errorf(ctx, "[svcuser.ResetPassword] person UpdateMap fail, err:%v", err)
		return code.GetError(code.UserResetPasswordError)
	}
	return nil
}

// ListRoles 用户已分配角色列表。
func (svc *userSvc) ListRoles(ctx *gin.Context, req *dtotenant.UserRolesListReq) (*dtotenant.UserRolesListResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil || userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return nil, code.GetError(code.UserNotExistError)
	}
	roles, err := svc.listRoles(ctx, tenantID, req.UserID)
	if err != nil {
		return nil, err
	}
	return &dtotenant.UserRolesListResp{List: roles}, nil
}

// UpdateRoles 全量替换用户角色（PUT 集合语义）。
func (svc *userSvc) UpdateRoles(ctx *gin.Context, req *dtotenant.UserRolesUpdateReq) error {
	tenantID := gincontext.GetTenantIDString(ctx)
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil || userEntity == nil || userEntity.ID == "" || userEntity.TenantID != tenantID {
		return code.GetError(code.UserNotExistError)
	}

	// 校验角色均属于本租户
	if len(req.RoleIDs) > 0 {
		roleList, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: req.RoleIDs})
		if err != nil || len(roleList) != len(req.RoleIDs) {
			return code.GetError(code.RoleNotExistError)
		}
	}

	userID := gincontext.GetUserIDString(ctx)
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除旧关联
		oldList, err := dao.NewUserRoleDao().GetListByCond(ctx, &dao.UserRoleCond{TenantID: tenantID, UserID: req.UserID})
		if err != nil {
			return err
		}
		for _, r := range oldList {
			if err := dao.NewUserRoleDao().WithTx(tx).Delete(ctx, r.ID, userID); err != nil {
				return err
			}
		}
		// 插入新关联
		for _, roleID := range req.RoleIDs {
			entity := &model.UserRoleEntity{
				TenantID:  tenantID,
				UserID:    req.UserID,
				RoleID:    roleID,
				CreatedBy: userID,
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
			Code:        role.Code,
			Description: role.Description,
			Type:        role.Type,
		})
	}
	return list, nil
}

// findExistingPersonID 按 username/email/phone 全局唯一标识查找已有 person（命中其一即返回其 ID）。
func (svc *userSvc) findExistingPersonID(ctx *gin.Context, req *dtotenant.UserCreateReq) string {
	personDao := dao.NewPersonDao()
	if req.Username != "" {
		if p, _ := personDao.GetByCond(ctx, &dao.PersonCond{Username: req.Username}); p != nil && p.ID != "" {
			return p.ID
		}
	}
	if req.PrimaryEmail != "" {
		if p, _ := personDao.GetByCond(ctx, &dao.PersonCond{PrimaryEmail: req.PrimaryEmail}); p != nil && p.ID != "" {
			return p.ID
		}
	}
	if req.PrimaryPhone != "" {
		if p, _ := personDao.GetByCond(ctx, &dao.PersonCond{PrimaryPhone: req.PrimaryPhone}); p != nil && p.ID != "" {
			return p.ID
		}
	}
	return ""
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
