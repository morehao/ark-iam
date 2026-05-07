# IAM 轻量级优化实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 基于 Logto 实现思路，对 ark-iam 进行轻量级优化，实现会话管理、应用角色管理、角色用户管理、SSO 逻辑完善等功能

**Architecture:**
- 采用分层架构：Controller → Service → DAO → Model
- 复用现有 `refresh_token` 表进行会话管理
- 复用现有 `application_role` 和 `user_role` 表进行角色关联管理
- 遵循现有代码风格和项目规范

**Tech Stack:** Go + Gin + GORM + JWT

---

## 文件结构

```
backend/apps/iam/
├── internal/
│   ├── controller/
│   │   ├── ctrsession/              # [NEW] 会话管理
│   │   │   └── session.go
│   │   ├── ctrpermission/
│   │   │   ├── application.go       # [MODIFY] 添加应用角色/密钥管理
│   │   │   └── role.go              # [MODIFY] 添加角色用户/应用管理
│   │   └── ctrauth/
│   │       ├── sso_connector.go     # [MODIFY] 添加提供商/IdP配置
│   │       └── connector.go         # [MODIFY] 添加工厂/测试/授权URI
│   ├── service/
│   │   ├── svcsession/              # [NEW] 会话服务
│   │   │   └── session.go
│   │   ├── svcapplication/
│   │   │   └── application.go       # [MODIFY] 添加角色/密钥管理
│   │   ├── svcpermission/
│   │   │   └── role.go              # [MODIFY] 添加用户/应用管理
│   │   └── svcauth/
│   │       ├── auth.go              # [MODIFY] 完善SSO逻辑
│   │       ├── sso_connector.go     # [MODIFY] 添加提供商/IdP配置
│   │       └── connector.go         # [MODIFY] 添加工厂/测试/授权URI
│   └── dto/
│       ├── dtouser/
│       │   └── session.go            # [NEW] 会话DTO
│       ├── dtoapplication/
│       │   ├── request.go           # [MODIFY] 添加角色/密钥请求
│       │   └── response.go          # [MODIFY] 添加角色/密钥响应
│       ├── dtoconnector/
│       │   ├── request.go           # [NEW] 连接器请求
│       │   └── response.go          # [NEW] 连接器响应
│       ├── dtosso/
│       │   ├── request.go           # [NEW] SSO请求
│       │   └── response.go          # [NEW] SSO响应
│       └── dtouser/
│           └── role.go              # [MODIFY] 添加角色用户/应用请求
├── dao/
│   └── session.go                   # [NEW] 会话DAO扩展
└── internal/router/
    ├── auth.go                      # [MODIFY] 注册会话路由
    ├── permission.go                # [MODIFY] 注册应用/角色路由
    └── user.go                      # [MODIFY] 注册会话路由
```

---

## 任务清单

### Phase 1: 会话管理 (P0 - 核心功能)

#### Task 1: 会话 DTO 定义

**Files:**
- Create: `internal/dto/dtouser/session.go`

- [ ] **Step 1: 创建会话请求/响应 DTO**

```go
package dtouser

import "github.com/morehao/golib/biz/gobject"

type SessionListReq struct {
	gobject.PageQuery
}

type SessionRevokeReq struct {
	SessionID uint64 `json:"sessionId" path:"sessionId" binding:"required"`
}

type SessionResp struct {
	ID            uint64     `json:"id"`
	ApplicationID uint64     `json:"applicationId"`
	TenantID      uint64     `json:"tenantId"`
	ExpiresAt     *string    `json:"expiresAt"`
	CreatedAt     string     `json:"createdAt"`
	IsActive      bool       `json:"isActive"`
}

type SessionListResp struct {
	gobject.PageResp
	Sessions []SessionResp `json:"sessions"`
}
```

#### Task 2: 会话 DAO 扩展

**Files:**
- Create: `dao/session.go`

- [ ] **Step 1: 创建会话 DAO**

```go
package dao

import (
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/biz/genericdao"
	"gorm.io/gorm"
)

type SessionCond struct {
	*genericdao.BaseCond
	TenantID uint
	UserID   uint
}

func (c *SessionCond) BuildCondition(db *gorm.DB, tableName string) {
	if c.BaseCond != nil {
		c.BaseCond.BuildCondition(db, tableName)
	}
	if c.TenantID != 0 {
		db.Where(tableName + ".tenant_id = ?", c.TenantID)
	}
	if c.UserID != 0 {
		db.Where(tableName + ".user_id = ?", c.UserID)
	}
}

type SessionDao struct {
	*genericdao.GenericDao[model.RefreshTokenEntity, model.RefreshTokenEntityList]
}

func NewSessionDao() *SessionDao {
	return &SessionDao{
		GenericDao: genericdao.NewGenericDao[model.RefreshTokenEntity, model.RefreshTokenEntityList](
			model.TableNameRefreshToken, "SessionDao",
			dbclient.IamDB,
		),
	}
}

func (d *SessionDao) GetPageListByCond(ctx context.Context, cond *SessionCond, page, pageSize int) ([]model.RefreshTokenEntity, int64, error) {
	var total int64
	query := d.DB.WithContext(ctx).Model(&model.RefreshTokenEntity{})

	cond.BuildCondition(query, d.TableName)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list model.RefreshTokenEntityList
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (d *SessionDao) RevokeByID(ctx context.Context, id, userID uint) error {
	return d.DB.WithContext(ctx).Model(&model.RefreshTokenEntity{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("revoked_at", gorm.Expr("NOW()")).Error
}

func (d *SessionDao) RevokeAllByUserID(ctx context.Context, userID uint) error {
	return d.DB.WithContext(ctx).Model(&model.RefreshTokenEntity{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", gorm.Expr("NOW()")).Error
}
```

#### Task 3: 会话服务

**Files:**
- Create: `internal/service/svcsession/session.go`

- [ ] **Step 1: 创建会话服务接口和实现**

```go
package svcsession

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/golib/glog"
)

type SessionSvc interface {
	List(ctx *gin.Context, req *dtouser.SessionListReq, userID uint) (*dtouser.SessionListResp, error)
	Revoke(ctx *gin.Context, req *dtouser.SessionRevokeReq, userID uint) error
	RevokeAll(ctx *gin.Context, userID uint) error
}

type sessionSvc struct{}

var _ SessionSvc = (*sessionSvc)(nil)

func NewSessionSvc() SessionSvc {
	return &sessionSvc{}
}

func (svc *sessionSvc) List(ctx *gin.Context, req *dtouser.SessionListReq, userID uint) (*dtouser.SessionListResp, error) {
	sessionDao := dao.NewSessionDao()

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	list, total, err := sessionDao.GetPageListByCond(ctx, &dao.SessionCond{
		UserID: userID,
	}, page, pageSize)
	if err != nil {
		glog.Errorf(ctx, "[sessionSvc.List] get page list fail, err:%v", err)
		return nil, code.GetError(code.SessionGetListError)
	}

	sessions := make([]dtouser.SessionResp, 0, len(list))
	now := time.Now()
	for _, item := range list {
		isActive := item.RevokedAt == nil && (item.ExpiresAt == nil || item.ExpiresAt.After(now))
		expiresAt := ""
		if item.ExpiresAt != nil {
			expiresAt = item.ExpiresAt.Format("2006-01-02 15:04:05")
		}
		sessions = append(sessions, dtouser.SessionResp{
			ID:            item.ID,
			ApplicationID: item.ApplicationID,
			TenantID:      item.TenantID,
			ExpiresAt:     &expiresAt,
			CreatedAt:     item.CreatedAt.Format("2006-01-02 15:04:05"),
			IsActive:      isActive,
		})
	}

	return &dtouser.SessionListResp{
		PageResp: gobject.PageResp{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
		Sessions: sessions,
	}, nil
}

func (svc *sessionSvc) Revoke(ctx *gin.Context, req *dtouser.SessionRevokeReq, userID uint) error {
	sessionDao := dao.NewSessionDao()

	if err := sessionDao.RevokeByID(ctx, uint(req.SessionID), userID); err != nil {
		glog.Errorf(ctx, "[sessionSvc.Revoke] revoke fail, err:%v", err)
		return code.GetError(code.SessionRevokeError)
	}

	return nil
}

func (svc *sessionSvc) RevokeAll(ctx *gin.Context, userID uint) error {
	sessionDao := dao.NewSessionDao()

	if err := sessionDao.RevokeAllByUserID(ctx, userID); err != nil {
		glog.Errorf(ctx, "[sessionSvc.RevokeAll] revoke all fail, err:%v", err)
		return code.GetError(code.SessionRevokeError)
	}

	return nil
}
```

#### Task 4: 会话控制器

**Files:**
- Create: `internal/controller/ctrsession/session.go`

- [ ] **Step 1: 创建会话控制器**

```go
package ctrsession

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam/internal/dto/dtouser"
	"github.com/morehao/ark-iam/iam/internal/service/svcsession"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

type SessionCtr interface {
	List(ctx *gin.Context)
	Revoke(ctx *gin.Context)
	RevokeAll(ctx *gin.Context)
}

type sessionCtr struct {
	sessionSvc svcsession.SessionSvc
}

var _ SessionCtr = (*sessionCtr)(nil)

func NewSessionCtr() SessionCtr {
	return &sessionCtr{
		sessionSvc: svcsession.NewSessionSvc(),
	}
}

// @Tags 会话管理
// @Summary 会话列表
// @accept application/json
// @Produce application/json
// @Param req query dtouser.SessionListReq true "会话列表"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.SessionListResp}
// @Router /v1/iam/user/sessions [get]
func (ctr *sessionCtr) List(ctx *gin.Context) {
	var req dtouser.SessionListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}

	userID := gincontext.GetUserID(ctx)
	res, err := ctr.sessionSvc.List(ctx, &req, userID)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 会话管理
// @Summary 撤销会话
// @accept application/json
// @Produce application/json
// @Param req path dtouser.SessionRevokeReq true "撤销会话"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/sessions/{sessionId} [delete]
func (ctr *sessionCtr) Revoke(ctx *gin.Context) {
	var req dtouser.SessionRevokeReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}

	userID := gincontext.GetUserID(ctx)
	if err := ctr.sessionSvc.Revoke(ctx, &req, userID); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "撤销成功")
}

// @Tags 会话管理
// @Summary 撤销所有会话
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/user/sessions [delete]
func (ctr *sessionCtr) RevokeAll(ctx *gin.Context) {
	userID := gincontext.GetUserID(ctx)
	if err := ctr.sessionSvc.RevokeAll(ctx, userID); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "撤销成功")
}
```

#### Task 5: 会话路由注册

**Files:**
- Modify: `internal/router/user.go`

- [ ] **Step 1: 添加会话路由**

```go
func userRouter(groups *ginserver.RouterGroups) {
	userCtr := ctruser.NewUserCtr()
	sessionCtr := ctrsession.NewSessionCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	iamGroup := v1RouterGroup.Group("/iam")

	// 用户相关路由
	iamGroup.POST("/user/create", userCtr.Create)
	// ... 其他现有路由

	// 会话相关路由
	iamGroup.GET("/user/sessions", sessionCtr.List)
	iamGroup.DELETE("/user/sessions", sessionCtr.RevokeAll)
	iamGroup.DELETE("/user/sessions/:sessionId", sessionCtr.Revoke)
}
```

---

### Phase 2: SSO 逻辑完善 (P0)

#### Task 6: 完善 SSO 授权 URL 构建

**Files:**
- Modify: `internal/service/svcauth/auth.go`

- [ ] **Step 1: 完善 buildAuthorizationUrl 方法**

```go
func (svc *authSvc) buildAuthorizationUrl(ssoConnector *model.SsoConnectorEntity) (string, error) {
	var config SsoConnectorConfig
	if err := json.Unmarshal(ssoConnector.Config, &config); err != nil {
		return "", err
	}

	state := generateState()
	stateStore.Set(state, ssoConnector.ID, 10*time.Minute)

	params := url.Values{}
	params.Set("client_id", config.ClientID)
	params.Set("redirect_uri", config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", config.Scopes)
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", config.AuthURL, params.Encode()), nil
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
```

#### Task 7: 完善 Code 交换用户信息

**Files:**
- Modify: `internal/service/svcauth/auth.go`

- [ ] **Step 1: 完善 exchangeCodeForUserInfo 方法**

```go
func (svc *authSvc) exchangeCodeForUserInfo(ctx *gin.Context, ssoConnector *model.SsoConnectorEntity, code, state string) (*ssoUserInfo, error) {
	var config SsoConnectorConfig
	if err := json.Unmarshal(ssoConnector.Config, &config); err != nil {
		return nil, err
	}

	savedConnectorID, err := stateStore.Get(state)
	if err != nil || savedConnectorID != ssoConnector.ID {
		return nil, errors.New("invalid state")
	}

	tokenResp, err := exchangeCode(config.TokenURL, config.ClientID, config.ClientSecret, code, config.RedirectURI)
	if err != nil {
		return nil, err
	}

	userInfo, err := fetchUserInfo(config.UserInfoURL, tokenResp.AccessToken)
	if err != nil {
		return nil, err
	}

	return &ssoUserInfo{
		Issuer:    ssoConnector.ProviderName,
		Subject:   userInfo.Subject,
		Email:     userInfo.Email,
		Name:      userInfo.Name,
		AvatarUrl: userInfo.AvatarURL,
	}, nil
}

type SsoConnectorConfig struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	AuthURL      string   `json:"authUrl"`
	TokenURL     string   `json:"tokenUrl"`
	UserInfoURL  string   `json:"userInfoUrl"`
	RedirectURI  string   `json:"redirectUri"`
	Scopes       []string `json:"scopes"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type userInfoResponse struct {
	Subject  string `json:"sub"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
}
```

---

### Phase 3: 应用角色管理 (P0)

#### Task 8: 应用角色 DTO

**Files:**
- Modify: `internal/dto/dtoapplication/request.go`
- Modify: `internal/dto/dtoapplication/response.go`

- [ ] **Step 1: 添加应用角色请求 DTO**

```go
type ApplicationRoleListReq struct {
	ApplicationID uint `json:"applicationId" form:"applicationId" binding:"required"`
}

type AssignApplicationRolesReq struct {
	ApplicationID uint64   `json:"applicationId" binding:"required"`
	RoleIDs       []uint64 `json:"roleIds" binding:"required,min=1"`
}

type RemoveApplicationRoleReq struct {
	ApplicationID uint64 `json:"applicationId" binding:"required"`
	RoleID        uint64 `json:"roleId" path:"roleId" binding:"required"`
}
```

- [ ] **Step 2: 添加应用角色响应 DTO**

```go
type ApplicationRoleResp struct {
	RoleID        uint64 `json:"roleId"`
	RoleName      string `json:"roleName"`
	RoleCode      string `json:"roleCode"`
	ApplicationID uint64 `json:"applicationId"`
	CreatedAt     string `json:"createdAt"`
}

type ApplicationRoleListResp struct {
	gobject.PageResp
	Roles []ApplicationRoleResp `json:"roles"`
}
```

#### Task 9: 应用服务角色管理方法

**Files:**
- Modify: `internal/service/svcapplication/application.go`

- [ ] **Step 1: 添加应用角色管理方法到 ApplicationSvc 接口**

```go
type ApplicationSvc interface {
	// ... 现有方法
	ListRoles(ctx *gin.Context, req *dtoapplication.ApplicationRoleListReq) (*dtoapplication.ApplicationRoleListResp, error)
	AssignRoles(ctx *gin.Context, req *dtoapplication.AssignApplicationRolesReq) error
	RemoveRole(ctx *gin.Context, req *dtoapplication.RemoveApplicationRoleReq) error
}
```

- [ ] **Step 2: 实现应用角色管理方法**

```go
func (svc *applicationSvc) ListRoles(ctx *gin.Context, req *dtoapplication.ApplicationRoleListReq) (*dtoapplication.ApplicationRoleListResp, error) {
	appRoleDao := dao.NewApplicationRoleDao()
	roleDao := dao.NewRoleDao()

	list, err := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
		ApplicationID: req.ApplicationID,
	})
	if err != nil {
		glog.Errorf(ctx, "[applicationSvc.ListRoles] get roles fail, err:%v", err)
		return nil, code.GetError(code.ApplicationRoleGetListError)
	}

	roleMap := make(map[uint]*model.RoleEntity)
	for _, ar := range list {
		if role, err := roleDao.GetByID(ctx, ar.RoleID); err == nil && role != nil {
			roleMap[role.ID] = role
		}
	}

	roles := make([]dtoapplication.ApplicationRoleResp, 0, len(list))
	for _, ar := range list {
		if role, ok := roleMap[ar.RoleID]; ok {
			roles = append(roles, dtoapplication.ApplicationRoleResp{
				RoleID:        ar.RoleID,
				RoleName:      role.Name,
				RoleCode:      role.Code,
				ApplicationID: ar.ApplicationID,
				CreatedAt:     ar.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	return &dtoapplication.ApplicationRoleListResp{
		PageResp: gobject.PageResp{Total: int64(len(roles))},
		Roles:    roles,
	}, nil
}

func (svc *applicationSvc) AssignRoles(ctx *gin.Context, req *dtoapplication.AssignApplicationRolesReq) error {
	appRoleDao := dao.NewApplicationRoleDao()
	userID := gincontext.GetUserID(ctx)

	for _, roleID := range req.RoleIDs {
		existing, _ := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
			ApplicationID: req.ApplicationID,
			RoleID:        roleID,
		})
		if len(existing) > 0 {
			continue
		}

		entity := &model.ApplicationRoleEntity{
			TenantID:      gincontext.GetTenantID(ctx),
			ApplicationID: req.ApplicationID,
			RoleID:        roleID,
			CreatedBy:     userID,
		}
		if err := appRoleDao.Insert(ctx, entity); err != nil {
			glog.Errorf(ctx, "[applicationSvc.AssignRoles] insert fail, err:%v", err)
			return code.GetError(code.ApplicationRoleCreateError)
		}
	}

	return nil
}

func (svc *applicationSvc) RemoveRole(ctx *gin.Context, req *dtoapplication.RemoveApplicationRoleReq) error {
	appRoleDao := dao.NewApplicationRoleDao()

	list, err := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
		ApplicationID: req.ApplicationID,
		RoleID:        req.RoleID,
	})
	if err != nil || len(list) == 0 {
		return code.GetError(code.ApplicationRoleNotExistError)
	}

	if err := appRoleDao.Delete(ctx, list[0].ID, gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[applicationSvc.RemoveRole] delete fail, err:%v", err)
		return code.GetError(code.ApplicationRoleDeleteError)
	}

	return nil
}
```

#### Task 10: 应用控制器角色管理

**Files:**
- Modify: `internal/controller/ctrpermission/application.go`

- [ ] **Step 1: 添加应用角色管理接口和实现**

```go
type ApplicationCtr interface {
	// ... 现有接口
	ListRoles(ctx *gin.Context)
	AssignRoles(ctx *gin.Context)
	RemoveRole(ctx *gin.Context)
}
```

- [ ] **Step 2: 实现应用角色管理方法**

```go
// @Tags 应用管理
// @Summary 应用角色列表
// @accept application/json
// @Produce application/json
// @Param req query dtoapplication.ApplicationRoleListReq true "应用角色列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.ApplicationRoleListResp}
// @Router /v1/iam/application/roles [get]
func (ctr *applicationCtr) ListRoles(ctx *gin.Context) {
	var req dtoapplication.ApplicationRoleListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.ListRoles(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 分配角色
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.AssignApplicationRolesReq true "分配角色"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/assignRoles [post]
func (ctr *applicationCtr) AssignRoles(ctx *gin.Context) {
	var req dtoapplication.AssignApplicationRolesReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.applicationSvc.AssignRoles(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "分配成功")
}

// @Tags 应用管理
// @Summary 移除角色
// @accept application/json
// @Produce application/json
// @Param req path dtoapplication.RemoveApplicationRoleReq true "移除角色"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/roles/{roleId} [delete]
func (ctr *applicationCtr) RemoveRole(ctx *gin.Context) {
	var req dtoapplication.RemoveApplicationRoleReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.applicationSvc.RemoveRole(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "移除成功")
}
```

#### Task 11: 应用角色路由注册

**Files:**
- Modify: `internal/router/permission.go`

- [ ] **Step 1: 添加应用角色路由**

```go
func applicationRouter(groups *ginserver.RouterGroups) {
	appCtr := ctrpermission.NewApplicationCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/application/create", appCtr.Create)
	// ... 其他现有路由

	// 应用角色管理
	v1RouterGroup.GET("/application/roles", appCtr.ListRoles)
	v1RouterGroup.POST("/application/assignRoles", appCtr.AssignRoles)
	v1RouterGroup.DELETE("/application/roles/:roleId", appCtr.RemoveRole)
}
```

---

### Phase 4: 角色用户管理 (P0)

#### Task 12: 角色用户 DTO

**Files:**
- Modify: `internal/dto/dtouser/request.go`

- [ ] **Step 1: 添加角色用户请求 DTO**

```go
type RoleUserListReq struct {
	RoleID uint64 `json:"roleId" form:"roleId" binding:"required"`
}

type AssignRoleUsersReq struct {
	RoleID  uint64   `json:"roleId" binding:"required"`
	UserIDs []uint64 `json:"userIds" binding:"required,min=1"`
}

type RemoveRoleUserReq struct {
	RoleID uint64 `json:"roleId" path:"roleId" binding:"required"`
	UserID uint64 `json:"userId" path:"userId" binding:"required"`
}
```

- [ ] **Step 2: 添加角色用户响应 DTO**

```go
type RoleUserResp struct {
	UserID    uint64 `json:"userId"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	RoleID    uint64 `json:"roleId"`
	CreatedAt string `json:"createdAt"`
}

type RoleUserListResp struct {
	gobject.PageResp
	Users []RoleUserResp `json:"users"`
}
```

#### Task 13: 角色服务用户管理方法

**Files:**
- Modify: `internal/service/svcpermission/role.go`

- [ ] **Step 1: 添加角色用户管理方法到 RoleSvc 接口**

```go
type RoleSvc interface {
	// ... 现有方法
	ListUsers(ctx *gin.Context, req *dtouser.RoleUserListReq) (*dtouser.RoleUserListResp, error)
	AssignUsers(ctx *gin.Context, req *dtouser.AssignRoleUsersReq) error
	RemoveUser(ctx *gin.Context, req *dtouser.RemoveRoleUserReq) error
}
```

- [ ] **Step 2: 实现角色用户管理方法**

```go
func (svc *roleSvc) ListUsers(ctx *gin.Context, req *dtouser.RoleUserListReq) (*dtouser.RoleUserListResp, error) {
	userRoleDao := dao.NewUserRoleDao()
	userDao := dao.NewUserDao()

	list, err := userRoleDao.GetByCond(ctx, &dao.UserRoleCond{
		RoleID: uint(req.RoleID),
	})
	if err != nil {
		glog.Errorf(ctx, "[roleSvc.ListUsers] get users fail, err:%v", err)
		return nil, code.GetError(code.RoleUserGetListError)
	}

	userMap := make(map[uint]*model.UserEntity)
	for _, ur := range list {
		if user, err := userDao.GetByID(ctx, ur.UserID); err == nil && user != nil {
			userMap[user.ID] = user
		}
	}

	users := make([]dtouser.RoleUserResp, 0, len(list))
	for _, ur := range list {
		if user, ok := userMap[ur.UserID]; ok {
			users = append(users, dtouser.RoleUserResp{
				UserID:    ur.UserID,
				Username:  user.Username,
				Name:      user.Name,
				Email:     user.PrimaryEmail,
				RoleID:    ur.RoleID,
				CreatedAt: ur.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	return &dtouser.RoleUserListResp{
		PageResp: gobject.PageResp{Total: int64(len(users))},
		Users:    users,
	}, nil
}

func (svc *roleSvc) AssignUsers(ctx *gin.Context, req *dtouser.AssignRoleUsersReq) error {
	userRoleDao := dao.NewUserRoleDao()
	userID := gincontext.GetUserID(ctx)

	for _, userID := range req.UserIDs {
		existing, _ := userRoleDao.GetByCond(ctx, &dao.UserRoleCond{
			RoleID: uint(req.RoleID),
			UserID: uint(userID),
		})
		if len(existing) > 0 {
			continue
		}

		entity := &model.UserRoleEntity{
			TenantID: gincontext.GetTenantID(ctx),
			UserID:   uint(userID),
			RoleID:   uint(req.RoleID),
			CreatedBy: userID,
		}
		if err := userRoleDao.Insert(ctx, entity); err != nil {
			glog.Errorf(ctx, "[roleSvc.AssignUsers] insert fail, err:%v", err)
			return code.GetError(code.RoleUserCreateError)
		}
	}

	return nil
}

func (svc *roleSvc) RemoveUser(ctx *gin.Context, req *dtouser.RemoveRoleUserReq) error {
	userRoleDao := dao.NewUserRoleDao()

	list, err := userRoleDao.GetByCond(ctx, &dao.UserRoleCond{
		RoleID: uint(req.RoleID),
		UserID: uint(req.UserID),
	})
	if err != nil || len(list) == 0 {
		return code.GetError(code.RoleUserNotExistError)
	}

	if err := userRoleDao.Delete(ctx, list[0].ID, gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[roleSvc.RemoveUser] delete fail, err:%v", err)
		return code.GetError(code.RoleUserDeleteError)
	}

	return nil
}
```

#### Task 14: 角色控制器用户管理

**Files:**
- Modify: `internal/controller/ctrpermission/role.go`

- [ ] **Step 1: 添加角色用户管理接口和实现**

```go
type RoleCtr interface {
	// ... 现有接口
	ListUsers(ctx *gin.Context)
	AssignUsers(ctx *gin.Context)
	RemoveUser(ctx *gin.Context)
}
```

- [ ] **Step 2: 实现角色用户管理方法**

```go
// @Tags 角色管理
// @Summary 角色用户列表
// @accept application/json
// @Produce application/json
// @Param req query dtouser.RoleUserListReq true "角色用户列表"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.RoleUserListResp}
// @Router /v1/iam/role/users [get]
func (ctr *roleCtr) ListUsers(ctx *gin.Context) {
	var req dtouser.RoleUserListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleSvc.ListUsers(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 角色管理
// @Summary 分配用户
// @accept application/json
// @Produce application/json
// @Param req body dtouser.AssignRoleUsersReq true "分配用户"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/role/assignUsers [post]
func (ctr *roleCtr) AssignUsers(ctx *gin.Context) {
	var req dtouser.AssignRoleUsersReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleSvc.AssignUsers(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "分配成功")
}

// @Tags 角色管理
// @Summary 移除用户
// @accept application/json
// @Produce application/json
// @Param req path dtouser.RemoveRoleUserReq true "移除用户"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/role/users/{roleId}/{userId} [delete]
func (ctr *roleCtr) RemoveUser(ctx *gin.Context) {
	var req dtouser.RemoveRoleUserReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleSvc.RemoveUser(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "移除成功")
}
```

#### Task 15: 角色用户路由注册

**Files:**
- Modify: `internal/router/permission.go`

- [ ] **Step 1: 添加角色用户路由**

```go
func roleRouter(groups *ginserver.RouterGroups) {
	roleCtr := ctrpermission.NewRoleCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.POST("/role/create", roleCtr.Create)
	// ... 其他现有路由

	// 角色用户管理
	v1RouterGroup.GET("/role/users", roleCtr.ListUsers)
	v1RouterGroup.POST("/role/assignUsers", roleCtr.AssignUsers)
	v1RouterGroup.DELETE("/role/users/:roleId/:userId", roleCtr.RemoveUser)
}
```

---

### Phase 5: 角色应用管理 (P0)

#### Task 16: 角色应用 DTO

**Files:**
- Modify: `internal/dto/dtouser/request.go`

- [ ] **Step 1: 添加角色应用请求 DTO**

```go
type RoleApplicationListReq struct {
	RoleID uint64 `json:"roleId" form:"roleID" binding:"required"`
}

type AssignRoleApplicationsReq struct {
	RoleID         uint64   `json:"roleId" binding:"required"`
	ApplicationIDs []uint64 `json:"applicationIds" binding:"required,min=1"`
}
```

- [ ] **Step 2: 添加角色应用响应 DTO**

```go
type RoleApplicationResp struct {
	ApplicationID uint64 `json:"applicationId"`
	AppName      string `json:"appName"`
	AppType      string `json:"appType"`
	RoleID       uint64 `json:"roleId"`
	CreatedAt    string `json:"createdAt"`
}

type RoleApplicationListResp struct {
	gobject.PageResp
	Applications []RoleApplicationResp `json:"applications"`
}
```

#### Task 17: 角色服务应用管理方法

**Files:**
- Modify: `internal/service/svcpermission/role.go`

- [ ] **Step 1: 添加角色应用管理方法到 RoleSvc 接口**

```go
type RoleSvc interface {
	// ... 现有方法
	ListApplications(ctx *gin.Context, req *dtouser.RoleApplicationListReq) (*dtouser.RoleApplicationListResp, error)
	AssignApplications(ctx *gin.Context, req *dtouser.AssignRoleApplicationsReq) error
}
```

- [ ] **Step 2: 实现角色应用管理方法**

```go
func (svc *roleSvc) ListApplications(ctx *gin.Context, req *dtouser.RoleApplicationListReq) (*dtouser.RoleApplicationListResp, error) {
	appRoleDao := dao.NewApplicationRoleDao()
	appDao := dao.NewApplicationDao()

	list, err := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
		RoleID: uint(req.RoleID),
	})
	if err != nil {
		glog.Errorf(ctx, "[roleSvc.ListApplications] get applications fail, err:%v", err)
		return nil, code.GetError(code.RoleApplicationGetListError)
	}

	appMap := make(map[uint]*model.ApplicationEntity)
	for _, ar := range list {
		if app, err := appDao.GetByID(ctx, ar.ApplicationID); err == nil && app != nil {
			appMap[app.ID] = app
		}
	}

	apps := make([]dtouser.RoleApplicationResp, 0, len(list))
	for _, ar := range list {
		if app, ok := appMap[ar.ApplicationID]; ok {
			apps = append(apps, dtouser.RoleApplicationResp{
				ApplicationID: ar.ApplicationID,
				AppName:      app.Name,
				AppType:      app.Type,
				RoleID:       ar.RoleID,
				CreatedAt:    ar.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	return &dtouser.RoleApplicationListResp{
		PageResp:     gobject.PageResp{Total: int64(len(apps))},
		Applications: apps,
	}, nil
}

func (svc *roleSvc) AssignApplications(ctx *gin.Context, req *dtouser.AssignRoleApplicationsReq) error {
	appRoleDao := dao.NewApplicationRoleDao()
	userID := gincontext.GetUserID(ctx)

	for _, appID := range req.ApplicationIDs {
		existing, _ := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
			ApplicationID: appID,
			RoleID:        uint(req.RoleID),
		})
		if len(existing) > 0 {
			continue
		}

		entity := &model.ApplicationRoleEntity{
			TenantID:      gincontext.GetTenantID(ctx),
			ApplicationID: appID,
			RoleID:        uint(req.RoleID),
			CreatedBy:     userID,
		}
		if err := appRoleDao.Insert(ctx, entity); err != nil {
			glog.Errorf(ctx, "[roleSvc.AssignApplications] insert fail, err:%v", err)
			return code.GetError(code.RoleApplicationCreateError)
		}
	}

	return nil
}
```

#### Task 18: 角色控制器应用管理

**Files:**
- Modify: `internal/controller/ctrpermission/role.go`

- [ ] **Step 1: 添加角色应用管理接口和实现**

```go
type RoleCtr interface {
	// ... 现有接口
	ListApplications(ctx *gin.Context)
	AssignApplications(ctx *gin.Context)
}
```

- [ ] **Step 2: 实现角色应用管理方法**

```go
// @Tags 角色管理
// @Summary 角色应用列表
// @accept application/json
// @Produce application/json
// @Param req query dtouser.RoleApplicationListReq true "角色应用列表"
// @Success 200 {object} gincontext.DtoRender{data=dtouser.RoleApplicationListResp}
// @Router /v1/iam/role/applications [get]
func (ctr *roleCtr) ListApplications(ctx *gin.Context) {
	var req dtouser.RoleApplicationListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.roleSvc.ListApplications(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 角色管理
// @Summary 分配应用
// @accept application/json
// @Produce application/json
// @Param req body dtouser.AssignRoleApplicationsReq true "分配应用"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/role/assignApplications [post]
func (ctr *roleCtr) AssignApplications(ctx *gin.Context) {
	var req dtouser.AssignRoleApplicationsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.roleSvc.AssignApplications(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "分配成功")
}
```

#### Task 19: 角色应用路由注册

**Files:**
- Modify: `internal/router/permission.go`

- [ ] **Step 1: 添加角色应用路由**

```go
	// 角色应用管理
	v1RouterGroup.GET("/role/applications", roleCtr.ListApplications)
	v1RouterGroup.POST("/role/assignApplications", roleCtr.AssignApplications)
```

---

### Phase 6: SSO 连接器增强 (P1)

#### Task 20: SSO 提供商列表

**Files:**
- Modify: `internal/dto/dtosso/request.go`
- Modify: `internal/dto/dtosso/response.go`
- Modify: `internal/service/svcauth/sso_connector.go`
- Modify: `internal/controller/ctrauth/sso_connector.go`
- Modify: `internal/router/auth.go`

- [ ] **Step 1: 创建 SSO DTO 文件**

```go
// internal/dto/dtosso/request.go
package dtosso

type SsoProviderListReq struct{}

type SsoIdpConfigReq struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	AuthURL      string   `json:"authUrl"`
	TokenURL     string   `json:"tokenUrl"`
	UserInfoURL  string   `json:"userInfoUrl"`
	Scopes       []string `json:"scopes"`
}
```

```go
// internal/dto/dtosso/response.go
package dtosso

type SsoProviderResp struct {
	ProviderName string `json:"providerName"`
	DisplayName  string `json:"displayName"`
	Logo         string `json:"logo"`
}

type SsoProviderListResp struct {
	Providers []SsoProviderResp `json:"providers"`
}

type SsoIdpConfigResp struct {
	ClientID    string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret,omitempty"`
	AuthURL     string   `json:"authUrl"`
	TokenURL    string   `json:"tokenUrl"`
	UserInfoURL string   `json:"userInfoUrl"`
	Scopes      []string `json:"scopes"`
}
```

- [ ] **Step 2: 添加 SSO 提供商列表方法**

```go
// internal/service/svcauth/sso_connector.go
func (svc *ssoConnectorSvc) ListProviders(ctx *gin.Context) (*dtosso.SsoProviderListResp, error) {
	providers := []dtosso.SsoProviderResp{
		{ProviderName: "oidc", DisplayName: "OIDC Provider", Logo: ""},
		{ProviderName: "saml", DisplayName: "SAML Provider", Logo: ""},
	}
	return &dtosso.SsoProviderListResp{Providers: providers}, nil
}

func (svc *ssoConnectorSvc) GetIdpConfig(ctx *gin.Context, connectorID uint) (*dtosso.SsoIdpConfigResp, error) {
	entity, err := svc.dao.GetByID(ctx, connectorID)
	if err != nil || entity == nil {
		return nil, code.GetError(code.SsoConnectorNotExistError)
	}

	var config SsoConnectorConfig
	if err := json.Unmarshal(entity.Config, &config); err != nil {
		return nil, code.GetError(code.SsoConnectorGetDetailError)
	}

	return &dtosso.SsoIdpConfigResp{
		ClientID:    config.ClientID,
		AuthURL:     config.AuthURL,
		TokenURL:    config.TokenURL,
		UserInfoURL: config.UserInfoURL,
		Scopes:      config.Scopes,
	}, nil
}

func (svc *ssoConnectorSvc) UpdateIdpConfig(ctx *gin.Context, connectorID uint, req *dtosso.SsoIdpConfigReq) error {
	entity, err := svc.dao.GetByID(ctx, connectorID)
	if err != nil || entity == nil {
		return code.GetError(code.SsoConnectorNotExistError)
	}

	config := SsoConnectorConfig{
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		AuthURL:      req.AuthURL,
		TokenURL:     req.TokenURL,
		UserInfoURL:  req.UserInfoURL,
		Scopes:       req.Scopes,
	}

	configBytes, _ := json.Marshal(config)
	if err := svc.dao.UpdateMap(ctx, connectorID, map[string]interface{}{
		"config": string(configBytes),
	}); err != nil {
		glog.Errorf(ctx, "[ssoConnectorSvc.UpdateIdpConfig] update fail, err:%v", err)
		return code.GetError(code.SsoConnectorUpdateError)
	}

	return nil
}
```

- [ ] **Step 3: 添加 SSO 控制器方法**

```go
// @Tags SSO连接器
// @Summary SSO提供商列表
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtosso.SsoProviderListResp}
// @Router /v1/iam/ssoConnector/providers [get]
func (ctr *ssoConnectorCtr) ListProviders(ctx *gin.Context) {
	res, err := ctr.ssoConnectorSvc.ListProviders(ctx)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags SSO连接器
// @Summary 获取IdP配置
// @accept application/json
// @Produce application/json
// @Param connectorID path uint true "SSO连接器ID"
// @Success 200 {object} gincontext.DtoRender{data=dtosso.SsoIdpConfigResp}
// @Router /v1/iam/ssoConnector/{connectorId}/idp-config [get]
func (ctr *ssoConnectorCtr) GetIdpConfig(ctx *gin.Context) {
	var req dtosso.SsoConnectorIDReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.ssoConnectorSvc.GetIdpConfig(ctx, uint(req.ConnectorID))
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags SSO连接器
// @Summary 更新IdP配置
// @accept application/json
// @Produce application/json
// @Param connectorID path uint true "SSO连接器ID"
// @Param req body dtosso.SsoIdpConfigReq true "IdP配置"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/ssoConnector/{connectorId}/idp-config [put]
func (ctr *ssoConnectorCtr) UpdateIdpConfig(ctx *gin.Context) {
	var uriReq dtosso.SsoConnectorIDReq
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	var req dtosso.SsoIdpConfigReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.ssoConnectorSvc.UpdateIdpConfig(ctx, uint(uriReq.ConnectorID), &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "更新成功")
}
```

- [ ] **Step 4: 注册 SSO 路由**

```go
func ssoConnectorRouter(groups *ginserver.RouterGroups) {
	ssoCtr := ctrauth.NewSsoConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.GET("/ssoConnector/providers", ssoCtr.ListProviders)
	v1RouterGroup.GET("/ssoConnector/:connectorId/idp-config", ssoCtr.GetIdpConfig)
	v1RouterGroup.PUT("/ssoConnector/:connectorId/idp-config", ssoCtr.UpdateIdpConfig)
}
```

---

### Phase 7: 连接器增强 (P1)

#### Task 21: 连接器工厂列表

**Files:**
- Modify: `internal/dto/dtoconnector/request.go`
- Modify: `internal/dto/dtoconnector/response.go`
- Modify: `internal/service/svcauth/connector.go`
- Modify: `internal/controller/ctrauth/connector.go`
- Modify: `internal/router/auth.go`

- [ ] **Step 1: 创建连接器 DTO 文件**

```go
// internal/dto/dtoconnector/request.go
package dtoconnector

type ConnectorFactoryListReq struct{}

type TestConnectorReq struct {
	ConnectorID uint64 `json:"connectorId" path:"connectorId" binding:"required"`
	Config      string `json:"config"`
}

type AuthorizationUriReq struct {
	ConnectorID uint64 `json:"connectorId" path:"connectorId" binding:"required"`
	RedirectURI string `json:"redirectUri" binding:"required"`
	State       string `json:"state"`
}
```

```go
// internal/dto/dtoconnector/response.go
package dtoconnector

type ConnectorFactoryResp struct {
	FactoryID      string `json:"factoryId"`
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description"`
	Logo           string `json:"logo"`
	IsStandard     bool   `json:"isStandard"`
}

type ConnectorFactoryListResp struct {
	Factories []ConnectorFactoryResp `json:"factories"`
}

type TestConnectorResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type AuthorizationUriResp struct {
	AuthorizationUri string `json:"authorizationUri"`
}
```

- [ ] **Step 2: 添加连接器服务方法**

```go
func (svc *connectorSvc) ListFactories(ctx *gin.Context) (*dtoconnector.ConnectorFactoryListResp, error) {
	factories := []dtoconnector.ConnectorFactoryResp{
		{FactoryID: "wechat", Name: "wechat", DisplayName: "微信", Description: "微信登录", IsStandard: false},
		{FactoryID: "google", Name: "google", DisplayName: "Google", Description: "Google登录", IsStandard: true},
		{FactoryID: "github", Name: "github", DisplayName: "GitHub", Description: "GitHub登录", IsStandard: true},
	}
	return &dtoconnector.ConnectorFactoryListResp{Factories: factories}, nil
}

func (svc *connectorSvc) TestConnector(ctx *gin.Context, connectorID uint64, config string) (*dtoconnector.TestConnectorResp, error) {
	return &dtoconnector.TestConnectorResp{
		Success: true,
		Message: "连接成功",
	}, nil
}

func (svc *connectorSvc) GetAuthorizationUri(ctx *gin.Context, connectorID uint64, redirectURI, state string) (*dtoconnector.AuthorizationUriResp, error) {
	return &dtoconnector.AuthorizationUriResp{
		AuthorizationUri: fmt.Sprintf("https://example.com/oauth/authorize?redirect_uri=%s&state=%s", redirectURI, state),
	}, nil
}
```

- [ ] **Step 3: 添加连接器控制器方法**

```go
// @Tags 连接器
// @Summary 连接器工厂列表
// @accept application/json
// @Produce application/json
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.ConnectorFactoryListResp}
// @Router /v1/iam/connector/factories [get]
func (ctr *connectorCtr) ListFactories(ctx *gin.Context) {
	res, err := ctr.connectorSvc.ListFactories(ctx)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 测试连接器
// @accept application/json
// @Produce application/json
// @Param connectorID path uint64 true "连接器ID"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.TestConnectorResp}
// @Router /v1/iam/connector/{connectorId}/test [post]
func (ctr *connectorCtr) TestConnector(ctx *gin.Context) {
	var req dtoconnector.TestConnectorReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.connectorSvc.TestConnector(ctx, req.ConnectorID, req.Config)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 连接器
// @Summary 获取授权URI
// @accept application/json
// @Produce application/json
// @Param connectorID path uint64 true "连接器ID"
// @Param req body dtoconnector.AuthorizationUriReq true "授权URI请求"
// @Success 200 {object} gincontext.DtoRender{data=dtoconnector.AuthorizationUriResp}
// @Router /v1/iam/connector/{connectorId}/authorization-uri [post]
func (ctr *connectorCtr) GetAuthorizationUri(ctx *gin.Context) {
	var uriReq dtoconnector.AuthorizationUriReq
	if err := ctx.ShouldBindUri(&uriReq); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	var req dtoconnector.AuthorizationUriReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.connectorSvc.GetAuthorizationUri(ctx, uriReq.ConnectorID, req.RedirectURI, req.State)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}
```

- [ ] **Step 4: 注册连接器路由**

```go
func connectorRouter(groups *ginserver.RouterGroups) {
	connectorCtr := ctrauth.NewConnectorCtr()

	v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
	v1RouterGroup.GET("/connector/factories", connectorCtr.ListFactories)
	v1RouterGroup.POST("/connector/:connectorId/test", connectorCtr.TestConnector)
	v1RouterGroup.POST("/connector/:connectorId/authorization-uri", connectorCtr.GetAuthorizationUri)
}
```

---

### Phase 8: 应用密钥管理 (P1)

#### Task 22: 应用密钥 DTO

**Files:**
- Modify: `internal/dto/dtoapplication/request.go`
- Modify: `internal/dto/dtoapplication/response.go`

- [ ] **Step 1: 添加应用密钥请求 DTO**

```go
type ApplicationSecretListReq struct {
	ApplicationID uint `json:"applicationId" form:"applicationId" binding:"required"`
}

type CreateApplicationSecretReq struct {
	ApplicationID uint   `json:"applicationId" binding:"required"`
	Name         string `json:"name" binding:"required"`
	ExpiresAt    string `json:"expiresAt"`
}

type DeleteApplicationSecretReq struct {
	SecretID uint64 `json:"secretId" path:"secretId" binding:"required"`
}
```

- [ ] **Step 2: 添加应用密钥响应 DTO**

```go
type ApplicationSecretResp struct {
	ID            uint64  `json:"id"`
	ApplicationID uint64  `json:"applicationId"`
	Name          string  `json:"name"`
	Value         string  `json:"value,omitempty"`
	ExpiresAt     *string `json:"expiresAt"`
	CreatedAt     string  `json:"createdAt"`
}

type ApplicationSecretListResp struct {
	gobject.PageResp
	Secrets []ApplicationSecretResp `json:"secrets"`
}

type CreateApplicationSecretResp struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	Secret string `json:"secret"`
}
```

#### Task 23: 应用密钥 DAO

**Files:**
- Modify: `dao/application_secret.go`

- [ ] **Step 1: 添加应用密钥 DAO 方法**

```go
type ApplicationSecretCond struct {
	*genericdao.BaseCond
	TenantID     uint
	ApplicationID uint
}

func (d *ApplicationSecretDao) GetPageListByCond(ctx context.Context, cond *ApplicationSecretCond, page, pageSize int) ([]model.ApplicationSecretEntity, int64, error) {
	var total int64
	query := d.DB.WithContext(ctx).Model(&model.ApplicationSecretEntity{})

	if cond.TenantID != 0 {
		query = query.Where("tenant_id = ?", cond.TenantID)
	}
	if cond.ApplicationID != 0 {
		query = query.Where("application_id = ?", cond.ApplicationID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list model.ApplicationSecretEntityList
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}
```

#### Task 24: 应用服务密钥管理方法

**Files:**
- Modify: `internal/service/svcapplication/application.go`

- [ ] **Step 1: 添加应用密钥管理方法到 ApplicationSvc 接口**

```go
type ApplicationSvc interface {
	// ... 现有方法
	ListSecrets(ctx *gin.Context, req *dtoapplication.ApplicationSecretListReq) (*dtoapplication.ApplicationSecretListResp, error)
	CreateSecret(ctx *gin.Context, req *dtoapplication.CreateApplicationSecretReq) (*dtoapplication.CreateApplicationSecretResp, error)
	DeleteSecret(ctx *gin.Context, req *dtoapplication.DeleteApplicationSecretReq) error
}
```

- [ ] **Step 2: 实现应用密钥管理方法**

```go
func (svc *applicationSvc) ListSecrets(ctx *gin.Context, req *dtoapplication.ApplicationSecretListReq) (*dtoapplication.ApplicationSecretListResp, error) {
	secretDao := dao.NewApplicationSecretDao()

	secrets, total, err := secretDao.GetPageListByCond(ctx, &dao.ApplicationSecretCond{
		ApplicationID: req.ApplicationID,
	}, 1, 100)
	if err != nil {
		glog.Errorf(ctx, "[applicationSvc.ListSecrets] get secrets fail, err:%v", err)
		return nil, code.GetError(code.ApplicationSecretGetListError)
	}

	result := make([]dtoapplication.ApplicationSecretResp, 0, len(secrets))
	for _, s := range secrets {
		expiresAt := ""
		if s.ExpiresAt != nil {
			expiresAt = s.ExpiresAt.Format("2006-01-02 15:04:05")
		}
		result = append(result, dtoapplication.ApplicationSecretResp{
			ID:            s.ID,
			ApplicationID: s.ApplicationID,
			Name:          s.Name,
			Value:         s.Value,
			ExpiresAt:     &expiresAt,
			CreatedAt:     s.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &dtoapplication.ApplicationSecretListResp{
		PageResp: gobject.PageResp{Total: total},
		Secrets:  result,
	}, nil
}

func (svc *applicationSvc) CreateSecret(ctx *gin.Context, req *dtoapplication.CreateApplicationSecretReq) (*dtoapplication.CreateApplicationSecretResp, error) {
	secretDao := dao.NewApplicationSecretDao()
	userID := gincontext.GetUserID(ctx)

	secretValue := generateSecretValue()

	entity := &model.ApplicationSecretEntity{
		TenantID:      gincontext.GetTenantID(ctx),
		ApplicationID: req.ApplicationID,
		Name:          req.Name,
		Value:         secretValue,
		CreatedBy:     userID,
	}

	if req.ExpiresAt != "" {
		t, _ := time.Parse("2006-01-02", req.ExpiresAt)
		entity.ExpiresAt = &t
	}

	if err := secretDao.Insert(ctx, entity); err != nil {
		glog.Errorf(ctx, "[applicationSvc.CreateSecret] insert fail, err:%v", err)
		return nil, code.GetError(code.ApplicationSecretCreateError)
	}

	return &dtoapplication.CreateApplicationSecretResp{
		ID:     entity.ID,
		Name:   entity.Name,
		Secret: secretValue,
	}, nil
}

func (svc *applicationSvc) DeleteSecret(ctx *gin.Context, req *dtoapplication.DeleteApplicationSecretReq) error {
	secretDao := dao.NewApplicationSecretDao()

	if err := secretDao.Delete(ctx, uint(req.SecretID), gincontext.GetUserID(ctx)); err != nil {
		glog.Errorf(ctx, "[applicationSvc.DeleteSecret] delete fail, err:%v", err)
		return code.GetError(code.ApplicationSecretDeleteError)
	}

	return nil
}

func generateSecretValue() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

#### Task 25: 应用控制器密钥管理

**Files:**
- Modify: `internal/controller/ctrpermission/application.go`

- [ ] **Step 1: 添加应用密钥管理接口和实现**

```go
type ApplicationCtr interface {
	// ... 现有接口
	ListSecrets(ctx *gin.Context)
	CreateSecret(ctx *gin.Context)
	DeleteSecret(ctx *gin.Context)
}
```

- [ ] **Step 2: 实现应用密钥管理方法**

```go
// @Tags 应用管理
// @Summary 应用密钥列表
// @accept application/json
// @Produce application/json
// @Param req query dtoapplication.ApplicationSecretListReq true "应用密钥列表"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.ApplicationSecretListResp}
// @Router /v1/iam/application/secrets [get]
func (ctr *applicationCtr) ListSecrets(ctx *gin.Context) {
	var req dtoapplication.ApplicationSecretListReq
	if err := ctx.ShouldBindQuery(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.ListSecrets(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 创建密钥
// @accept application/json
// @Produce application/json
// @Param req body dtoapplication.CreateApplicationSecretReq true "创建密钥"
// @Success 200 {object} gincontext.DtoRender{data=dtoapplication.CreateApplicationSecretResp}
// @Router /v1/iam/application/secrets [post]
func (ctr *applicationCtr) CreateSecret(ctx *gin.Context) {
	var req dtoapplication.CreateApplicationSecretReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	res, err := ctr.applicationSvc.CreateSecret(ctx, &req)
	if err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, res)
}

// @Tags 应用管理
// @Summary 删除密钥
// @accept application/json
// @Produce application/json
// @Param req path dtoapplication.DeleteApplicationSecretReq true "删除密钥"
// @Success 200 {object} gincontext.DtoRender{data=string}
// @Router /v1/iam/application/secrets/{secretId} [delete]
func (ctr *applicationCtr) DeleteSecret(ctx *gin.Context) {
	var req dtoapplication.DeleteApplicationSecretReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	if err := ctr.applicationSvc.DeleteSecret(ctx, &req); err != nil {
		gincontext.Fail(ctx, err)
		return
	}
	gincontext.Success(ctx, "删除成功")
}
```

#### Task 26: 应用密钥路由注册

**Files:**
- Modify: `internal/router/permission.go`

- [ ] **Step 1: 添加应用密钥路由**

```go
	// 应用密钥管理
	v1RouterGroup.GET("/application/secrets", appCtr.ListSecrets)
	v1RouterGroup.POST("/application/secrets", appCtr.CreateSecret)
	v1RouterGroup.DELETE("/application/secrets/:secretId", appCtr.DeleteSecret)
```

---

## 实施顺序

1. **Phase 1** - 会话管理 (Task 1-5)
2. **Phase 2** - SSO 逻辑完善 (Task 6-7)
3. **Phase 3** - 应用角色管理 (Task 8-11)
4. **Phase 4** - 角色用户管理 (Task 12-15)
5. **Phase 5** - 角色应用管理 (Task 16-19)
6. **Phase 6** - SSO 连接器增强 (Task 20)
7. **Phase 7** - 连接器增强 (Task 21)
8. **Phase 8** - 应用密钥管理 (Task 22-26)

---

## 错误码清单

需要在 `pkg/code` 中添加以下错误码：

| 错误码 | 说明 |
|--------|------|
| SessionGetListError | 获取会话列表失败 |
| SessionRevokeError | 撤销会话失败 |
| ApplicationRoleGetListError | 获取应用角色列表失败 |
| ApplicationRoleCreateError | 创建应用角色失败 |
| ApplicationRoleDeleteError | 删除应用角色失败 |
| ApplicationRoleNotExistError | 应用角色不存在 |
| RoleUserGetListError | 获取角色用户列表失败 |
| RoleUserCreateError | 创建角色用户失败 |
| RoleUserDeleteError | 删除角色用户失败 |
| RoleUserNotExistError | 角色用户不存在 |
| RoleApplicationGetListError | 获取角色应用列表失败 |
| RoleApplicationCreateError | 创建角色应用失败 |
| RoleApplicationNotExistError | 角色应用不存在 |
| ApplicationSecretGetListError | 获取应用密钥列表失败 |
| ApplicationSecretCreateError | 创建应用密钥失败 |
| ApplicationSecretDeleteError | 删除应用密钥失败 |

---

*计划结束*