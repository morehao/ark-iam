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
	"github.com/morehao/ark-iam/pkg/iam/person"
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

// PageList 返回当前租户内的用户目录（含自然人基础信息），支持关键词（姓名/用户名/邮箱/手机）与状态过滤；
// 传 organizationID 时仅返回"恰在该部门"的用户（含 primary/secondary/leader 任一关系，不含子部门）。
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
	if req.OrganizationID != "" {
		relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
			TenantID:       tenantID,
			OrganizationID: req.OrganizationID,
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

// loadPrimaryOrgNameMap 批量查询用户行政归属组织名称（member 唯一行）。
func loadPrimaryOrgNameMap(ctx *gin.Context, tenantID string, userIDs []string) map[string]string {
	result := make(map[string]string, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}
	relationList, err := dao.NewOrganizationUserDao().GetListByCond(ctx, &dao.OrganizationUserCond{
		TenantID:     tenantID,
		RelationType: model.OrgUserRelationPrimary,
		UserIDs:      userIDs,
	})
	if err != nil {
		glog.Warnf(ctx, "[svcuser.loadPrimaryOrgNameMap] query member org fail, err:%v", err)
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
// 同时按 organizationIDs 建立行政归属（member，至多 1 个）、leaderOrgIDs 建立负责关系（leader）。
// 业务约束：用户必须从属于至少一个部门，organizationIDs 必传。
func (svc *userSvc) Create(ctx *gin.Context, req *dtotenant.UserCreateReq) (*dtotenant.UserCreateResp, error) {
	tenantID := gincontext.GetTenantIDString(ctx)
	operatorID := gincontext.GetUserIDString(ctx)

	// 0. 用户必须从属于至少一个部门（业务约束，防绕过 DTO 校验）
	if len(req.OrganizationIDs) == 0 {
		return nil, code.GetError(code.UserOrganizationRequiredError)
	}
	// 0.1 邮箱或手机号至少填写一个（联系方式是用户识别/找回的必需信息）
	if req.PrimaryEmail == "" && req.PrimaryPhone == "" {
		return nil, code.GetError(code.UserContactRequiredError)
	}

	// 1. 解析 personID：显式提供时校验存在并直接关联；未提供时在事务内 find-or-create
	// （person 领域能力，见 pkg/iam/person）：命中已有全局身份则复用，未命中则同事务新建。
	personID := req.PersonID
	resolvePersonInTx := false
	if personID != "" {
		person, err := dao.NewPersonDao().GetByID(ctx, personID)
		if err != nil {
			glog.Errorf(ctx, "[svcuser.Create] dao GetByID person fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return nil, code.GetError(code.UserCreateError)
		}
		if person == nil || person.ID == "" {
			return nil, code.GetError(code.UserNotExistError)
		}
	} else {
		resolvePersonInTx = true
	}

	// 2. 校验归属组织均属于本租户
	orgSet := make(map[string]bool)
	if len(req.OrganizationIDs) > 0 || len(req.SecondaryOrgIDs) > 0 || len(req.LeaderOrgIDs) > 0 {
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
		for _, orgID := range req.SecondaryOrgIDs {
			if !orgSet[orgID] {
				return nil, code.GetError(code.OrganizationNotExistError)
			}
		}
		for _, orgID := range req.LeaderOrgIDs {
			if !orgSet[orgID] {
				return nil, code.GetError(code.OrganizationNotExistError)
			}
		}
	}

	// 3. 事务：person find-or-create + 创建 user + 建立组织归属
	var createdUserID string
	txErr := dbclient.IamDB(ctx).Transaction(func(tx *gorm.DB) error {
		if resolvePersonInTx {
			passwordHash := ""
			if req.Password != "" {
				hash, hashErr := gcrypto.GeneratePasswordHash(req.Password)
				if hashErr != nil {
					glog.Errorf(ctx, "[svcuser.Create] GeneratePasswordHash fail, err:%v", hashErr)
					return code.GetError(code.PasswordHashError)
				}
				passwordHash = hash
			}
			personEntity, _, personErr := person.FindOrCreate(ctx, tx, &person.FindOrCreateReq{
				Username:          req.Username,
				PrimaryEmail:      req.PrimaryEmail,
				PrimaryPhone:      req.PrimaryPhone,
				PasswordEncrypted: passwordHash,
				PasswordMethod:    "bcrypt",
				Name:              req.Name,
				Avatar:            req.Avatar,
				CreatedBy:         operatorID,
			})
			if personErr != nil {
				glog.Errorf(ctx, "[svcuser.Create] person FindOrCreate fail, err:%v, req:%s", personErr, gutil.ToJsonString(req))
				return fmt.Errorf("person find-or-create: %w", personErr)
			}
			personID = personEntity.ID
		}

		// 同一自然人在本租户内只能有一条 user（person 关联/新建后即可确定 personID 校验）
		existing, err := dao.NewUserDao().WithTx(tx).GetListByCond(ctx, &dao.UserCond{TenantID: tenantID, PersonID: personID})
		if err != nil {
			glog.Errorf(ctx, "[svcuser.Create] query user by person fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return fmt.Errorf("query user by person: %w", err)
		}
		if len(existing) > 0 {
			return code.GetError(code.UserAlreadyInTenantError)
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

		// 建立行政主部门（primary 关系，至多 1 条）
		if len(req.OrganizationIDs) > 1 {
			return code.GetError(code.UserOrganizationRequiredError)
		}
		for _, orgID := range req.OrganizationIDs {
			relation := &model.OrganizationUserEntity{
				TenantID:       tenantID,
				OrganizationID: orgID,
				UserID:         insertEntity.ID,
				RelationType:   model.OrgUserRelationPrimary,
				CreatedBy:      operatorID,
			}
			if insertErr := dao.NewOrganizationUserDao().WithTx(tx).Insert(ctx, relation); insertErr != nil {
				glog.Errorf(ctx, "[svcuser.Create] org relation Insert fail, err:%v", insertErr)
				return fmt.Errorf("org relation insert: %w", insertErr)
			}
		}
		// 建立负责部门关系（leader，独立于归属；一个部门至多一个负责人）
		for _, orgID := range req.LeaderOrgIDs {
			if err := ensureOrgLeaderUnique(ctx, tx, tenantID, orgID, insertEntity.ID); err != nil {
				return err
			}
			relation := &model.OrganizationUserEntity{
				TenantID:       tenantID,
				OrganizationID: orgID,
				UserID:         insertEntity.ID,
				RelationType:   model.OrgUserRelationLeader,
				CreatedBy:      operatorID,
			}
			if insertErr := dao.NewOrganizationUserDao().WithTx(tx).Insert(ctx, relation); insertErr != nil {
				glog.Errorf(ctx, "[svcuser.Create] leader relation Insert fail, err:%v", insertErr)
				return fmt.Errorf("leader relation insert: %w", insertErr)
			}
		}
		// 建立参与部门关系（secondary，可多条）
		for _, orgID := range req.SecondaryOrgIDs {
			relation := &model.OrganizationUserEntity{
				TenantID:       tenantID,
				OrganizationID: orgID,
				UserID:         insertEntity.ID,
				RelationType:   model.OrgUserRelationSecondary,
				CreatedBy:      operatorID,
			}
			if insertErr := dao.NewOrganizationUserDao().WithTx(tx).Insert(ctx, relation); insertErr != nil {
				glog.Errorf(ctx, "[svcuser.Create] secondary relation Insert fail, err:%v", insertErr)
				return fmt.Errorf("secondary relation insert: %w", insertErr)
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

// Update 局部更新用户（PATCH）：姓名/头像/状态 + 主/参与/负责部门更新。
func (svc *userSvc) Update(ctx *gin.Context, req *dtotenant.UserUpdateReq) error {
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
	if req.PrimaryOrgID != nil && *req.PrimaryOrgID == "" {
		return code.GetError(code.UserOrganizationRequiredError)
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

	// 校验传入的组织均属本租户。
	orgSet := make(map[string]bool)
	if (req.PrimaryOrgID != nil && *req.PrimaryOrgID != "") || req.SecondaryOrgIDs != nil || req.LeaderOrgIDs != nil {
		orgList, err := dao.NewOrganizationDao().GetListByCond(ctx, &dao.OrganizationCond{TenantID: tenantID})
		if err != nil {
			glog.Errorf(ctx, "[svcuser.Update] query org fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.UserUpdateError)
		}
		for _, o := range orgList {
			orgSet[o.ID] = true
		}
		checkOwnership := func(id string) bool { return id == "" || orgSet[id] }
		if req.PrimaryOrgID != nil && !checkOwnership(*req.PrimaryOrgID) {
			return code.GetError(code.OrganizationNotExistError)
		}
		for _, id := range derefSlice(req.SecondaryOrgIDs) {
			if !orgSet[id] {
				return code.GetError(code.OrganizationNotExistError)
			}
		}
		for _, id := range derefSlice(req.LeaderOrgIDs) {
			if !orgSet[id] {
				return code.GetError(code.OrganizationNotExistError)
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
		if req.PrimaryOrgID != nil {
			// 替换主部门：删旧 primary（至多 1 行）后建新。
			if err := replaceOrgRelationList(ctx, tx, tenantID, req.UserID, []string{*req.PrimaryOrgID}, model.OrgUserRelationPrimary, operatorID); err != nil {
				return err
			}
		}
		if req.SecondaryOrgIDs != nil {
			if err := replaceOrgRelationList(ctx, tx, tenantID, req.UserID, *req.SecondaryOrgIDs, model.OrgUserRelationSecondary, operatorID); err != nil {
				return err
			}
		}
		if req.LeaderOrgIDs != nil {
			if err := replaceOrgRelationList(ctx, tx, tenantID, req.UserID, *req.LeaderOrgIDs, model.OrgUserRelationLeader, operatorID); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		if txErr == code.GetError(code.OrganizationUserLeaderConflictError) {
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

// ResetPassword 重置密码：更新关联 person 的密码哈希（无自然人关联的用户不可登录，直接拒绝）。
func (svc *userSvc) ResetPassword(ctx *gin.Context, req *dtotenant.UserResetPasswordReq) error {
	userEntity, err := dao.NewUserDao().GetByID(ctx, req.UserID)
	if err != nil {
		glog.Errorf(ctx, "[svcuser.ResetPassword] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserUpdateError)
	}
	if userEntity == nil || userEntity.ID == "" || userEntity.TenantID != gincontext.GetTenantIDString(ctx) {
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

// UpdateRoles 全量替换用户角色（PUT 集合语义）。
func (svc *userSvc) UpdateRoles(ctx *gin.Context, req *dtotenant.UserRolesUpdateReq) error {
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

	// 校验角色均属于本租户
	if len(req.RoleIDs) > 0 {
		roleList, err := dao.NewRoleDao().GetListByCond(ctx, &dao.RoleCond{TenantID: tenantID, IDs: req.RoleIDs})
		if err != nil {
			glog.Errorf(ctx, "[svcuser.UpdateRoles] dao GetListByCond roles fail, err:%v, req:%s", err, gutil.ToJsonString(req))
			return code.GetError(code.UserUpdateError)
		}
		if len(roleList) != len(req.RoleIDs) {
			return code.GetError(code.RoleNotExistError)
		}
	}

	// 内置管理员保护：禁止移除「最后一个内置系统管理角色持有者」的系统管理能力，防止平台锁死
	if keepLastAdmin, err := svc.hasOtherSystemAdminHolder(ctx, tenantID, req.UserID, req.RoleIDs); err != nil {
		glog.Errorf(ctx, "[svcuser.UpdateRoles] check other admin holder fail, err:%v, req:%s", err, gutil.ToJsonString(req))
		return code.GetError(code.UserRoleReplaceError)
	} else if keepLastAdmin {
		return code.GetError(code.UserRoleRemoveLastAdminForbiddenError)
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

// hasOtherSystemAdminHolder 判断本次角色全量替换是否会移除目标用户持有的「最后一个内置系统管理角色」。
// 返回 true 表示应拒绝该操作（防止平台系统管理能力永久锁死）。
// 规则：目标用户当前持有 ≥1 个内置系统管理角色（source=builtin 且 admin_level>=basic），且新角色列表不包含任何内置系统管理角色，
// 且当前租户内除目标用户外没有其他用户仍持有内置系统管理角色 → 需保留，返回 true。
func (svc *userSvc) hasOtherSystemAdminHolder(ctx *gin.Context, tenantID, targetUserID string, newRoleIDs []string) (bool, error) {
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
	// 2. 目标用户当前持有的内置系统管理角色
	sysRoleIDs, err := svc.filterBuiltinSystemRoles(ctx, tenantID, currentRoleIDs)
	if err != nil {
		return false, err
	}
	if len(sysRoleIDs) == 0 {
		return false, nil // 目标用户本就不具备内置系统管理能力
	}
	// 3. 新列表中是否还包含内置系统管理角色
	newSysRoleIDs, err := svc.filterBuiltinSystemRoles(ctx, tenantID, newRoleIDs)
	if err != nil {
		return false, err
	}
	if len(newSysRoleIDs) > 0 {
		return false, nil // 新列表仍保留系统管理能力
	}
	// 4. 是否还有其他用户持有任一内置系统管理角色
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
				return false, nil // 其他用户仍持有系统管理角色，允许释放目标用户
			}
		}
	}
	return true, nil
}

// filterBuiltinSystemRoles 从 roleIDs 中筛出「内置 + 系统管理」的角色 ID（内置管理员：source=builtin && admin_level=super）。
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
		TenantID:          tenantID,
		Source:            string(model.RoleSourceBuiltin),
		AdminLevelAtLeast: string(model.SysAdminLevelSuper),
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
			Code:        role.Code,
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
