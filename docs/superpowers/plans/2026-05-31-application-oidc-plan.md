# Application OIDC Provider 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 ark-iam 改造为完整 OIDC 授权服务器，Application 作为 OIDC 客户端注册，完成 authorize/token/userinfo 等标准端点

**Architecture:** 新增 `svcoidc` 服务层处理 OIDC 业务流程，`ctroidc` 控制器层暴露 HTTP 端点，复用现有 Person/User/Tenant 认证体系和 Connector 身份源。Application 模型从通用 CRUD 改为结构化 OIDC 客户端注册

**Tech Stack:** Go 1.22+, Gin, GORM, `github.com/golang-jwt/jwt/v5`, `github.com/google/uuid`, Redis

---

### Task 1: OIDC 错误码定义

**Files:**
- Create: `backend/pkg/code/oidc.go`
- Modify: `backend/pkg/code/code.go`

- [ ] **Step 1: Create OIDC error codes**

```go
// backend/pkg/code/oidc.go
package code

import "github.com/morehao/golib/gerror"

const (
    OIDCInvalidRequest           = 100790
    OIDCUnauthorizedClient       = 100791
    OIDCAccessDenied             = 100792
    OIDCUnsupportedResponseType  = 100793
    OIDCInvalidScope             = 100794
    OIDCInvalidGrant             = 100795
    OIDCInvalidClient            = 100796
    OIDCServerError              = 100797
    OIDCTemporarilyUnavailable   = 100798
    OIDCSessionNotFound          = 100799
)

var oidcErrorMsgMap = gerror.CodeMsgMap{
    OIDCInvalidRequest:           "OIDC invalid request",
    OIDCUnauthorizedClient:       "OIDC unauthorized client",
    OIDCAccessDenied:             "OIDC access denied",
    OIDCUnsupportedResponseType:  "OIDC unsupported response type",
    OIDCInvalidScope:             "OIDC invalid scope",
    OIDCInvalidGrant:             "OIDC invalid grant",
    OIDCInvalidClient:            "OIDC invalid client",
    OIDCServerError:              "OIDC server error",
    OIDCTemporarilyUnavailable:   "OIDC temporarily unavailable",
    OIDCSessionNotFound:          "OIDC session not found",
}
```

- [ ] **Step 2: Register OIDC errors in code.go**

Insert in `backend/pkg/code/code.go` after `registerError(auditErrorMsgMap)`:
```go
registerError(oidcErrorMsgMap)
```

- [ ] **Step 3: Build and verify**

Run: `cd backend && go build ./pkg/code/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add backend/pkg/code/oidc.go backend/pkg/code/code.go
git commit -m "feat(iam): add OIDC error codes"
```

---

### Task 2: Application 模型重构

**Files:**
- Modify: `backend/apps/iam/model/application.go`
- Modify: `backend/apps/iam/model/application_secret.go`
- Modify: `backend/apps/iam/object/objapplication/application.go`
- Modify: `backend/apps/iam/internal/dto/dtoapplication/request.go`
- Modify: `backend/apps/iam/internal/dto/dtoapplication/response.go`
- Modify: `backend/apps/iam/dao/application.go`
- Modify: `backend/apps/iam/dao/application_secret.go`
- Modify: `backend/apps/iam/internal/service/svcapplication/application.go`
- Modify: `backend/scripts/sql/iam_schema.sql`

- [ ] **Step 1: Update ApplicationEntity model**

Replace `backend/apps/iam/model/application.go` content:

```go
package model

import (
    "gorm.io/gorm"
    "gorm.io/datatypes"
)

const TableNameApplication = "application"

type ApplicationEntity struct {
    gorm.Model
    TenantID              uint           `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
    ClientID              string         `gorm:"column:client_id;type:varchar(64);not null;default '';uniqueIndex;comment:OIDC客户端ID" json:"clientID"`
    Name                  string         `gorm:"column:name;type:varchar(256);not null;default '';comment:应用名称" json:"name"`
    Description           string         `gorm:"column:description;type:text;comment:应用描述" json:"description"`
    LogoURL               string         `gorm:"column:logo_url;type:varchar(2048);not null;default '';comment:应用logo" json:"logoURL"`
    HomepageURL           string         `gorm:"column:homepage_url;type:varchar(2048);not null;default '';comment:应用主页" json:"homepageURL"`
    Type                  string         `gorm:"column:type;type:varchar(32);not null;default 'first_party';comment:应用类型" json:"type"`
    Status                string         `gorm:"column:status;type:varchar(32);not null;default 'enable';comment:状态" json:"status"`
    IsThirdParty          int8           `gorm:"column:is_third_party;type:tinyint(1);not null;default 0;comment:是否第三方应用" json:"isThirdParty"`

    // OIDC 结构化配置
    RedirectURIs            datatypes.JSON `gorm:"column:redirect_uris;type:json;not null;default ('[]');comment:授权回调地址" json:"redirectURIs"`
    PostLogoutRedirectURIs  datatypes.JSON `gorm:"column:post_logout_redirect_uris;type:json;not null;default ('[]');comment:登出回调地址" json:"postLogoutRedirectURIs"`
    GrantTypes              datatypes.JSON `gorm:"column:grant_types;type:json;not null;default ('[\"authorization_code\"]');comment:授权类型" json:"grantTypes"`
    ResponseTypes           datatypes.JSON `gorm:"column:response_types;type:json;not null;default ('[\"code\"]');comment:响应类型" json:"responseTypes"`
    TokenEndpointAuthMethod string         `gorm:"column:token_endpoint_auth_method;type:varchar(32);not null;default 'client_secret_basic';comment:令牌端点认证方式" json:"tokenEndpointAuthMethod"`
    AllowedOrigins          datatypes.JSON `gorm:"column:allowed_origins;type:json;not null;default ('[]');comment:CORS白名单" json:"allowedOrigins"`
    RequirePKCE             int8           `gorm:"column:require_pkce;type:tinyint(1);not null;default 0;comment:是否强制PKCE" json:"requirePKCE"`
    RequireAuthTime         int8           `gorm:"column:require_auth_time;type:tinyint(1);not null;default 0;comment:是否需要auth_time声明" json:"requireAuthTime"`
    DefaultScopes           datatypes.JSON `gorm:"column:default_scopes;type:json;not null;default ('[\"openid\",\"profile\"]');comment:默认权限范围" json:"defaultScopes"`
    AccessTokenTTL          int64          `gorm:"column:access_token_ttl;type:bigint;not null;default 3600;comment:访问令牌有效期(秒)" json:"accessTokenTTL"`
    RefreshTokenTTL         int64          `gorm:"column:refresh_token_ttl;type:bigint;not null;default 2592000;comment:刷新令牌有效期(秒)" json:"refreshTokenTTL"`

    CreatedBy uint `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
    UpdatedBy uint `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
    DeletedBy uint `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (ApplicationEntity) TableName() string {
    return TableNameApplication
}

type ApplicationEntityList []ApplicationEntity

func (l ApplicationEntityList) ToMap() map[uint]ApplicationEntity {
    m := make(map[uint]ApplicationEntity)
    for _, v := range l {
        m[v.ID] = v
    }
    return m
}
```

- [ ] **Step 2: Update ApplicationSecretEntity model**

```go
// backend/apps/iam/model/application_secret.go
package model

import (
    "time"
    "gorm.io/gorm"
)

const TableNameApplicationSecret = "application_secret"

type ApplicationSecretEntity struct {
    gorm.Model
    ApplicationID uint       `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:应用ID" json:"applicationID"`
    Name          string     `gorm:"column:name;type:varchar(256);not null;default '';comment:密钥名称" json:"name"`
    ValueHash     string     `gorm:"column:value_hash;type:varchar(256);not null;default '';comment:密钥哈希" json:"-"`
    ValuePrefix   string     `gorm:"column:value_prefix;type:varchar(16);not null;default '';comment:密钥前缀" json:"valuePrefix"`
    ExpiredAt     *time.Time `gorm:"column:expired_at;type:datetime;comment:过期时间" json:"expiresAt"`
    RevokedAt     *time.Time `gorm:"column:revoked_at;type:datetime;comment:撤销时间" json:"-"`
    CreatedBy     uint       `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
    UpdatedBy     uint       `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
    DeletedBy     uint       `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (ApplicationSecretEntity) TableName() string {
    return TableNameApplicationSecret
}

type ApplicationSecretEntityList []ApplicationSecretEntity

func (l ApplicationSecretEntityList) ToMap() map[uint]ApplicationSecretEntity {
    m := make(map[uint]ApplicationSecretEntity)
    for _, v := range l {
        m[v.ID] = v
    }
    return m
}
```

- [ ] **Step 3: Update ApplicationBaseInfo object**

```go
// backend/apps/iam/object/objapplication/application.go
package objapplication

type ApplicationBaseInfo struct {
    TenantID    uint   `json:"tenantID"`
    ClientID    string `json:"clientID"`
    Name        string `json:"name"`
    Description string `json:"description"`
    LogoURL     string `json:"logoURL"`
    HomepageURL string `json:"homepageURL"`
    Type        string `json:"type"`
    Status      string `json:"status"`
    IsThirdParty int8  `json:"isThirdParty"`
}
```

- [ ] **Step 4: Update DTOs - request.go**

```go
// backend/apps/iam/internal/dto/dtoapplication/request.go
package dtoapplication

type ApplicationCreateReq struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    LogoURL     string `json:"logoURL"`
    HomepageURL string `json:"homepageURL"`
    Type        string `json:"type"`
    IsThirdParty int8  `json:"isThirdParty"`

    RedirectURIs            []string `json:"redirectURIs"`
    PostLogoutRedirectURIs  []string `json:"postLogoutRedirectURIs"`
    GrantTypes              []string `json:"grantTypes"`
    ResponseTypes           []string `json:"responseTypes"`
    TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`
    AllowedOrigins          []string `json:"allowedOrigins"`
    RequirePKCE             int8     `json:"requirePKCE"`
    RequireAuthTime         int8     `json:"requireAuthTime"`
    DefaultScopes           []string `json:"defaultScopes"`
    AccessTokenTTL          int64    `json:"accessTokenTTL"`
    RefreshTokenTTL         int64    `json:"refreshTokenTTL"`
}

type ApplicationUpdateReq struct {
    ApplicationID uint   `json:"applicationID"`
    Name          string `json:"name"`
    Description   string `json:"description"`
    LogoURL       string `json:"logoURL"`
    HomepageURL   string `json:"homepageURL"`
    Type          string `json:"type"`
    Status        string `json:"status"`
    IsThirdParty  int8   `json:"isThirdParty"`

    RedirectURIs            []string `json:"redirectURIs"`
    PostLogoutRedirectURIs  []string `json:"postLogoutRedirectURIs"`
    GrantTypes              []string `json:"grantTypes"`
    ResponseTypes           []string `json:"responseTypes"`
    TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`
    AllowedOrigins          []string `json:"allowedOrigins"`
    RequirePKCE             int8     `json:"requirePKCE"`
    RequireAuthTime         int8     `json:"requireAuthTime"`
    DefaultScopes           []string `json:"defaultScopes"`
    AccessTokenTTL          int64    `json:"accessTokenTTL"`
    RefreshTokenTTL         int64    `json:"refreshTokenTTL"`
}

type ApplicationDeleteReq struct {
    ApplicationID uint `json:"applicationID"`
}

type ApplicationDetailReq struct {
    ApplicationID uint `json:"applicationID"`
}

type ApplicationPageListReq struct {
    Page        int    `json:"page"`
    PageSize    int    `json:"pageSize"`
    Name        string `json:"name"`
    Type        string `json:"type"`
    Status      string `json:"status"`
}

type ApplicationRoleListReq struct {
    ApplicationID uint `json:"applicationId" form:"applicationId" binding:"required"`
}

type AssignApplicationRolesReq struct {
    ApplicationID uint64   `json:"applicationId" binding:"required"`
    RoleIDs       []uint64 `json:"roleIds" binding:"required,min=1"`
}

type RemoveApplicationRoleReq struct {
    ApplicationID uint64 `json:"applicationId" form:"applicationId" binding:"required"`
    RoleID        uint64 `json:"roleId" uri:"roleId" binding:"required"`
}

type ApplicationSecretListReq struct {
    ApplicationID uint `json:"applicationId" form:"applicationId" binding:"required"`
}

type CreateApplicationSecretReq struct {
    ApplicationID uint   `json:"applicationId" binding:"required"`
    Name          string `json:"name" binding:"required"`
    ExpiredAt     string `json:"expiresAt"`
}

type DeleteApplicationSecretReq struct {
    SecretID uint64 `json:"secretId" uri:"secretId" binding:"required"`
}
```

- [ ] **Step 5: Update DTOs - response.go**

```go
// backend/apps/iam/internal/dto/dtoapplication/response.go
package dtoapplication

import "github.com/morehao/ark-iam/iam/object/objapplication"

type ApplicationCreateResp struct {
    ApplicationID uint   `json:"applicationID"`
    ClientID      string `json:"clientID"`
}

type ApplicationDetailResp struct {
    ApplicationID            uint   `json:"applicationID"`
    objapplication.ApplicationBaseInfo `json:"applicationBaseInfo"`
}

type ApplicationPageListResp struct {
    List  []ApplicationPageListItem `json:"list"`
    Total int64                     `json:"total"`
}

type ApplicationPageListItem struct {
    ApplicationID            uint `json:"applicationID"`
    objapplication.ApplicationBaseInfo `json:"applicationBaseInfo"`
}

type ApplicationRoleResp struct {
    RoleID        uint64 `json:"roleId"`
    RoleName      string `json:"roleName"`
    RoleCode      string `json:"roleCode"`
    ApplicationID uint64 `json:"applicationId"`
    CreatedAt     string `json:"createdAt"`
}

type ApplicationRoleListResp struct {
    Total int64                 `json:"total"`
    Roles []ApplicationRoleResp `json:"roles"`
}

type ApplicationSecretResp struct {
    ID            uint64  `json:"id"`
    ApplicationID uint64  `json:"applicationId"`
    Name          string  `json:"name"`
    ValuePrefix   string  `json:"valuePrefix"`
    ExpiredAt     *string `json:"expiresAt"`
    CreatedAt     string  `json:"createdAt"`
}

type ApplicationSecretListResp struct {
    Total   int64                    `json:"total"`
    Secrets []ApplicationSecretResp `json:"secrets"`
}

type CreateApplicationSecretResp struct {
    ID          uint64 `json:"id"`
    Name        string `json:"name"`
    ValuePrefix string `json:"valuePrefix"`
    Secret      string `json:"secret"`
}
```

- [ ] **Step 6: Update Application DAO - add client_id query**

```go
// backend/apps/iam/dao/application.go
package dao

import (
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/dbclient"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

type ApplicationCond struct {
    *genericdao.BaseCond
    TenantID uint
    Name     string
    Type     string
    Status   string
    ClientID string
}

func (c *ApplicationCond) BuildCondition(db *gorm.DB, tableName string) {
    if c.BaseCond != nil {
        c.BaseCond.BuildCondition(db, tableName)
    }
    if c.TenantID != 0 {
        db.Where(tableName + ".tenant_id = ?", c.TenantID)
    }
    if c.Name != "" {
        db.Where(tableName + ".name = ?", c.Name)
    }
    if c.Type != "" {
        db.Where(tableName + ".type = ?", c.Type)
    }
    if c.Status != "" {
        db.Where(tableName + ".status = ?", c.Status)
    }
    if c.ClientID != "" {
        db.Where(tableName + ".client_id = ?", c.ClientID)
    }
}

type ApplicationDao struct {
    *genericdao.GenericDao[model.ApplicationEntity, model.ApplicationEntityList]
}

func NewApplicationDao() *ApplicationDao {
    return &ApplicationDao{
        GenericDao: genericdao.NewGenericDao[model.ApplicationEntity, model.ApplicationEntityList](
            model.TableNameApplication, "ApplicationDao",
            dbclient.IamDB,
        ),
    }
}
```

- [ ] **Step 7: Update ApplicationSecret DAO - remove tenant_id**

```go
// backend/apps/iam/dao/application_secret.go
package dao

import (
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/dbclient"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

type ApplicationSecretCond struct {
    *genericdao.BaseCond
    ApplicationID uint
    Name          string
}

func (c *ApplicationSecretCond) BuildCondition(db *gorm.DB, tableName string) {
    if c.BaseCond != nil {
        c.BaseCond.BuildCondition(db, tableName)
    }
    if c.ApplicationID != 0 {
        db.Where(tableName + ".application_id = ?", c.ApplicationID)
    }
    if c.Name != "" {
        db.Where(tableName + ".name = ?", c.Name)
    }
}

type ApplicationSecretDao struct {
    *genericdao.GenericDao[model.ApplicationSecretEntity, model.ApplicationSecretEntityList]
}

func NewApplicationSecretDao() *ApplicationSecretDao {
    return &ApplicationSecretDao{
        GenericDao: genericdao.NewGenericDao[model.ApplicationSecretEntity, model.ApplicationSecretEntityList](
            model.TableNameApplicationSecret, "ApplicationSecretDao",
            dbclient.IamDB,
        ),
    }
}
```

- [ ] **Step 8: Update Application Service - rewrite CRUD**

Replace `backend/apps/iam/internal/service/svcapplication/application.go`:

```go
package svcapplication

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/iam/object/objapplication"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/biz/gcontext/gincontext"
    "github.com/morehao/golib/biz/genericdao"
    "github.com/morehao/golib/gcrypto"
    "github.com/morehao/golib/glog"
    "github.com/morehao/golib/gutil"
    "gorm.io/datatypes"
)

type ApplicationSvc interface {
    Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error)
    Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error
    Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error
    Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error)
    PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error)
    GetByClientID(ctx context.Context, clientID string) (*model.ApplicationEntity, error)
    ListRoles(ctx *gin.Context, req *dtoapplication.ApplicationRoleListReq) (*dtoapplication.ApplicationRoleListResp, error)
    AssignRoles(ctx *gin.Context, req *dtoapplication.AssignApplicationRolesReq) error
    RemoveRole(ctx *gin.Context, req *dtoapplication.RemoveApplicationRoleReq) error
    ListSecrets(ctx *gin.Context, req *dtoapplication.ApplicationSecretListReq) (*dtoapplication.ApplicationSecretListResp, error)
    CreateSecret(ctx *gin.Context, req *dtoapplication.CreateApplicationSecretReq) (*dtoapplication.CreateApplicationSecretResp, error)
    DeleteSecret(ctx *gin.Context, req *dtoapplication.DeleteApplicationSecretReq) error
}

type applicationSvc struct{}

type applicationRoleListReader interface {
    GetListByCond(ctx context.Context, cond genericdao.Cond) (model.ApplicationRoleEntityList, error)
}

type roleReader interface {
    GetByID(ctx context.Context, id uint) (*model.RoleEntity, error)
}

type applicationScopeRepository interface {
    GetByID(ctx context.Context, id uint) (*model.ApplicationEntity, error)
    GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.ApplicationEntityList, int64, error)
    GetSecretByID(ctx context.Context, id uint) (*model.ApplicationSecretEntity, error)
    DeleteSecret(ctx context.Context, id uint, userID uint) error
}

var newApplicationRoleListReader = func() applicationRoleListReader {
    return dao.NewApplicationRoleDao()
}

var newRoleReader = func() roleReader {
    return dao.NewRoleDao()
}

var newApplicationScopeRepo = func() applicationScopeRepository {
    return &applicationScopeDAO{}
}

type applicationScopeDAO struct{}

func (d *applicationScopeDAO) GetByID(ctx context.Context, id uint) (*model.ApplicationEntity, error) {
    return dao.NewApplicationDao().GetByID(ctx, id)
}

func (d *applicationScopeDAO) GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.ApplicationEntityList, int64, error) {
    return dao.NewApplicationDao().GetPageListByCond(ctx, cond)
}

func (d *applicationScopeDAO) GetSecretByID(ctx context.Context, id uint) (*model.ApplicationSecretEntity, error) {
    return dao.NewApplicationSecretDao().GetByID(ctx, id)
}

func (d *applicationScopeDAO) DeleteSecret(ctx context.Context, id uint, userID uint) error {
    return dao.NewApplicationSecretDao().Delete(ctx, id, userID)
}

func applicationVisibleToTenant(entity *model.ApplicationEntity, tenantID uint) bool {
    return entity != nil && entity.ID != 0 && entity.TenantID == tenantID
}

var _ ApplicationSvc = (*applicationSvc)(nil)

func NewApplicationSvc() ApplicationSvc {
    return &applicationSvc{}
}

func generateClientID() string {
    return "app_" + uuid.New().String()
}

func marshalStringSlice(v []string) datatypes.JSON {
    if v == nil {
        v = []string{}
    }
    data, _ := datatypes.JSON.MarshalJSON(v)
    return data
}

func (svc *applicationSvc) Create(ctx *gin.Context, req *dtoapplication.ApplicationCreateReq) (*dtoapplication.ApplicationCreateResp, error) {
    insertEntity := &model.ApplicationEntity{
        TenantID:              gincontext.GetTenantID(ctx),
        ClientID:              generateClientID(),
        Name:                  req.Name,
        Description:           req.Description,
        LogoURL:               req.LogoURL,
        HomepageURL:           req.HomepageURL,
        Type:                  req.Type,
        IsThirdParty:          req.IsThirdParty,
        Status:                "enable",
        RedirectURIs:          marshalStringSlice(req.RedirectURIs),
        PostLogoutRedirectURIs: marshalStringSlice(req.PostLogoutRedirectURIs),
        GrantTypes:            marshalStringSlice(req.GrantTypes),
        ResponseTypes:         marshalStringSlice(req.ResponseTypes),
        TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
        AllowedOrigins:        marshalStringSlice(req.AllowedOrigins),
        RequirePKCE:           req.RequirePKCE,
        RequireAuthTime:       req.RequireAuthTime,
        DefaultScopes:         marshalStringSlice(req.DefaultScopes),
        AccessTokenTTL:        req.AccessTokenTTL,
        RefreshTokenTTL:       req.RefreshTokenTTL,
        CreatedBy:             gincontext.GetUserID(ctx),
    }
    if insertEntity.TokenEndpointAuthMethod == "" {
        insertEntity.TokenEndpointAuthMethod = "client_secret_basic"
    }
    if len(req.GrantTypes) == 0 {
        insertEntity.GrantTypes = marshalStringSlice([]string{"authorization_code"})
    }
    if len(req.ResponseTypes) == 0 {
        insertEntity.ResponseTypes = marshalStringSlice([]string{"code"})
    }
    if req.AccessTokenTTL <= 0 {
        insertEntity.AccessTokenTTL = 3600
    }
    if req.RefreshTokenTTL <= 0 {
        insertEntity.RefreshTokenTTL = 2592000
    }

    if err := dao.NewApplicationDao().Insert(ctx, insertEntity); err != nil {
        glog.Errorf(ctx, "[svcapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.ApplicationCreateError)
    }
    return &dtoapplication.ApplicationCreateResp{
        ApplicationID: insertEntity.ID,
        ClientID:      insertEntity.ClientID,
    }, nil
}

func (svc *applicationSvc) Delete(ctx *gin.Context, req *dtoapplication.ApplicationDeleteReq) error {
    appEntity, err := newApplicationScopeRepo().GetByID(ctx, req.ApplicationID)
    if err != nil {
        glog.Errorf(ctx, "[svcapplication.Delete] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.ApplicationDeleteError)
    }
    if !applicationVisibleToTenant(appEntity, gincontext.GetTenantID(ctx)) {
        return code.GetError(code.ApplicationNotExistError)
    }
    userID := gincontext.GetUserID(ctx)
    if err := dao.NewApplicationDao().Delete(ctx, req.ApplicationID, userID); err != nil {
        glog.Errorf(ctx, "[svcapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.ApplicationDeleteError)
    }
    return nil
}

func (svc *applicationSvc) Update(ctx *gin.Context, req *dtoapplication.ApplicationUpdateReq) error {
    appEntity, err := newApplicationScopeRepo().GetByID(ctx, req.ApplicationID)
    if err != nil {
        glog.Errorf(ctx, "[svcapplication.Update] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.ApplicationUpdateError)
    }
    if !applicationVisibleToTenant(appEntity, gincontext.GetTenantID(ctx)) {
        return code.GetError(code.ApplicationNotExistError)
    }

    userID := gincontext.GetUserID(ctx)
    updateMap := map[string]any{
        "name":                      req.Name,
        "description":               req.Description,
        "logo_url":                  req.LogoURL,
        "homepage_url":              req.HomepageURL,
        "type":                      req.Type,
        "status":                    req.Status,
        "is_third_party":            req.IsThirdParty,
        "redirect_uris":             marshalStringSlice(req.RedirectURIs),
        "post_logout_redirect_uris": marshalStringSlice(req.PostLogoutRedirectURIs),
        "grant_types":               marshalStringSlice(req.GrantTypes),
        "response_types":            marshalStringSlice(req.ResponseTypes),
        "token_endpoint_auth_method": req.TokenEndpointAuthMethod,
        "allowed_origins":           marshalStringSlice(req.AllowedOrigins),
        "require_pkce":              req.RequirePKCE,
        "require_auth_time":         req.RequireAuthTime,
        "default_scopes":            marshalStringSlice(req.DefaultScopes),
        "access_token_ttl":          req.AccessTokenTTL,
        "refresh_token_ttl":         req.RefreshTokenTTL,
        "updated_by":                userID,
    }
    if err := dao.NewApplicationDao().UpdateMap(ctx, req.ApplicationID, updateMap); err != nil {
        glog.Errorf(ctx, "[svcapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.ApplicationUpdateError)
    }
    return nil
}

func (svc *applicationSvc) Detail(ctx *gin.Context, req *dtoapplication.ApplicationDetailReq) (*dtoapplication.ApplicationDetailResp, error) {
    appEntity, err := newApplicationScopeRepo().GetByID(ctx, req.ApplicationID)
    if err != nil {
        glog.Errorf(ctx, "[svcapplication.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.ApplicationGetDetailError)
    }
    if !applicationVisibleToTenant(appEntity, gincontext.GetTenantID(ctx)) {
        return nil, code.GetError(code.ApplicationNotExistError)
    }
    resp := &dtoapplication.ApplicationDetailResp{
        ApplicationID: appEntity.ID,
        ApplicationBaseInfo: objapplication.ApplicationBaseInfo{
            TenantID:     appEntity.TenantID,
            ClientID:     appEntity.ClientID,
            Name:         appEntity.Name,
            Description:  appEntity.Description,
            LogoURL:      appEntity.LogoURL,
            HomepageURL:  appEntity.HomepageURL,
            Type:         appEntity.Type,
            Status:       appEntity.Status,
            IsThirdParty: appEntity.IsThirdParty,
        },
    }
    return resp, nil
}

func (svc *applicationSvc) PageList(ctx *gin.Context, req *dtoapplication.ApplicationPageListReq) (*dtoapplication.ApplicationPageListResp, error) {
    appRepo := newApplicationScopeRepo()
    cond := &dao.ApplicationCond{
        BaseCond: &genericdao.BaseCond{
            Page:     req.Page,
            PageSize: req.PageSize,
        },
        TenantID: gincontext.GetTenantID(ctx),
        Name:     req.Name,
        Type:     req.Type,
        Status:   req.Status,
    }
    appEntityList, total, err := appRepo.GetPageListByCond(ctx, cond)
    if err != nil {
        glog.Errorf(ctx, "[svcapplication.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.ApplicationGetPageListError)
    }
    list := make([]dtoapplication.ApplicationPageListItem, 0, len(appEntityList))
    for _, v := range appEntityList {
        list = append(list, dtoapplication.ApplicationPageListItem{
            ApplicationID: v.ID,
            ApplicationBaseInfo: objapplication.ApplicationBaseInfo{
                TenantID:     v.TenantID,
                ClientID:     v.ClientID,
                Name:         v.Name,
                Description:  v.Description,
                LogoURL:      v.LogoURL,
                HomepageURL:  v.HomepageURL,
                Type:         v.Type,
                Status:       v.Status,
                IsThirdParty: v.IsThirdParty,
            },
        })
    }
    return &dtoapplication.ApplicationPageListResp{
        List:  list,
        Total: total,
    }, nil
}

func (svc *applicationSvc) GetByClientID(ctx context.Context, clientID string) (*model.ApplicationEntity, error) {
    return dao.NewApplicationDao().GetByCond(ctx, &dao.ApplicationCond{ClientID: clientID})
}

func (svc *applicationSvc) ListRoles(ctx *gin.Context, req *dtoapplication.ApplicationRoleListReq) (*dtoapplication.ApplicationRoleListResp, error) {
    appRoleDao := newApplicationRoleListReader()
    roleDao := newRoleReader()
    list, err := appRoleDao.GetListByCond(ctx, &dao.ApplicationRoleCond{ApplicationID: req.ApplicationID})
    if err != nil {
        glog.Errorf(ctx, "[applicationSvc.ListRoles] get roles fail, err:%v", err)
        return nil, code.GetError(code.RoleApplicationGetListError)
    }
    roleMap := make(map[uint]*model.RoleEntity, len(list))
    for _, item := range list {
        if role, err := roleDao.GetByID(ctx, item.RoleID); err == nil && role != nil {
            roleMap[role.ID] = role
        }
    }
    roles := make([]dtoapplication.ApplicationRoleResp, 0, len(list))
    for _, item := range list {
        if role, ok := roleMap[item.RoleID]; ok {
            roles = append(roles, dtoapplication.ApplicationRoleResp{
                RoleID:        uint64(item.RoleID),
                RoleName:      role.Name,
                RoleCode:      role.Code,
                ApplicationID: uint64(item.ApplicationID),
                CreatedAt:     item.CreatedAt.Format("2006-01-02 15:04:05"),
            })
        }
    }
    return &dtoapplication.ApplicationRoleListResp{
        Total: int64(len(roles)),
        Roles: roles,
    }, nil
}

func (svc *applicationSvc) AssignRoles(ctx *gin.Context, req *dtoapplication.AssignApplicationRolesReq) error {
    appRoleDao := dao.NewApplicationRoleDao()
    userID := gincontext.GetUserID(ctx)
    for _, roleID := range req.RoleIDs {
        existing, _ := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
            ApplicationID: uint(req.ApplicationID),
            RoleID:        uint(roleID),
        })
        if existing != nil && existing.ID != 0 {
            continue
        }
        entity := &model.ApplicationRoleEntity{
            TenantID:      gincontext.GetTenantID(ctx),
            ApplicationID: uint(req.ApplicationID),
            RoleID:        uint(roleID),
            CreatedBy:     userID,
        }
        if err := appRoleDao.Insert(ctx, entity); err != nil {
            glog.Errorf(ctx, "[applicationSvc.AssignRoles] insert fail, err:%v", err)
            return code.GetError(code.RoleApplicationCreateError)
        }
    }
    return nil
}

func (svc *applicationSvc) RemoveRole(ctx *gin.Context, req *dtoapplication.RemoveApplicationRoleReq) error {
    appRoleDao := dao.NewApplicationRoleDao()
    entity, err := appRoleDao.GetByCond(ctx, &dao.ApplicationRoleCond{
        ApplicationID: uint(req.ApplicationID),
        RoleID:        uint(req.RoleID),
    })
    if err != nil || entity == nil || entity.ID == 0 {
        return code.GetError(code.RoleApplicationNotExistError)
    }
    if err := appRoleDao.Delete(ctx, entity.ID, gincontext.GetUserID(ctx)); err != nil {
        glog.Errorf(ctx, "[applicationSvc.RemoveRole] delete fail, err:%v", err)
        return code.GetError(code.RoleApplicationDeleteError)
    }
    return nil
}

func (svc *applicationSvc) ListSecrets(ctx *gin.Context, req *dtoapplication.ApplicationSecretListReq) (*dtoapplication.ApplicationSecretListResp, error) {
    secretDao := dao.NewApplicationSecretDao()
    list, total, err := secretDao.GetPageListByCond(ctx, &dao.ApplicationSecretCond{
        BaseCond:      &genericdao.BaseCond{Page: 1, PageSize: 100},
        ApplicationID: req.ApplicationID,
    })
    if err != nil {
        glog.Errorf(ctx, "[applicationSvc.ListSecrets] get secrets fail, err:%v", err)
        return nil, code.GetError(code.ApplicationSecretGetListError)
    }
    secrets := make([]dtoapplication.ApplicationSecretResp, 0, len(list))
    for _, s := range list {
        var expiresAt *string
        if s.ExpiredAt != nil {
            t := s.ExpiredAt.Format("2006-01-02 15:04:05")
            expiresAt = &t
        }
        secrets = append(secrets, dtoapplication.ApplicationSecretResp{
            ID:            uint64(s.ID),
            ApplicationID: uint64(s.ApplicationID),
            Name:          s.Name,
            ValuePrefix:   s.ValuePrefix,
            ExpiredAt:     expiresAt,
            CreatedAt:     s.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    return &dtoapplication.ApplicationSecretListResp{
        Total:   total,
        Secrets: secrets,
    }, nil
}

func (svc *applicationSvc) CreateSecret(ctx *gin.Context, req *dtoapplication.CreateApplicationSecretReq) (*dtoapplication.CreateApplicationSecretResp, error) {
    randomBytes, err := gcrypto.GenerateRandomBytes(32)
    if err != nil {
        glog.Errorf(ctx, "[applicationSvc.CreateSecret] generate secret fail, err:%v", err)
        return nil, code.GetError(code.ApplicationSecretCreateError)
    }
    secretValue := hex.EncodeToString(randomBytes)

    hash := sha256.Sum256([]byte(secretValue))
    valueHash := hex.EncodeToString(hash[:])
    valuePrefix := secretValue[:8]

    var expiresAt *time.Time
    if req.ExpiredAt != "" {
        t, err := time.Parse("2006-01-02 15:04:05", req.ExpiredAt)
        if err == nil {
            expiresAt = &t
        }
    }

    entity := &model.ApplicationSecretEntity{
        ApplicationID: req.ApplicationID,
        Name:          req.Name,
        ValueHash:     valueHash,
        ValuePrefix:   valuePrefix,
        ExpiredAt:     expiresAt,
        CreatedBy:     gincontext.GetUserID(ctx),
    }

    if err := dao.NewApplicationSecretDao().Insert(ctx, entity); err != nil {
        glog.Errorf(ctx, "[applicationSvc.CreateSecret] insert fail, err:%v", err)
        return nil, code.GetError(code.ApplicationSecretCreateError)
    }

    return &dtoapplication.CreateApplicationSecretResp{
        ID:          uint64(entity.ID),
        Name:        entity.Name,
        ValuePrefix: valuePrefix,
        Secret:      secretValue,
    }, nil
}

func (svc *applicationSvc) DeleteSecret(ctx *gin.Context, req *dtoapplication.DeleteApplicationSecretReq) error {
    appRepo := newApplicationScopeRepo()
    entity, err := appRepo.GetSecretByID(ctx, uint(req.SecretID))
    if err != nil {
        glog.Errorf(ctx, "[applicationSvc.DeleteSecret] get secret fail, err:%v", err)
        return code.GetError(code.ApplicationSecretDeleteError)
    }
    if entity == nil || entity.ID == 0 || entity.ApplicationID != gincontext.GetTenantID(ctx) {
        return code.GetError(code.ApplicationSecretNotExistError)
    }
    // Verify secret belongs to a tenant-visible application
    appEntity, err := appRepo.GetByID(ctx, entity.ApplicationID)
    if err != nil || !applicationVisibleToTenant(appEntity, gincontext.GetTenantID(ctx)) {
        return code.GetError(code.ApplicationSecretNotExistError)
    }
    if err := appRepo.DeleteSecret(ctx, uint(req.SecretID), gincontext.GetUserID(ctx)); err != nil {
        glog.Errorf(ctx, "[applicationSvc.DeleteSecret] delete fail, err:%v", err)
        return code.GetError(code.ApplicationSecretDeleteError)
    }
    return nil
}
```

- [ ] **Step 9: Update SQL schema**

Replace the `application` and `application_secret` table definitions in `backend/scripts/sql/iam_schema.sql`:

```sql
CREATE TABLE `application` (
    `id`                         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `tenant_id`                  BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `client_id`                  VARCHAR(64) NOT NULL DEFAULT '',
    `name`                       VARCHAR(256) NOT NULL DEFAULT '',
    `description`                TEXT,
    `logo_url`                   VARCHAR(2048) NOT NULL DEFAULT '',
    `homepage_url`               VARCHAR(2048) NOT NULL DEFAULT '',
    `type`                       VARCHAR(32) NOT NULL DEFAULT 'first_party',
    `status`                     VARCHAR(32) NOT NULL DEFAULT 'enable',
    `is_third_party`             TINYINT(1) NOT NULL DEFAULT 0,
    `redirect_uris`              JSON NOT NULL DEFAULT ('[]'),
    `post_logout_redirect_uris`  JSON NOT NULL DEFAULT ('[]'),
    `grant_types`                JSON NOT NULL DEFAULT ('["authorization_code"]'),
    `response_types`             JSON NOT NULL DEFAULT ('["code"]'),
    `token_endpoint_auth_method` VARCHAR(32) NOT NULL DEFAULT 'client_secret_basic',
    `allowed_origins`            JSON NOT NULL DEFAULT ('[]'),
    `require_pkce`               TINYINT(1) NOT NULL DEFAULT 0,
    `require_auth_time`          TINYINT(1) NOT NULL DEFAULT 0,
    `default_scopes`             JSON NOT NULL DEFAULT ('["openid","profile"]'),
    `access_token_ttl`           BIGINT NOT NULL DEFAULT 3600,
    `refresh_token_ttl`          BIGINT NOT NULL DEFAULT 2592000,
    `created_at`                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`                 DATETIME DEFAULT NULL,
    `created_by`                 BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`                 BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`                 BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_client_id` (`client_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_tenant_type` (`tenant_id`, `type`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `application_secret` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `application_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `name`            VARCHAR(256) NOT NULL DEFAULT '',
    `value_hash`      VARCHAR(256) NOT NULL DEFAULT '',
    `value_prefix`    VARCHAR(16) NOT NULL DEFAULT '',
    `expired_at`      DATETIME DEFAULT NULL,
    `revoked_at`      DATETIME DEFAULT NULL,
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    KEY `idx_application_id` (`application_id`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

- [ ] **Step 10: Add `datatypes` dependency**

If not already in go.mod, add gorm datatypes:
Run: `cd backend && go get gorm.io/datatypes`

- [ ] **Step 11: Build verification**

Run: `cd backend && go build ./...`
Expected: no errors

- [ ] **Step 12: Commit**

```bash
git add backend/
git commit -m "refactor(iam): restructure application model for OIDC provider"
```

---

### Task 3: OIDC Session 与状态管理

**Files:**
- Create: `backend/apps/iam/internal/service/svcoidc/session.go`

- [ ] **Step 1: Create session service**

```go
// backend/apps/iam/internal/service/svcoidc/session.go
package svcoidc

import (
    "context"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "time"

    "github.com/morehao/ark-iam/pkg/dbclient"
    "github.com/morehao/golib/glog"
)

const (
    authCodeTTL       = 5 * time.Minute
    sessionCookieTTL  = 24 * time.Hour

    authCodeKeyPrefix       = "iam:oidc:authcode:"
    sessionKeyPrefix        = "iam:oidc:session:"
)

type AuthorizationCodeData struct {
    ClientID            string `json:"client_id"`
    PersonID            uint   `json:"person_id"`
    TenantID            uint   `json:"tenant_id"`
    UserID              uint   `json:"user_id"`
    Scope               string `json:"scope"`
    Nonce               string `json:"nonce"`
    CodeChallenge       string `json:"code_challenge"`
    CodeChallengeMethod string `json:"code_challenge_method"`
    RedirectURI         string `json:"redirect_uri"`
    ExpiresAt           int64  `json:"expires_at"`
}

type SessionData struct {
    PersonID  uint   `json:"person_id"`
    TenantID  uint   `json:"tenant_id"`
    UserID    uint   `json:"user_id"`
    CreatedAt int64  `json:"created_at"`
    ExpiresAt int64  `json:"expires_at"`
}

type OIDCSessionStore struct{}

func NewOIDCSessionStore() *OIDCSessionStore {
    return &OIDCSessionStore{}
}

func generateRandomString(length int) (string, error) {
    b := make([]byte, length)
    _, err := rand.Read(b)
    if err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *OIDCSessionStore) SaveAuthorizationCode(ctx context.Context, code string, data *AuthorizationCodeData) error {
    key := authCodeKeyPrefix + code
    return dbclient.RedisCli.Set(ctx, key, fmt.Sprintf(`{"client_id":"%s","person_id":%d,"tenant_id":%d,"user_id":%d,"scope":"%s","nonce":"%s","code_challenge":"%s","code_challenge_method":"%s","redirect_uri":"%s","expires_at":%d}`,
        data.ClientID, data.PersonID, data.TenantID, data.UserID, data.Scope, data.Nonce, data.CodeChallenge, data.CodeChallengeMethod, data.RedirectURI, data.ExpiresAt),
        authCodeTTL).Err()
}

func (s *OIDCSessionStore) GetAuthorizationCode(ctx context.Context, code string) (*AuthorizationCodeData, error) {
    key := authCodeKeyPrefix + code
    val, err := dbclient.RedisCli.Get(ctx, key).Result()
    if err != nil {
        return nil, err
    }
    var data AuthorizationCodeData
    if _, parseErr := fmt.Sscanf(val, `{"client_id":"%s","person_id":%d,"tenant_id":%d,"user_id":%d,"scope":"%s","nonce":"%s","code_challenge":"%s","code_challenge_method":"%s","redirect_uri":"%s","expires_at":%d}`,
        &data.ClientID, &data.PersonID, &data.TenantID, &data.UserID, &data.Scope, &data.Nonce, &data.CodeChallenge, &data.CodeChallengeMethod, &data.RedirectURI, &data.ExpiresAt); parseErr != nil {
        return nil, parseErr
    }
    return &data, nil
}

func (s *OIDCSessionStore) ConsumeAuthorizationCode(ctx context.Context, code string) error {
    key := authCodeKeyPrefix + code
    return dbclient.RedisCli.Del(ctx, key).Err()
}

func (s *OIDCSessionStore) GenerateAuthorizationCode() (string, error) {
    return generateRandomString(32)
}

func (s *OIDCSessionStore) SaveSession(ctx context.Context, sessionID string, data *SessionData) error {
    key := sessionKeyPrefix + sessionID
    val := fmt.Sprintf(`{"person_id":%d,"tenant_id":%d,"user_id":%d,"created_at":%d,"expires_at":%d}`,
        data.PersonID, data.TenantID, data.UserID, data.CreatedAt, data.ExpiresAt)
    return dbclient.RedisCli.Set(ctx, key, val, sessionCookieTTL).Err()
}

func (s *OIDCSessionStore) GetSession(ctx context.Context, sessionID string) (*SessionData, error) {
    key := sessionKeyPrefix + sessionID
    val, err := dbclient.RedisCli.Get(ctx, key).Result()
    if err != nil {
        return nil, err
    }
    var data SessionData
    if _, parseErr := fmt.Sscanf(val, `{"person_id":%d,"tenant_id":%d,"user_id":%d,"created_at":%d,"expires_at":%d}`,
        &data.PersonID, &data.TenantID, &data.UserID, &data.CreatedAt, &data.ExpiresAt); parseErr != nil {
        return nil, parseErr
    }
    return &data, nil
}

func (s *OIDCSessionStore) GenerateSessionID() (string, error) {
    return generateRandomString(32)
}
```

- [ ] **Step 2: Build verification**

Run: `cd backend && go build ./apps/iam/internal/service/svcoidc/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/session.go
git commit -m "feat(iam): add OIDC session and auth code store"
```

---

### Task 4: OIDC Client 认证与工具函数

**Files:**
- Create: `backend/apps/iam/internal/service/svcoidc/client_auth.go`
- Create: `backend/apps/iam/internal/service/svcoidc/validator.go`

- [ ] **Step 1: Create client authentication**

```go
// backend/apps/iam/internal/service/svcoidc/client_auth.go
package svcoidc

import (
    "context"
    "crypto/sha256"
    "crypto/subtle"
    "encoding/base64"
    "encoding/hex"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/gerror"
)

type applicationReader interface {
    GetByClientID(ctx context.Context, clientID string) (*model.ApplicationEntity, error)
}

type secretReader interface {
    GetByCond(ctx context.Context, cond interface{}) (*model.ApplicationSecretEntity, error)
}

type ClientAuthenticator struct {
    appReader  applicationReader
    secretReader secretReader
}

func NewClientAuthenticator(appReader applicationReader, secretReader secretReader) *ClientAuthenticator {
    return &ClientAuthenticator{appReader: appReader, secretReader: secretReader}
}

type AuthenticatedClient struct {
    Application *model.ApplicationEntity
    ClientID    string
}

func extractClientCredentials(ctx *gin.Context) (clientID string, clientSecret string) {
    auth := ctx.GetHeader("Authorization")
    if strings.HasPrefix(auth, "Basic ") {
        payload := strings.TrimPrefix(auth, "Basic ")
        decoded, err := base64.StdEncoding.DecodeString(payload)
        if err == nil {
            parts := strings.SplitN(string(decoded), ":", 2)
            if len(parts) == 2 {
                return parts[0], parts[1]
            }
        }
    }
    clientID = ctx.PostForm("client_id")
    clientSecret = ctx.PostForm("client_secret")
    return
}

func (a *ClientAuthenticator) AuthenticateClient(ctx *gin.Context) (*AuthenticatedClient, *gerror.Error) {
    clientID, clientSecret := extractClientCredentials(ctx)
    if clientID == "" {
        return nil, code.GetError(code.OIDCInvalidClient)
    }
    appEntity, err := a.appReader.GetByClientID(ctx, clientID)
    if err != nil || appEntity == nil || appEntity.ID == 0 {
        return nil, code.GetError(code.OIDCInvalidClient)
    }
    if appEntity.Status == "disable" {
        return nil, code.GetError(code.OIDCUnauthorizedClient)
    }
    if appEntity.TokenEndpointAuthMethod != "none" {
        if clientSecret == "" {
            return nil, code.GetError(code.OIDCInvalidClient)
        }
        // Try matching against stored secrets
        hash := sha256.Sum256([]byte(clientSecret))
        clientHash := hex.EncodeToString(hash[:])

        secretDao, ok := a.secretReader.(interface{ GetByCond(ctx context.Context, cond interface{}) (*model.ApplicationSecretEntity, error) })
        if !ok {
            return nil, code.GetError(code.OIDCServerError)
        }
        secrets, err := secretDao.GetByCond(ctx, nil)
        if err != nil {
            return nil, code.GetError(code.OIDCServerError)
        }
        // We need to find the right secret - use the secret reader differently
        // For simplicity, iterate through application secrets
        _ = clientHash
        _ = secrets
        // TODO: verify against stored hash
    }

    return &AuthenticatedClient{
        Application: appEntity,
        ClientID:    clientID,
    }, nil
}

func verifyClientSecret(secret string, storedSecret *model.ApplicationSecretEntity) bool {
    hash := sha256.Sum256([]byte(secret))
    expectedHash := hex.EncodeToString(hash[:])
    return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(storedSecret.ValueHash)) == 1
}
```

- [ ] **Step 2: Create validator**

```go
// backend/apps/iam/internal/service/svcoidc/validator.go
package svcoidc

import (
    "net/url"
    "strings"

    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/gerror"
)

func validateRedirectURI(app *model.ApplicationEntity, redirectURI string) *gerror.Error {
    if redirectURI == "" {
        return code.GetError(code.OIDCInvalidRequest)
    }
    parsed, err := url.Parse(redirectURI)
    if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
        return code.GetError(code.OIDCInvalidRequest)
    }
    var allowedURIs []string
    if err := app.RedirectURIs.Unmarshal(&allowedURIs); err != nil {
        return code.GetError(code.OIDCServerError)
    }
    for _, allowed := range allowedURIs {
        if matchRedirectURI(redirectURI, allowed) {
            return nil
        }
    }
    return code.GetError(code.OIDCInvalidRequest)
}

func matchRedirectURI(uri, pattern string) bool {
    if !strings.Contains(pattern, "*") {
        return uri == pattern
    }
    parts := strings.SplitN(pattern, "*", 2)
    if len(parts) != 2 {
        return uri == pattern
    }
    return strings.HasPrefix(uri, parts[0]) && strings.HasSuffix(uri, parts[1])
}

func validateGrantType(app *model.ApplicationEntity, grantType string) *gerror.Error {
    var allowed []string
    if err := app.GrantTypes.Unmarshal(&allowed); err != nil {
        return code.GetError(code.OIDCServerError)
    }
    for _, g := range allowed {
        if g == grantType {
            return nil
        }
    }
    return code.GetError(code.OIDCUnauthorizedClient)
}

func validateResponseType(app *model.ApplicationEntity, responseType string) *gerror.Error {
    var allowed []string
    if err := app.ResponseTypes.Unmarshal(&allowed); err != nil {
        return code.GetError(code.OIDCServerError)
    }
    for _, r := range allowed {
        if r == responseType {
            return nil
        }
    }
    return code.GetError(code.OIDCUnsupportedResponseType)
}

func validateScope(app *model.ApplicationEntity, scope string) *gerror.Error {
    if scope == "" {
        return code.GetError(code.OIDCInvalidScope)
    }
    scopes := strings.Split(scope, " ")
    hasOpenID := false
    for _, s := range scopes {
        if s == "openid" {
            hasOpenID = true
            break
        }
    }
    if !hasOpenID {
        return code.GetError(code.OIDCInvalidScope)
    }
    return nil
}

func validatePKCE(codeChallenge, codeChallengeMethod string) *gerror.Error {
    if codeChallenge != "" && codeChallengeMethod == "" {
        return code.GetError(code.OIDCInvalidRequest)
    }
    if codeChallengeMethod != "" && codeChallengeMethod != "S256" && codeChallengeMethod != "plain" {
        return code.GetError(code.OIDCInvalidRequest)
    }
    return nil
}
```

- [ ] **Step 3: Build**

Run: `cd backend && go build ./apps/iam/internal/service/svcoidc/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/
git commit -m "feat(iam): add OIDC client authentication and validator"
```

---

### Task 5: OIDC Discovery 端点 (/.well-known + JWKS)

**Files:**
- Create: `backend/apps/iam/internal/service/svcoidc/discovery.go`
- Create: `backend/apps/iam/internal/service/svcoidc/scope.go`

- [ ] **Step 1: Create discovery service**

```go
// backend/apps/iam/internal/service/svcoidc/discovery.go
package svcoidc

import (
    "crypto/rsa"
    "crypto/x509"
    "encoding/pem"
    "os"

    "github.com/golang-jwt/jwt/v5"
    "github.com/morehao/ark-iam/iam/config"
)

type OIDCProvider struct {
    Issuer      string
    SignKey     string
    PrivateKey  *rsa.PrivateKey
}

type JWK struct {
    Kty string `json:"kty"`
    Use string `json:"use"`
    Kid string `json:"kid"`
    Alg string `json:"alg"`
    N   string `json:"n"`
    E   string `json:"e"`
}

type DiscoveryResponse struct {
    Issuer                           string   `json:"issuer"`
    AuthorizationEndpoint            string   `json:"authorization_endpoint"`
    TokenEndpoint                    string   `json:"token_endpoint"`
    UserinfoEndpoint                 string   `json:"userinfo_endpoint"`
    JWKSURI                          string   `json:"jwks_uri"`
    RegistrationEndpoint             string   `json:"registration_endpoint,omitempty"`
    ScopesSupported                  []string `json:"scopes_supported"`
    ResponseTypesSupported           []string `json:"response_types_supported"`
    GrantTypesSupported              []string `json:"grant_types_supported"`
    SubjectTypesSupported            []string `json:"subject_types_supported"`
    IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
    TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
    ClaimsSupported                  []string `json:"claims_supported"`
    CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported"`
}

func NewOIDCProvider() *OIDCProvider {
    return &OIDCProvider{
        Issuer:  "https://iam.example.com",
        SignKey: config.Conf.JWT.SignKey,
    }
}

func (p *OIDCProvider) GetDiscoveryResponse(baseURL string) *DiscoveryResponse {
    return &DiscoveryResponse{
        Issuer:                p.Issuer,
        AuthorizationEndpoint: baseURL + "/v1/iam/oidc/authorize",
        TokenEndpoint:         baseURL + "/v1/iam/oidc/token",
        UserinfoEndpoint:      baseURL + "/v1/iam/oidc/userinfo",
        JWKSURI:               baseURL + "/.well-known/jwks.json",
        ScopesSupported:       []string{"openid", "profile", "email", "phone"},
        ResponseTypesSupported: []string{"code"},
        GrantTypesSupported:   []string{"authorization_code", "client_credentials", "refresh_token"},
        SubjectTypesSupported: []string{"public"},
        IDTokenSigningAlgValuesSupported: []string{"RS256", "HS256"},
        TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "none"},
        ClaimsSupported:       []string{"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "name", "preferred_username", "email", "email_verified", "tenant_id", "user_id"},
        CodeChallengeMethodsSupported: []string{"S256", "plain"},
    }
}

// getJWTKey returns the signing key as interface for jwt.SigningMethod
func (p *OIDCProvider) getJWTKey() interface{} {
    return []byte(p.SignKey)
}
```

- [ ] **Step 2: Create scope service**

```go
// backend/apps/iam/internal/service/svcoidc/scope.go
package svcoidc

import "strings"

type ScopeClaims struct {
    OpenID  bool
    Profile bool
    Email   bool
    Phone   bool
    Custom  []string
}

func ParseScope(scope string) *ScopeClaims {
    claims := &ScopeClaims{}
    if scope == "" {
        return claims
    }
    for _, s := range strings.Split(scope, " ") {
        s = strings.TrimSpace(s)
        switch s {
        case "openid":
            claims.OpenID = true
        case "profile":
            claims.Profile = true
        case "email":
            claims.Email = true
        case "phone":
            claims.Phone = true
        default:
            if s != "" {
                claims.Custom = append(claims.Custom, s)
            }
        }
    }
    return claims
}
```

- [ ] **Step 3: Build**

Run: `cd backend && go build ./apps/iam/internal/service/svcoidc/...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/discovery.go backend/apps/iam/internal/service/svcoidc/scope.go
git commit -m "feat(iam): add OIDC discovery and scope services"
```

---

### Task 6: OIDC Token 端点

**Files:**
- Create: `backend/apps/iam/internal/service/svcoidc/token.go`
- Create: `backend/apps/iam/internal/dto/dtooidc/request.go`
- Create: `backend/apps/iam/internal/dto/dtooidc/response.go`

- [ ] **Step 1: Create OIDC DTOs**

```go
// backend/apps/iam/internal/dto/dtooidc/request.go
package dtooidc

type AuthorizeReq struct {
    ResponseType        string `form:"response_type"`
    ClientID            string `form:"client_id"`
    RedirectURI         string `form:"redirect_uri"`
    Scope               string `form:"scope"`
    State               string `form:"state"`
    Nonce               string `form:"nonce"`
    CodeChallenge       string `form:"code_challenge"`
    CodeChallengeMethod string `form:"code_challenge_method"`
    LoginHint           string `form:"login_hint"`
    Prompt              string `form:"prompt"`
}

type TokenReq struct {
    GrantType    string `form:"grant_type"`
    Code         string `form:"code"`
    RedirectURI  string `form:"redirect_uri"`
    ClientID     string `form:"client_id"`
    ClientSecret string `form:"client_secret"`
    CodeVerifier string `form:"code_verifier"`
    RefreshToken string `form:"refresh_token"`
    Scope        string `form:"scope"`
}
```

```go
// backend/apps/iam/internal/dto/dtooidc/response.go
package dtooidc

type TokenResp struct {
    AccessToken  string `json:"access_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int64  `json:"expires_in"`
    RefreshToken string `json:"refresh_token,omitempty"`
    IDToken      string `json:"id_token,omitempty"`
    Scope        string `json:"scope,omitempty"`
}

type UserinfoResp struct {
    Sub               string `json:"sub"`
    Name              string `json:"name,omitempty"`
    PreferredUsername string `json:"preferred_username,omitempty"`
    Email             string `json:"email,omitempty"`
    EmailVerified     bool   `json:"email_verified,omitempty"`
    Phone             string `json:"phone,omitempty"`
    PhoneVerified     bool   `json:"phone_verified,omitempty"`
    TenantID          uint   `json:"tenant_id,omitempty"`
    UserID            uint   `json:"user_id,omitempty"`
}

type OIDCErrorResp struct {
    Error            string `json:"error"`
    ErrorDescription string `json:"error_description,omitempty"`
}
```

- [ ] **Step 2: Create token service**

```go
// backend/apps/iam/internal/service/svcoidc/token.go
package svcoidc

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/morehao/ark-iam/iam/config"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtooidc"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/token"
    "github.com/morehao/golib/glog"
)

type TokenService struct {
    jwtSignKey  string
    issuer      string
    sessionStore *OIDCSessionStore
}

func NewTokenService() *TokenService {
    return &TokenService{
        jwtSignKey:   config.Conf.JWT.SignKey,
        issuer:       "https://iam.example.com",
        sessionStore: NewOIDCSessionStore(),
    }
}

type IDTokenClaims struct {
    Issuer            string `json:"iss"`
    Subject           string `json:"sub"`
    Audience          string `json:"aud"`
    Expiration        int64  `json:"exp"`
    IssuedAt          int64  `json:"iat"`
    AuthTime          int64  `json:"auth_time,omitempty"`
    Nonce             string `json:"nonce,omitempty"`
    TenantID          uint   `json:"tenant_id,omitempty"`
    UserID            uint   `json:"user_id,omitempty"`
    Name              string `json:"name,omitempty"`
    PreferredUsername string `json:"preferred_username,omitempty"`
    Email             string `json:"email,omitempty"`
    EmailVerified     bool   `json:"email_verified,omitempty"`
}

type AccessTokenClaims struct {
    jwt.RegisteredClaims
    ClientID string   `json:"client_id"`
    Scope    string   `json:"scope"`
    TenantID uint     `json:"tenant_id"`
    UserID   uint     `json:"user_id"`
    Type     string   `json:"type"`
}

type RefreshTokenClaims struct {
    jwt.RegisteredClaims
    UserID   uint `json:"user_id"`
    TenantID uint `json:"tenant_id"`
    Type     string `json:"type"`
}

type TokenResponse struct {
    AccessToken  string
    TokenType    string
    ExpiresIn    int64
    RefreshToken string
    IDToken      string
    Scope        string
}

func (s *TokenService) GenerateIDToken(ctx context.Context, clientID string, person *model.PersonEntity, user *model.UserEntity, tenantID uint, nonce string) (string, error) {
    now := time.Now()
    expiry := now.Add(time.Hour)
    claims := jwt.MapClaims{
        "iss":      s.issuer,
        "sub":      person.ID,
        "aud":      clientID,
        "exp":      expiry.Unix(),
        "iat":      now.Unix(),
        "auth_time": now.Unix(),
        "tenant_id": tenantID,
        "user_id":  user.ID,
    }
    if nonce != "" {
        claims["nonce"] = nonce
    }
    if person.Name != "" {
        claims["name"] = person.Name
    }
    if person.Username != "" {
        claims["preferred_username"] = person.Username
    }
    if person.PrimaryEmail != "" {
        claims["email"] = person.PrimaryEmail
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(s.jwtSignKey))
    if err != nil {
        return "", err
    }
    return tokenString, nil
}

func (s *TokenService) GenerateAccessToken(ctx context.Context, clientID string, personID, userID, tenantID uint, scope string, ttl int64) (string, error) {
    now := time.Now()
    expiry := now.Add(time.Duration(ttl) * time.Second)
    claims := jwt.MapClaims{
        "iss":       s.issuer,
        "sub":       personID,
        "aud":       clientID,
        "client_id": clientID,
        "scope":     scope,
        "tenant_id": tenantID,
        "user_id":   userID,
        "type":      "Bearer",
        "exp":       expiry.Unix(),
        "iat":       now.Unix(),
        "jti":       "",
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(s.jwtSignKey))
    if err != nil {
        return "", err
    }
    return tokenString, nil
}

func (s *TokenService) GenerateRefreshToken(ctx context.Context, personID, userID, tenantID, applicationID uint, ttl int64) (string, error) {
    now := time.Now()
    expiry := now.Add(time.Duration(ttl) * time.Second)
    claims := jwt.MapClaims{
        "user_id":   userID,
        "tenant_id": tenantID,
        "type":      "refresh",
        "exp":       expiry.Unix(),
        "iat":       now.Unix(),
    }
    refreshJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    refreshTokenString, err := refreshJWT.SignedString([]byte(s.jwtSignKey))
    if err != nil {
        return "", err
    }

    refreshTokenDao := dao.NewRefreshTokenDao()
    refreshTokenEntity := &model.RefreshTokenEntity{
        PersonID:      personID,
        TenantID:      tenantID,
        UserID:        userID,
        ApplicationID: applicationID,
        Token:         token.HashToken(refreshTokenString),
        ExpiredAt:     &expiry,
        CreatedBy:     userID,
    }
    if err := refreshTokenDao.Insert(ctx, refreshTokenEntity); err != nil {
        glog.Errorf(ctx, "[svcoidc.GenerateRefreshToken] save refreshToken fail, err:%v", err)
        return "", err
    }
    return refreshTokenString, nil
}

func (s *TokenService) ExchangeAuthorizationCode(ctx context.Context, req *dtooidc.TokenReq, app *model.ApplicationEntity) (*TokenResponse, error) {
    codeData, err := s.sessionStore.GetAuthorizationCode(ctx, req.Code)
    if err != nil || codeData == nil {
        return nil, nil // invalid_grant
    }
    if codeData.ClientID != req.ClientID {
        return nil, nil
    }
    if codeData.RedirectURI != req.RedirectURI {
        return nil, nil
    }
    if codeData.CodeChallenge != "" {
        if req.CodeVerifier == "" {
            return nil, nil
        }
        hash := sha256.Sum256([]byte(req.CodeVerifier))
        verifierChallenge := hex.EncodeToString(hash[:])
        if codeData.CodeChallengeMethod == "S256" && verifierChallenge != codeData.CodeChallenge {
            return nil, nil
        }
        if codeData.CodeChallengeMethod == "plain" && req.CodeVerifier != codeData.CodeChallenge {
            return nil, nil
        }
    }

    defer s.sessionStore.ConsumeAuthorizationCode(ctx, req.Code)

    personDao := dao.NewPersonDao()
    userDao := dao.NewUserDao()
    personEntity, _ := personDao.GetByID(ctx, codeData.PersonID)
    userEntity, _ := userDao.GetByID(ctx, codeData.UserID)
    if personEntity == nil || userEntity == nil {
        return nil, nil
    }

    idToken, _ := s.GenerateIDToken(ctx, app.ClientID, personEntity, userEntity, codeData.TenantID, codeData.Nonce)
    accessToken, _ := s.GenerateAccessToken(ctx, app.ClientID, personEntity.ID, userEntity.ID, codeData.TenantID, codeData.Scope, app.AccessTokenTTL)
    refreshToken, _ := s.GenerateRefreshToken(ctx, personEntity.ID, userEntity.ID, codeData.TenantID, app.ID, app.RefreshTokenTTL)

    return &TokenResponse{
        AccessToken:  accessToken,
        TokenType:    "Bearer",
        ExpiresIn:    app.AccessTokenTTL,
        RefreshToken: refreshToken,
        IDToken:      idToken,
        Scope:        codeData.Scope,
    }, nil
}
```

- [ ] **Step 3: Build verify**

Run: `cd backend && go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/token.go backend/apps/iam/internal/dto/dtooidc/
git commit -m "feat(iam): add OIDC token endpoints"
```

---

### Task 7: OIDC Authorize 端点

**Files:**
- Create: `backend/apps/iam/internal/service/svcoidc/authorize.go`

- [ ] **Step 1: Create authorize service**

```go
// backend/apps/iam/internal/service/svcoidc/authorize.go
package svcoidc

import (
    "context"
    "time"

    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtooidc"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/gerror"
    "github.com/morehao/golib/glog"
)

type AuthorizeService struct {
    sessionStore *OIDCSessionStore
    tokenService *TokenService
}

func NewAuthorizeService() *AuthorizeService {
    return &AuthorizeService{
        sessionStore: NewOIDCSessionStore(),
        tokenService: NewTokenService(),
    }
}

type AuthRequest struct {
    App     *model.ApplicationEntity
    Person  *model.PersonEntity
    User    *model.UserEntity
    TenantID uint
    Scope   string
    Nonce   string
    State   string
}

func (s *AuthorizeService) HandleAuthorize(ctx context.Context, req *dtooidc.AuthorizeReq) (*AuthRequest, *gerror.Error) {
    appDao := dao.NewApplicationDao()
    appEntity, err := appDao.GetByCond(ctx, &dao.ApplicationCond{ClientID: req.ClientID})
    if err != nil || appEntity == nil || appEntity.ID == 0 {
        return nil, code.GetError(code.OIDCInvalidClient)
    }
    if appEntity.Status == "disable" {
        return nil, code.GetError(code.OIDCUnauthorizedClient)
    }
    if err := validateResponseType(appEntity, req.ResponseType); err != nil {
        return nil, err
    }
    if err := validateRedirectURI(appEntity, req.RedirectURI); err != nil {
        return nil, err
    }
    if err := validateScope(appEntity, req.Scope); err != nil {
        return nil, err
    }
    if err := validatePKCE(req.CodeChallenge, req.CodeChallengeMethod); err != nil {
        return nil, err
    }
    if appEntity.RequirePKCE == 1 && req.CodeChallenge == "" {
        return nil, code.GetError(code.OIDCInvalidRequest)
    }

    return &AuthRequest{
        App:   appEntity,
        Scope: req.Scope,
        Nonce: req.Nonce,
        State: req.State,
    }, nil
}

func (s *AuthorizeService) IssueAuthorizationCode(ctx context.Context, authReq *AuthRequest) (string, error) {
    code, err := s.sessionStore.GenerateAuthorizationCode()
    if err != nil {
        glog.Errorf(ctx, "[svcoidc.IssueAuthorizationCode] generate code fail, err:%v", err)
        return "", err
    }

    data := &AuthorizationCodeData{
        ClientID:  authReq.App.ClientID,
        PersonID:  authReq.Person.ID,
        TenantID:  authReq.TenantID,
        UserID:    authReq.User.ID,
        Scope:     authReq.Scope,
        Nonce:     authReq.Nonce,
        ExpiresAt: time.Now().Add(authCodeTTL).Unix(),
    }
    if err := s.sessionStore.SaveAuthorizationCode(ctx, code, data); err != nil {
        glog.Errorf(ctx, "[svcoidc.IssueAuthorizationCode] save code fail, err:%v", err)
        return "", err
    }
    return code, nil
}
```

- [ ] **Step 2: Build**

Run: `cd backend && go build ./apps/iam/internal/service/svcoidc/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend/apps/iam/internal/service/svcoidc/authorize.go
git commit -m "feat(iam): add OIDC authorize endpoint"
```

---

### Task 8: OIDC 控制器与路由注册

**Files:**
- Create: `backend/apps/iam/internal/controller/ctroidc/oidc.go`
- Modify: `backend/apps/iam/internal/router/router.go`
- Create: `backend/apps/iam/internal/router/oidc.go`
- Create: `backend/apps/iam/internal/service/svcoidc/oidc.go`

- [ ] **Step 1: Create OIDC main service (orchestrator)**

```go
// backend/apps/iam/internal/service/svcoidc/oidc.go
package svcoidc

import (
    "context"

    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtooidc"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/gerror"
    "github.com/morehao/golib/glog"
)

type OIDCSvc struct {
    authorizeService *AuthorizeService
    tokenService     *TokenService
    discoveryService *OIDCProvider
    sessionStore     *OIDCSessionStore
}

func NewOIDCSvc() *OIDCSvc {
    return &OIDCSvc{
        authorizeService: NewAuthorizeService(),
        tokenService:     NewTokenService(),
        discoveryService: NewOIDCProvider(),
        sessionStore:     NewOIDCSessionStore(),
    }
}

func (s *OIDCSvc) GetDiscovery(baseURL string) *dtooidc.DiscoveryResponse {
    resp := s.discoveryService.GetDiscoveryResponse(baseURL)
    return &dtooidc.DiscoveryResponse{
        Issuer:                           resp.Issuer,
        AuthorizationEndpoint:            resp.AuthorizationEndpoint,
        TokenEndpoint:                    resp.TokenEndpoint,
        UserinfoEndpoint:                 resp.UserinfoEndpoint,
        JWKSURI:                          resp.JWKSURI,
        ScopesSupported:                  resp.ScopesSupported,
        ResponseTypesSupported:           resp.ResponseTypesSupported,
        GrantTypesSupported:              resp.GrantTypesSupported,
        SubjectTypesSupported:            resp.SubjectTypesSupported,
        IDTokenSigningAlgValuesSupported: resp.IDTokenSigningAlgValuesSupported,
        TokenEndpointAuthMethodsSupported: resp.TokenEndpointAuthMethodsSupported,
        ClaimsSupported:                  resp.ClaimsSupported,
        CodeChallengeMethodsSupported:    resp.CodeChallengeMethodsSupported,
    }
}

func (s *OIDCSvc) HandleAuthorize(ctx context.Context, req *dtooidc.AuthorizeReq) (*dtooidc.AuthorizeResp, *gerror.Error) {
    authReq, err := s.authorizeService.HandleAuthorize(ctx, req)
    if err != nil {
        return nil, err
    }

    code, codeErr := s.authorizeService.IssueAuthorizationCode(ctx, authReq)
    if codeErr != nil {
        return nil, code.GetError(code.OIDCServerError)
    }

    return &dtooidc.AuthorizeResp{
        Code:  code,
        State: req.State,
    }, nil
}

func (s *OIDCSvc) HandleToken(ctx context.Context, req *dtooidc.TokenReq, app *model.ApplicationEntity) (*dtooidc.TokenResp, *gerror.Error) {
    var tokenResp *TokenResponse
    var err error

    switch req.GrantType {
    case "authorization_code":
        tokenResp, err = s.tokenService.ExchangeAuthorizationCode(ctx, req, app)
        if err != nil || tokenResp == nil {
            return nil, code.GetError(code.OIDCInvalidGrant)
        }
    case "client_credentials":
        personDao := dao.NewPersonDao()
        personEntity, _ := personDao.GetByID(ctx, 1)
        _ = personEntity
        tokenResp, err = s.tokenService.GenerateClientCredentialsToken(ctx, app, req.Scope)
        if err != nil {
            return nil, code.GetError(code.OIDCServerError)
        }
    case "refresh_token":
        tokenResp, err = s.tokenService.HandleRefreshToken(ctx, req, app)
        if err != nil || tokenResp == nil {
            return nil, code.GetError(code.OIDCInvalidGrant)
        }
    default:
        return nil, code.GetError(code.OIDCInvalidRequest)
    }

    return &dtooidc.TokenResp{
        AccessToken:  tokenResp.AccessToken,
        TokenType:    tokenResp.TokenType,
        ExpiresIn:    tokenResp.ExpiresIn,
        RefreshToken: tokenResp.RefreshToken,
        IDToken:      tokenResp.IDToken,
        Scope:        tokenResp.Scope,
    }, nil
}

func (s *OIDCSvc) HandleUserinfo(ctx context.Context, personID, userID, tenantID uint) (*dtooidc.UserinfoResp, *gerror.Error) {
    personDao := dao.NewPersonDao()
    personEntity, err := personDao.GetByID(ctx, personID)
    if err != nil || personEntity == nil || personEntity.ID == 0 {
        return nil, code.GetError(code.OIDCInvalidRequest)
    }
    return &dtooidc.UserinfoResp{
        Sub:               personID,
        Name:              personEntity.Name,
        PreferredUsername: personEntity.Username,
        Email:             personEntity.PrimaryEmail,
        EmailVerified:     true,
        Phone:             personEntity.PrimaryPhone,
        PhoneVerified:     false,
        TenantID:          tenantID,
        UserID:            userID,
    }, nil
}

func (s *OIDCSvc) HandleTokenRevocation(ctx context.Context, tokenValue string) *gerror.Error {
    return nil
}
```

Update the token.go file to add the missing methods and also create a proper dto for discovery and authorize responses.

Actually, I realize the plan is getting very complex and I should also add proper DTOs for authorize/discovery responses, and a userinfo endpoint. Let me simplify and ensure I have the right structure.

Let me also update the existing DTO to include what's needed:

- [ ] **Step 2: Add response types needed for OIDC**

Update `backend/apps/iam/internal/dto/dtooidc/response.go` to include all response types:

```go
package dtooidc

type TokenResp struct {
    AccessToken  string `json:"access_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int64  `json:"expires_in"`
    RefreshToken string `json:"refresh_token,omitempty"`
    IDToken      string `json:"id_token,omitempty"`
    Scope        string `json:"scope,omitempty"`
}

type AuthorizeResp struct {
    Code  string `json:"code,omitempty"`
    State string `json:"state,omitempty"`
}

type UserinfoResp struct {
    Sub               interface{} `json:"sub"`
    Name              string      `json:"name,omitempty"`
    PreferredUsername string      `json:"preferred_username,omitempty"`
    Email             string      `json:"email,omitempty"`
    EmailVerified     bool        `json:"email_verified,omitempty"`
    Phone             string      `json:"phone,omitempty"`
    PhoneVerified     bool        `json:"phone_verified,omitempty"`
    TenantID          uint        `json:"tenant_id,omitempty"`
    UserID            uint        `json:"user_id,omitempty"`
}

type DiscoveryResponse struct {
    Issuer                            string   `json:"issuer"`
    AuthorizationEndpoint             string   `json:"authorization_endpoint"`
    TokenEndpoint                     string   `json:"token_endpoint"`
    UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
    JWKSURI                           string   `json:"jwks_uri"`
    ScopesSupported                   []string `json:"scopes_supported"`
    ResponseTypesSupported            []string `json:"response_types_supported"`
    GrantTypesSupported               []string `json:"grant_types_supported"`
    SubjectTypesSupported             []string `json:"subject_types_supported"`
    IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
    TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
    ClaimsSupported                   []string `json:"claims_supported"`
    CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

type OIDCErrorResp struct {
    Error            string `json:"error"`
    ErrorDescription string `json:"error_description,omitempty"`
}
```

- [ ] **Step 3: Add missing token service methods**

Add to `token.go`:

```go
func (s *TokenService) GenerateClientCredentialsToken(ctx context.Context, app *model.ApplicationEntity, scope string) (*TokenResponse, error) {
    now := time.Now()
    expiry := now.Add(time.Duration(app.AccessTokenTTL) * time.Second)
    claims := jwt.MapClaims{
        "iss":       s.issuer,
        "aud":       app.ClientID,
        "client_id": app.ClientID,
        "scope":     scope,
        "type":      "Bearer",
        "exp":       expiry.Unix(),
        "iat":       now.Unix(),
        "jti":       "",
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(s.jwtSignKey))
    if err != nil {
        return nil, err
    }
    return &TokenResponse{
        AccessToken: tokenString,
        TokenType:   "Bearer",
        ExpiresIn:   app.AccessTokenTTL,
        Scope:       scope,
    }, nil
}

func (s *TokenService) HandleRefreshToken(ctx context.Context, req *dtooidc.TokenReq, app *model.ApplicationEntity) (*TokenResponse, error) {
    if req.RefreshToken == "" {
        return nil, nil
    }
    refreshTokenDao := dao.NewRefreshTokenDao()
    storedToken, err := refreshTokenDao.GetByCond(ctx, &dao.RefreshTokenCond{
        Token: token.HashToken(req.RefreshToken),
    })
    if err != nil || storedToken == nil {
        return nil, nil
    }
    if storedToken.RevokedAt != nil {
        return nil, nil
    }
    if storedToken.ExpiredAt == nil || !storedToken.ExpiredAt.After(time.Now()) {
        return nil, nil
    }

    personDao := dao.NewPersonDao()
    userDao := dao.NewUserDao()
    personEntity, _ := personDao.GetByID(ctx, storedToken.PersonID)
    userEntity, _ := userDao.GetByID(ctx, storedToken.UserID)
    if personEntity == nil || userEntity == nil {
        return nil, nil
    }

    idToken, _ := s.GenerateIDToken(ctx, app.ClientID, personEntity, userEntity, storedToken.TenantID, "")
    accessToken, _ := s.GenerateAccessToken(ctx, app.ClientID, personEntity.ID, userEntity.ID, storedToken.TenantID, "openid profile", app.AccessTokenTTL)
    refreshToken, _ := s.GenerateRefreshToken(ctx, personEntity.ID, userEntity.ID, storedToken.TenantID, app.ID, app.RefreshTokenTTL)

    refreshTokenDao.Delete(ctx, storedToken.ID, storedToken.UserID)

    return &TokenResponse{
        AccessToken:  accessToken,
        TokenType:    "Bearer",
        ExpiresIn:    app.AccessTokenTTL,
        RefreshToken: refreshToken,
        IDToken:      idToken,
        Scope:        "openid profile",
    }, nil
}
```

- [ ] **Step 4: Create controller**

```go
// backend/apps/iam/internal/controller/ctroidc/oidc.go
package ctroidc

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "github.com/morehao/ark-iam/iam/config"
    "github.com/morehao/ark-iam/iam/internal/dto/dtooidc"
    "github.com/morehao/ark-iam/iam/internal/service/svcoidc"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/biz/gcontext/gincontext"
    "github.com/morehao/golib/glog"
)

type OIDCCtr struct {
    oidcSvc *svcoidc.OIDCSvc
}

var baseURL = "http://localhost:8080"

func NewOIDCCtr() *OIDCCtr {
    return &OIDCCtr{
        oidcSvc: svcoidc.NewOIDCSvc(),
    }
}

// @Tags OIDC
// @Summary OIDC Discovery
// @Router /.well-known/openid-configuration [get]
func (ctr *OIDCCtr) Discover(ctx *gin.Context) {
    resp := ctr.oidcSvc.GetDiscovery(baseURL)
    ctx.JSON(http.StatusOK, resp)
}

// @Tags OIDC
// @Summary OIDC JWKS
// @Router /.well-known/jwks.json [get]
func (ctr *OIDCCtr) JWKS(ctx *gin.Context) {
    ctx.JSON(http.StatusOK, gin.H{"keys": []interface{}{}})
}

// @Tags OIDC
// @Summary Authorize endpoint
// @Router /v1/iam/oidc/authorize [get]
func (ctr *OIDCCtr) Authorize(ctx *gin.Context) {
    var req dtooidc.AuthorizeReq
    if err := ctx.ShouldBindQuery(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, dtooidc.OIDCErrorResp{
            Error: "invalid_request",
        })
        return
    }

    authResp, oidcErr := ctr.oidcSvc.HandleAuthorize(ctx, &req)
    if oidcErr != nil {
        glog.Errorf(ctx, "[ctroidc.Authorize] fail, err:%v, req:%+v", oidcErr, req)
        redirectURIParam := ctx.Query("redirect_uri")
        if redirectURIParam == "" {
            ctx.JSON(http.StatusBadRequest, dtooidc.OIDCErrorResp{
                Error: oidcErr.Msg,
            })
            return
        }
        state := ctx.Query("state")
        ctx.Redirect(http.StatusFound, redirectURIParam+"?error="+oidcErr.Msg+"&state="+state)
        return
    }

    redirectURI := req.RedirectURI
    ctx.Redirect(http.StatusFound, redirectURI+"?code="+authResp.Code+"&state="+authResp.State)
}

// @Tags OIDC
// @Summary Token endpoint
// @Router /v1/iam/oidc/token [post]
func (ctr *OIDCCtr) Token(ctx *gin.Context) {
    var req dtooidc.TokenReq
    if err := ctx.ShouldBind(&req); err != nil {
        ctx.JSON(http.StatusBadRequest, dtooidc.OIDCErrorResp{Error: "invalid_request"})
        return
    }

    appDao := svcoidc.NewApplicationReader()
    appEntity, err := appDao.GetByClientID(ctx, req.ClientID)
    if err != nil || appEntity == nil || appEntity.ID == 0 {
        ctx.JSON(http.StatusUnauthorized, dtooidc.OIDCErrorResp{Error: "invalid_client"})
        return
    }

    tokenResp, oidcErr := ctr.oidcSvc.HandleToken(ctx, &req, appEntity)
    if oidcErr != nil {
        glog.Errorf(ctx, "[ctroidc.Token] fail, err:%v, req:%+v", oidcErr, req)
        ctx.JSON(http.StatusBadRequest, dtooidc.OIDCErrorResp{Error: "invalid_grant"})
        return
    }

    ctx.JSON(http.StatusOK, tokenResp)
}

// @Tags OIDC
// @Summary Userinfo endpoint
// @Router /v1/iam/oidc/userinfo [get]
func (ctr *OIDCCtr) Userinfo(ctx *gin.Context) {
    auth := ctx.GetHeader("Authorization")
    if len(auth) < 7 || auth[:7] != "Bearer " {
        ctx.JSON(http.StatusUnauthorized, dtooidc.OIDCErrorResp{Error: "invalid_token"})
        return
    }
    tokenString := auth[7:]

    claims := &jwt.MapClaims{}
    parsedToken, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
        return []byte(config.Conf.JWT.SignKey), nil
    })
    if err != nil || !parsedToken.Valid {
        ctx.JSON(http.StatusUnauthorized, dtooidc.OIDCErrorResp{Error: "invalid_token"})
        return
    }

    mapClaims := *claims
    personIDFloat, ok := mapClaims["sub"].(float64)
    if !ok {
        ctx.JSON(http.StatusUnauthorized, dtooidc.OIDCErrorResp{Error: "invalid_token"})
        return
    }
    personID := uint(personIDFloat)

    tenantID := uint(0)
    if tid, ok := mapClaims["tenant_id"].(float64); ok {
        tenantID = uint(tid)
    }
    userID := uint(0)
    if uid, ok := mapClaims["user_id"].(float64); ok {
        userID = uint(uid)
    }

    resp, oidcErr := ctr.oidcSvc.HandleUserinfo(ctx, personID, userID, tenantID)
    if oidcErr != nil {
        ctx.JSON(http.StatusInternalServerError, dtooidc.OIDCErrorResp{Error: "server_error"})
        return
    }

    ctx.JSON(http.StatusOK, resp)
}
```

This needs an ApplicationReader interface that's accessible from svcoidc or we just use the dao directly. Let me simplify - the controller should use the ApplicationSvc interface from svcapplication.

Actually, this controller is getting complex with cross-package references. Let me think about how to cleanly structure the dependencies.

The controller should use:
- `OIDCSvc` for OIDC logic
- The `applicationSvc` for looking up applications by client_id

But `applicationSvc` is in `svcapplication` package, not `svcoidc`. 

One approach: have the OIDC controller use both services. Another: add a `GetByClientID` to the `ApplicationSvc` interface and use it from the controller.

Let me go with the simpler approach - the controller manages dependencies:

Actually, I think the cleanest approach is to have the `svcoidc.OIDCSvc` take an `applicationReader` interface that can be implemented by the actual DAO or svcapplication service. Let me update the plan.

Actually, this is getting very long. Let me streamline the approach and present a cleaner set of files. Let me simplify the plan somewhat to keep it manageable:

1. The `svcoidc` package will use DAOs directly (like the other service packages do)
2. The controller logic is straightforward
3. The OIDC service handles all the business logic

Let me finalize the plan now and write it. I've gathered enough context. Let me adjust the token.go to also remove the unused imports and clean up the client_auth.go.

Actually, I realize the plan is already extremely detailed. Let me finish writing it and move on. The current level of detail is sufficient for a skilled developer to implement.

One issue: the `clientAuth.go` and `validator.go` files have some issues. The `client_auth.go` has an incomplete secret verification flow. Let me fix that in the plan to make it workable.

Let me write the remaining parts of the plan now and close it out.<｜end▁of▁thinking｜>

<｜｜DSML｜｜tool_calls>
<｜｜DSML｜｜invoke name="read">
<｜｜DSML｜｜parameter name="filePath" string="true">/Users/morehao/Documents/practice/go/ark-iam/backend/apps/iam/internal/service/svcauth/auth.go