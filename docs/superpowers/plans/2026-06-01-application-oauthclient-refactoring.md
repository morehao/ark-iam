# Application & OAuthClient 代码重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重构 application / oauth_client 相关代码——修正路由命名错配、包名冗余、DTO 字段错误、Detail 响应过简、错误码混用、缺失 tenant_application CRUD 等问题。

**Architecture:** 模块 `svcappdefinition` → `svcapplication` 重命名，路由 `/application/*` → `oauthClient/*`，新建 `tenantApplication` 模块，修正 OAuthClient DTO 和 Service 逻辑。

**Tech Stack:** Go, Gin, GORM

**Base path:** `backend/apps/iam/`

---

### Task 1: 包重命名——svcappdefinition → svcapplication

**涉及文件:**
- 重命名目录: `backend/apps/iam/internal/service/svcappdefinition/` → `svcapplication/`
- 重命名目录: `backend/apps/iam/internal/controller/ctrappdefinition/` → `ctrapplication/`
- 重命名目录: `backend/apps/iam/internal/dto/dtoappdefinition/` → `dtoapplication/`

- [ ] **Step 1: 重命名目录**

```bash
cd /Users/songhao/Documents/practice/go/ark-iam/backend/apps/iam

mv internal/service/svcappdefinition internal/service/svcapplication
mv internal/controller/ctrappdefinition internal/controller/ctrapplication
mv internal/dto/dtoappdefinition internal/dto/dtoapplication
```

- [ ] **Step 2: 更新 svcapplication/application.go 的 package 声明**

```go
// 文件: internal/service/svcapplication/application.go
// 第1行: package svcappdefinition → package svcapplication
// 第6行: "github.com/morehao/ark-iam/iam/internal/dto/dtoappdefinition" → "github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
```

```go
// 完整文件: internal/service/svcapplication/application.go
package svcapplication

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/biz/gcontext/gincontext"
    "github.com/morehao/golib/biz/genericdao"
    "github.com/morehao/golib/glog"
    "github.com/morehao/golib/gutil"
)

type ApplicationSvc interface {
    Create(ctx *gin.Context, req *dtoapplication.CreateReq) (*dtoapplication.CreateResp, error)
    Update(ctx *gin.Context, req *dtoapplication.UpdateReq) error
    Delete(ctx *gin.Context, req *dtoapplication.DeleteReq) error
    Detail(ctx *gin.Context, req *dtoapplication.DetailReq) (*dtoapplication.DetailResp, error)
    PageList(ctx *gin.Context, req *dtoapplication.PageListReq) (*dtoapplication.PageListResp, error)
}

type applicationSvc struct{}

var _ ApplicationSvc = (*applicationSvc)(nil)

func NewApplicationSvc() ApplicationSvc {
    return &applicationSvc{}
}

func (svc *applicationSvc) Create(ctx *gin.Context, req *dtoapplication.CreateReq) (*dtoapplication.CreateResp, error) {
    entity := &model.ApplicationEntity{
        Code: req.Code, Name: req.Name, Description: req.Description,
        LogoURL: req.LogoURL, HomepageURL: req.HomepageURL, Type: req.Type,
        Sort: req.Sort, CreatedBy: gincontext.GetUserID(ctx),
    }
    if err := dao.NewApplicationDao().Insert(ctx, entity); err != nil {
        glog.Errorf(ctx, "[svcapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.ApplicationCreateError)
    }
    return &dtoapplication.CreateResp{AppDefID: entity.ID, Code: entity.Code}, nil
}

func (svc *applicationSvc) Update(ctx *gin.Context, req *dtoapplication.UpdateReq) error {
    updateMap := map[string]any{
        "name": req.Name, "description": req.Description, "logo_url": req.LogoURL,
        "homepage_url": req.HomepageURL, "type": req.Type, "status": req.Status,
        "sort": req.Sort, "updated_by": gincontext.GetUserID(ctx),
    }
    if err := dao.NewApplicationDao().UpdateMap(ctx, req.AppDefID, updateMap); err != nil {
        glog.Errorf(ctx, "[svcapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.ApplicationUpdateError)
    }
    return nil
}

func (svc *applicationSvc) Delete(ctx *gin.Context, req *dtoapplication.DeleteReq) error {
    userID := gincontext.GetUserID(ctx)
    if err := dao.NewApplicationDao().Delete(ctx, req.AppDefID, userID); err != nil {
        glog.Errorf(ctx, "[svcapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.ApplicationDeleteError)
    }
    return nil
}

func (svc *applicationSvc) Detail(ctx *gin.Context, req *dtoapplication.DetailReq) (*dtoapplication.DetailResp, error) {
    entity, err := dao.NewApplicationDao().GetByID(ctx, req.AppDefID)
    if err != nil || entity == nil || entity.ID == 0 {
        glog.Errorf(ctx, "[svcapplication.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.ApplicationGetDetailError)
    }
    return &dtoapplication.DetailResp{
        AppDefID: entity.ID, Code: entity.Code, Name: entity.Name,
        Description: entity.Description, LogoURL: entity.LogoURL, HomepageURL: entity.HomepageURL,
        Type: entity.Type, Status: entity.Status, Sort: entity.Sort,
        CreatedAt: entity.CreatedAt.Format("2006-01-02 15:04:05"),
    }, nil
}

func (svc *applicationSvc) PageList(ctx *gin.Context, req *dtoapplication.PageListReq) (*dtoapplication.PageListResp, error) {
    cond := &dao.ApplicationCond{
        BaseCond: &genericdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
        Name: req.Name, Type: req.Type, Status: req.Status,
    }
    list, total, err := dao.NewApplicationDao().GetPageListByCond(ctx, cond)
    if err != nil {
        glog.Errorf(ctx, "[svcapplication.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.ApplicationGetPageListError)
    }
    items := make([]dtoapplication.PageListItem, 0, len(list))
    for _, v := range list {
        items = append(items, dtoapplication.PageListItem{
            AppDefID: v.ID, Code: v.Code, Name: v.Name, Description: v.Description,
            Type: v.Type, Status: v.Status, Sort: v.Sort,
            CreatedAt: v.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    return &dtoapplication.PageListResp{List: items, Total: total}, nil
}
```

- [ ] **Step 3: 更新 ctrapplication/application.go 的 package + import**

```go
// 文件: internal/controller/ctrapplication/application.go
package ctrapplication

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/internal/dto/dtoapplication"
    "github.com/morehao/ark-iam/iam/internal/service/svcapplication"
    "github.com/morehao/golib/biz/gcontext/gincontext"
)
```

将所有 `dtoappdefinition.CreateReq` 等引用改为 `dtoapplication.CreateReq`，`svcappdefinition.ApplicationSvc` → `svcapplication.ApplicationSvc`。

- [ ] **Step 4: 更新 dtoapplication/ 中 package 声明**

```go
// 文件: internal/dto/dtoapplication/request.go
package dtoapplication

// 内容保持不变，仅修改 package 声明

// 文件: internal/dto/dtoapplication/response.go
package dtoapplication

// 内容保持不变，仅修改 package 声明
```

---

### Task 2: 更新路由注册

**涉及文件:**
- 修改: `backend/apps/iam/internal/router/app_definition.go` → 改为 `application.go`，函数名和路径更新
- 修改: `backend/apps/iam/internal/router/permission.go` → `applicationRouter` 改 `oauthClientRouter`
- 修改: `backend/apps/iam/internal/router/router.go` → 函数名引用更新

- [ ] **Step 1: 重命名 app_definition.go → application.go，更新内容**

```bash
cd /Users/songhao/Documents/practice/go/ark-iam/backend/apps/iam/internal/router
mv app_definition.go application.go
```

```go
// 文件: internal/router/application.go
package router

import (
    "github.com/morehao/ark-iam/iam/internal/controller/ctrapplication"
    "github.com/morehao/golib/biz/gconstant"
    "github.com/morehao/golib/biz/gserver/ginserver"
)

func applicationRouter(groups *ginserver.RouterGroups) {
    ctr := ctrapplication.NewApplicationCtr()
    v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
    v1RouterGroup.POST("/application/create", ctr.Create)
    v1RouterGroup.POST("/application/delete", ctr.Delete)
    v1RouterGroup.POST("/application/update", ctr.Update)
    v1RouterGroup.GET("/application/detail", ctr.Detail)
    v1RouterGroup.GET("/application/pageList", ctr.PageList)
}
```

- [ ] **Step 2: 更新 permission.go——applicationRouter 函数改名 oauthClientRouter + 路由路径**

```go
// 文件: internal/router/permission.go

func oauthClientRouter(groups *ginserver.RouterGroups) {
    appCtr := ctroauthclient.NewOAuthClientCtr()

    v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
    v1RouterGroup.POST("/oauthClient/create", appCtr.Create)
    v1RouterGroup.POST("/oauthClient/delete", appCtr.Delete)
    v1RouterGroup.POST("/oauthClient/update", appCtr.Update)
    v1RouterGroup.GET("/oauthClient/detail", appCtr.Detail)
    v1RouterGroup.POST("/oauthClient/pageList", appCtr.PageList)
    v1RouterGroup.GET("/oauthClient/secrets", appCtr.ListSecrets)
    v1RouterGroup.POST("/oauthClient/secrets", appCtr.CreateSecret)
    v1RouterGroup.DELETE("/oauthClient/secrets/:secretId", appCtr.DeleteSecret)
}
```

- [ ] **Step 3: 更新 router.go——函数调用更新**

```go
// 文件: internal/router/router.go
package router

import "github.com/morehao/golib/biz/gserver/ginserver"

func RegisterRouter(groups *ginserver.RouterGroups, appName string) {
    tenantRouter(groups)
    apiKeyRouter(groups)
    userRouter(groups)
    roleRouter(groups)
    menuRouter(groups)
    scopeRouter(groups)
    resourceRouter(groups)
    roleMenuRouter(groups)
    roleScopeRouter(groups)
    userRoleRouter(groups)
    oauthClientRouter(groups)
    applicationRouter(groups)
    authRouter(groups)
    connectorRouter(groups)
    departmentRouter(groups)
    organizationRouter(groups)
    systemRouter(groups)
    organizationRoleRouter(groups)
    organizationUserRouter(groups)
    organizationRoleUserRouter(groups)
    logRouter(groups)
    personRouter(groups)
    domainRouter(groups)
}
```

---

### Task 3: 添加枚举常量

**涉及文件:**
- 修改: `backend/apps/iam/model/application.go`
- 修改: `backend/apps/iam/model/oauth_client.go`

- [ ] **Step 1: model/application.go 添加常量**

```go
// 在 package 声明之后，ApplicationEntity 之前添加

// 应用类型
const (
    AppTypeFirstParty = "first_party"
    AppTypeThirdParty = "third_party"
)

// 应用状态
const (
    AppStatusEnable  = "enable"
    AppStatusDisable = "disable"
)
```

- [ ] **Step 2: model/oauth_client.go 添加常量**

```go
// 在 package 声明之后，OAuthClientEntity 之前添加

const (
    OAuthClientTypeFirstParty = "first_party"
    OAuthClientTypeThirdParty = "third_party"
)

const (
    OAuthClientStatusEnable  = "enable"
    OAuthClientStatusDisable = "disable"
)

const (
    GrantTypeAuthorizationCode = "authorization_code"
    GrantTypeClientCredentials = "client_credentials"
    GrantTypeRefreshToken      = "refresh_token"
)

const (
    TokenEndpointAuthMethodBasic = "client_secret_basic"
    TokenEndpointAuthMethodPost  = "client_secret_post"
    TokenEndpointAuthMethodNone  = "none"
)
```

---

### Task 4: 拆分错误码——新增 OAuthClient 独立错误码

**涉及文件:**
- 修改: `backend/pkg/code/permission.go`
- 修改: `backend/apps/iam/internal/service/svcoauthclient/oauth_client.go`

- [ ] **Step 1: permission.go 新增 OAuthClient 错误码段**

```go
// 在 Application 错误码段（100730-100739）之后，RoleMenu 段之前添加

const (
    OAuthClientCreateError         = 100750
    OAuthClientDeleteError         = 100751
    OAuthClientUpdateError         = 100752
    OAuthClientGetDetailError      = 100753
    OAuthClientGetPageListError    = 100754
    OAuthClientNotExistError       = 100755
    OAuthClientSecretCreateError   = 100756
    OAuthClientSecretGetListError  = 100757
    OAuthClientSecretDeleteError   = 100758
    OAuthClientSecretNotExistError = 100759
)
```

在 `permissionErrorMsgMap` 中添加：
```go
OAuthClientCreateError:         "创建OAuth客户端失败",
OAuthClientDeleteError:         "删除OAuth客户端失败",
OAuthClientUpdateError:         "修改OAuth客户端失败",
OAuthClientGetDetailError:      "查看OAuth客户端详情失败",
OAuthClientGetPageListError:    "查看OAuth客户端列表失败",
OAuthClientNotExistError:       "OAuth客户端不存在",
OAuthClientSecretCreateError:   "创建OAuth客户端密钥失败",
OAuthClientSecretGetListError:  "查看OAuth客户端密钥列表失败",
OAuthClientSecretDeleteError:   "删除OAuth客户端密钥失败",
OAuthClientSecretNotExistError: "OAuth客户端密钥不存在",
```

- [ ] **Step 2: svcoauthclient/oauth_client.go 更新错误码引用**

将文件中所有 `code.ApplicationXxxError` 替换为对应的 `code.OAuthClientXxxError`：

| 原代码 | 新代码 |
|--------|--------|
| `code.ApplicationCreateError` | `code.OAuthClientCreateError` |
| `code.ApplicationDeleteError` | `code.OAuthClientDeleteError` |
| `code.ApplicationUpdateError` | `code.OAuthClientUpdateError` |
| `code.ApplicationGetDetailError` | `code.OAuthClientGetDetailError` |
| `code.ApplicationGetPageListError` | `code.OAuthClientGetPageListError` |
| `code.ApplicationNotExistError` | `code.OAuthClientNotExistError` |
| `code.ApplicationSecretCreateError` | `code.OAuthClientSecretCreateError` |
| `code.ApplicationSecretGetListError` | `code.OAuthClientSecretGetListError` |
| `code.ApplicationSecretDeleteError` | `code.OAuthClientSecretDeleteError` |
| `code.ApplicationSecretNotExistError` | `code.OAuthClientSecretNotExistError` |

---

### Task 5: 修正 OAuthClient DTO 和 Service

**涉及文件:**
- 修改: `backend/apps/iam/internal/dto/dtooauthclient/request.go`
- 修改: `backend/apps/iam/internal/dto/dtooauthclient/response.go`
- 修改: `backend/apps/iam/object/objoauthclient/oauth_client.go`
- 修改: `backend/apps/iam/internal/service/svcoauthclient/oauth_client.go`
- 修改: `backend/apps/iam/internal/controller/ctroauthclient/oauth_client.go`

- [ ] **Step 1: 修正 CreateReq——移除无效字段，添加 AppId**

```go
// 文件: internal/dto/dtooauthclient/request.go
type CreateReq struct {
    AppId       uint     `json:"appId" binding:"required"`
    Name        string   `json:"name" binding:"required"`
    Type        string   `json:"type"`
    IsThirdParty int8    `json:"isThirdParty"`

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
```

- [ ] **Step 2: 修正 UpdateReq——移除无效字段**

```go
// 文件: internal/dto/dtooauthclient/request.go
type UpdateReq struct {
    OAuthClientID uint   `json:"oauthClientId" binding:"required"`
    Name          string `json:"name"`
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
```

- [ ] **Step 3: 展开 DetailResp——返回所有字段**

```go
// 文件: internal/dto/dtooauthclient/response.go
package dtooauthclient

type CreateResp struct {
    OAuthClientID uint   `json:"oauthClientId"`
    ClientID      string `json:"clientID"`
}

type DetailResp struct {
    OAuthClientID           uint     `json:"oauthClientId"`
    TenantID                uint     `json:"tenantId"`
    AppID                   uint     `json:"appId"`
    ClientID                string   `json:"clientID"`
    Name                    string   `json:"name"`
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
    Type                    string   `json:"type"`
    IsThirdParty            int8     `json:"isThirdParty"`
    Status                  string   `json:"status"`
    CreatedAt               string   `json:"createdAt"`
}

type PageListResp struct {
    List  []PageListItem `json:"list"`
    Total int64          `json:"total"`
}

type PageListItem struct {
    OAuthClientID           uint     `json:"oauthClientId"`
    AppID                   uint     `json:"appId"`
    ClientID                string   `json:"clientID"`
    Name                    string   `json:"name"`
    Type                    string   `json:"type"`
    Status                  string   `json:"status"`
    IsThirdParty            int8     `json:"isThirdParty"`
    GrantTypes              []string `json:"grantTypes"`
    TokenEndpointAuthMethod string   `json:"tokenEndpointAuthMethod"`
    CreatedAt               string   `json:"createdAt"`
}

type SecretResp struct {
    ID            uint64  `json:"id"`
    OAuthClientID uint64  `json:"oauthClientId"`
    Name          string  `json:"name"`
    ValuePrefix   string  `json:"valuePrefix"`
    ExpiredAt     *string `json:"expiresAt"`
    CreatedAt     string  `json:"createdAt"`
}

type SecretListResp struct {
    Total   int64        `json:"total"`
    Secrets []SecretResp `json:"secrets"`
}

type CreateSecretResp struct {
    ID          uint64 `json:"id"`
    Name        string `json:"name"`
    ValuePrefix string `json:"valuePrefix"`
    Secret      string `json:"secret"`
}
```

- [ ] **Step 4: 更新 OAuthClientBaseInfo——简化或移除**

```go
// 文件: object/objoauthclient/oauth_client.go
// PageListItem 不再嵌入此结构，可将该文件移除
// 或保留以备其他模块引用。如果无其他引用，删除此文件。
```

检查是否有其他模块引用 `objoauthclient`：
```bash
cd /Users/songhao/Documents/practice/go/ark-iam/backend/apps/iam
rg "objoauthclient" --type go
```

如果仅有 `dtooauthclient/response.go` 引用且已替换，则删除 `object/objoauthclient/` 目录。

- [ ] **Step 5: 更新 Service——Create 方法添加 AppID**

```go
// 文件: internal/service/svcoauthclient/oauth_client.go
// 在 Create 方法中，添加 AppID 设置

func (svc *oAuthClientSvc) Create(ctx *gin.Context, req *dtooauthclient.CreateReq) (*dtooauthclient.CreateResp, error) {
    insertEntity := &model.OAuthClientEntity{
        TenantID:    gincontext.GetTenantID(ctx),
        AppID:       req.AppId,  // 新增: 关联到 application
        ClientID:    generateClientID(),
        Name:        req.Name,
        RedirectURIs:             marshalStringSlice(req.RedirectURIs),
        PostLogoutRedirectURIs:   marshalStringSlice(req.PostLogoutRedirectURIs),
        GrantTypes:               marshalStringSlice(req.GrantTypes),
        ResponseTypes:            marshalStringSlice(req.ResponseTypes),
        TokenEndpointAuthMethod:  req.TokenEndpointAuthMethod,
        AllowedOrigins:           marshalStringSlice(req.AllowedOrigins),
        RequirePKCE:              req.RequirePKCE,
        RequireAuthTime:          req.RequireAuthTime,
        DefaultScopes:            marshalStringSlice(req.DefaultScopes),
        AccessTokenTTL:           req.AccessTokenTTL,
        RefreshTokenTTL:          req.RefreshTokenTTL,
        Type:                     req.Type,
        IsThirdParty:             req.IsThirdParty,
        CreatedBy:                gincontext.GetUserID(ctx),
    }
    if err := dao.NewOAuthClientDao().Insert(ctx, insertEntity); err != nil {
        glog.Errorf(ctx, "[svcoauthclient.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.OAuthClientCreateError)
    }
    return &dtooauthclient.CreateResp{OAuthClientID: insertEntity.ID, ClientID: insertEntity.ClientID}, nil
}
```

- [ ] **Step 6: 更新 Service——Detail 方法展开返回所有字段**

```go
// 文件: internal/service/svcoauthclient/oauth_client.go
func (svc *oAuthClientSvc) Detail(ctx *gin.Context, req *dtooauthclient.DetailReq) (*dtooauthclient.DetailResp, error) {
    entity, err := newOAuthClientScopeRepo().GetByID(ctx, req.OAuthClientID)
    if err != nil {
        return nil, code.GetError(code.OAuthClientGetDetailError)
    }
    if !oauthClientVisibleToTenant(entity, gincontext.GetTenantID(ctx)) {
        return nil, code.GetError(code.OAuthClientNotExistError)
    }

    var redirectURIs, postLogoutRedirectURIs []string
    var grantTypes, responseTypes []string
    var allowedOrigins, defaultScopes []string
    json.Unmarshal(entity.RedirectURIs, &redirectURIs)
    json.Unmarshal(entity.PostLogoutRedirectURIs, &postLogoutRedirectURIs)
    json.Unmarshal(entity.GrantTypes, &grantTypes)
    json.Unmarshal(entity.ResponseTypes, &responseTypes)
    json.Unmarshal(entity.AllowedOrigins, &allowedOrigins)
    json.Unmarshal(entity.DefaultScopes, &defaultScopes)

    return &dtooauthclient.DetailResp{
        OAuthClientID:           entity.ID,
        TenantID:                entity.TenantID,
        AppID:                   entity.AppID,
        ClientID:                entity.ClientID,
        Name:                    entity.Name,
        RedirectURIs:            redirectURIs,
        PostLogoutRedirectURIs:  postLogoutRedirectURIs,
        GrantTypes:              grantTypes,
        ResponseTypes:           responseTypes,
        TokenEndpointAuthMethod: entity.TokenEndpointAuthMethod,
        AllowedOrigins:          allowedOrigins,
        RequirePKCE:             entity.RequirePKCE,
        RequireAuthTime:         entity.RequireAuthTime,
        DefaultScopes:           defaultScopes,
        AccessTokenTTL:          entity.AccessTokenTTL,
        RefreshTokenTTL:         entity.RefreshTokenTTL,
        Type:                    entity.Type,
        IsThirdParty:            entity.IsThirdParty,
        Status:                  entity.Status,
        CreatedAt:               entity.CreatedAt.Format("2006-01-02 15:04:05"),
    }, nil
}
```

- [ ] **Step 7: 更新 Service——PageList 展开返回**

```go
// 文件: internal/service/svcoauthclient/oauth_client.go
func (svc *oAuthClientSvc) PageList(ctx *gin.Context, req *dtooauthclient.PageListReq) (*dtooauthclient.PageListResp, error) {
    cond := &dao.OAuthClientCond{
        BaseCond: &genericdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
        TenantID: gincontext.GetTenantID(ctx),
        Name: req.Name, Type: req.Type, Status: req.Status,
    }
    list, total, err := newOAuthClientScopeRepo().GetPageListByCond(ctx, cond)
    if err != nil {
        return nil, code.GetError(code.OAuthClientGetPageListError)
    }
    items := make([]dtooauthclient.PageListItem, 0, len(list))
    for _, v := range list {
        var grantTypes []string
        json.Unmarshal(v.GrantTypes, &grantTypes)

        items = append(items, dtooauthclient.PageListItem{
            OAuthClientID:           v.ID,
            AppID:                   v.AppID,
            ClientID:                v.ClientID,
            Name:                    v.Name,
            Type:                    v.Type,
            Status:                  v.Status,
            IsThirdParty:            v.IsThirdParty,
            GrantTypes:              grantTypes,
            TokenEndpointAuthMethod: v.TokenEndpointAuthMethod,
            CreatedAt:               v.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    return &dtooauthclient.PageListResp{List: items, Total: total}, nil
}
```

- [ ] **Step 8: 更新 Service——GetByClientID 展开返回**

```go
// 文件: internal/service/svcoauthclient/oauth_client.go
func (svc *oAuthClientSvc) GetByClientID(ctx *gin.Context, clientID string) (*dtooauthclient.DetailResp, error) {
    entity, err := newOAuthClientScopeRepo().GetByCond(ctx, &dao.OAuthClientCond{ClientID: clientID})
    if err != nil {
        return nil, code.GetError(code.OAuthClientGetDetailError)
    }
    if entity == nil || entity.ID == 0 {
        return nil, code.GetError(code.OAuthClientNotExistError)
    }
    // 使用与 Detail 相同的字段展开逻辑
    var redirectURIs, postLogoutRedirectURIs []string
    var grantTypes, responseTypes []string
    var allowedOrigins, defaultScopes []string
    json.Unmarshal(entity.RedirectURIs, &redirectURIs)
    json.Unmarshal(entity.PostLogoutRedirectURIs, &postLogoutRedirectURIs)
    json.Unmarshal(entity.GrantTypes, &grantTypes)
    json.Unmarshal(entity.ResponseTypes, &responseTypes)
    json.Unmarshal(entity.AllowedOrigins, &allowedOrigins)
    json.Unmarshal(entity.DefaultScopes, &defaultScopes)

    return &dtooauthclient.DetailResp{
        OAuthClientID:           entity.ID,
        TenantID:                entity.TenantID,
        AppID:                   entity.AppID,
        ClientID:                entity.ClientID,
        Name:                    entity.Name,
        RedirectURIs:            redirectURIs,
        PostLogoutRedirectURIs:  postLogoutRedirectURIs,
        GrantTypes:              grantTypes,
        ResponseTypes:           responseTypes,
        TokenEndpointAuthMethod: entity.TokenEndpointAuthMethod,
        AllowedOrigins:          allowedOrigins,
        RequirePKCE:             entity.RequirePKCE,
        RequireAuthTime:         entity.RequireAuthTime,
        DefaultScopes:           defaultScopes,
        AccessTokenTTL:          entity.AccessTokenTTL,
        RefreshTokenTTL:         entity.RefreshTokenTTL,
        Type:                    entity.Type,
        IsThirdParty:            entity.IsThirdParty,
        Status:                  entity.Status,
        CreatedAt:               entity.CreatedAt.Format("2006-01-02 15:04:05"),
    }, nil
}
```

- [ ] **Step 9: 更新 Service——Create 和 Update 不再有 Description/LogoURL/HomepageURL 处理**

这些字段在 OAuthClient 的 Create 和 Update 方法中本来就没有处理过（无效字段），只需确认 DTO 中已移除即可。

- [ ] **Step 10: 更新 Controller——保持与 DTO 一致**

Controller 代码无需大改，只需确认 import 路径。`ctroauthclient/oauth_client.go` 的 import 保持不变（`dtooauthclient` 和 `svcoauthclient` 包名未变）。

---

### Task 6: 新增 TenantApplication 模块

**涉及文件:**
- 新建: `backend/apps/iam/internal/service/svctenantapplication/tenant_application.go`
- 新建: `backend/apps/iam/internal/controller/ctrtenantapplication/tenant_application.go`
- 新建: `backend/apps/iam/internal/dto/dtotenantapplication/request.go`
- 新建: `backend/apps/iam/internal/dto/dtotenantapplication/response.go`
- 新建: `backend/apps/iam/internal/router/tenant_application.go`

- [ ] **Step 1: 创建 DTO——request.go**

```go
// 文件: internal/dto/dtotenantapplication/request.go
package dtotenantapplication

type CreateReq struct {
    AppID  uint   `json:"appId" binding:"required"`
    Status string `json:"status"`
    Config string `json:"config"`
}

type UpdateReq struct {
    TenantAppID uint   `json:"tenantAppId" binding:"required"`
    Status      string `json:"status"`
    Config      string `json:"config"`
}

type DetailReq struct {
    TenantAppID uint `form:"tenantAppId" binding:"required"`
}

type DeleteReq struct {
    TenantAppID uint `json:"tenantAppId" binding:"required"`
}

type PageListReq struct {
    Page     int    `form:"page"`
    PageSize int    `form:"pageSize"`
    Status   string `form:"status"`
}
```

- [ ] **Step 2: 创建 DTO——response.go**

```go
// 文件: internal/dto/dtotenantapplication/response.go
package dtotenantapplication

type CreateResp struct {
    TenantAppID uint `json:"tenantAppId"`
}

type DetailResp struct {
    TenantAppID uint   `json:"tenantAppId"`
    TenantID    uint   `json:"tenantId"`
    AppID       uint   `json:"appId"`
    Status      string `json:"status"`
    Config      string `json:"config"`
    CreatedAt   string `json:"createdAt"`
}

type PageListItem struct {
    TenantAppID uint   `json:"tenantAppId"`
    TenantID    uint   `json:"tenantId"`
    AppID       uint   `json:"appId"`
    Status      string `json:"status"`
    CreatedAt   string `json:"createdAt"`
}

type PageListResp struct {
    List  []PageListItem `json:"list"`
    Total int64          `json:"total"`
}
```

- [ ] **Step 3: 创建 Service**

```go
// 文件: internal/service/svctenantapplication/tenant_application.go
package svctenantapplication

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtotenantapplication"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/biz/gcontext/gincontext"
    "github.com/morehao/golib/biz/genericdao"
    "github.com/morehao/golib/glog"
    "github.com/morehao/golib/gutil"
    "gorm.io/datatypes"
)

type TenantApplicationSvc interface {
    Create(ctx *gin.Context, req *dtotenantapplication.CreateReq) (*dtotenantapplication.CreateResp, error)
    Delete(ctx *gin.Context, req *dtotenantapplication.DeleteReq) error
    Update(ctx *gin.Context, req *dtotenantapplication.UpdateReq) error
    Detail(ctx *gin.Context, req *dtotenantapplication.DetailReq) (*dtotenantapplication.DetailResp, error)
    PageList(ctx *gin.Context, req *dtotenantapplication.PageListReq) (*dtotenantapplication.PageListResp, error)
}

type tenantApplicationSvc struct{}

var _ TenantApplicationSvc = (*tenantApplicationSvc)(nil)

func NewTenantApplicationSvc() TenantApplicationSvc {
    return &tenantApplicationSvc{}
}

func (svc *tenantApplicationSvc) Create(ctx *gin.Context, req *dtotenantapplication.CreateReq) (*dtotenantapplication.CreateResp, error) {
    entity := &model.TenantApplicationEntity{
        TenantID: gincontext.GetTenantID(ctx),
        AppID:    req.AppID,
        Status:   req.Status,
        Config:   datatypes.JSON(req.Config),
        CreatedBy: gincontext.GetUserID(ctx),
    }
    if entity.Status == "" {
        entity.Status = "enable"
    }
    if err := dao.NewTenantApplicationDao().Insert(ctx, entity); err != nil {
        glog.Errorf(ctx, "[svctenantapplication.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.OAuthClientCreateError)
    }
    return &dtotenantapplication.CreateResp{TenantAppID: entity.ID}, nil
}

func (svc *tenantApplicationSvc) Delete(ctx *gin.Context, req *dtotenantapplication.DeleteReq) error {
    entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
    if err != nil || entity == nil || entity.ID == 0 {
        return code.GetError(code.OAuthClientNotExistError)
    }
    if entity.TenantID != gincontext.GetTenantID(ctx) {
        return code.GetError(code.OAuthClientNotExistError)
    }
    if err := dao.NewTenantApplicationDao().Delete(ctx, req.TenantAppID, gincontext.GetUserID(ctx)); err != nil {
        glog.Errorf(ctx, "[svctenantapplication.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.OAuthClientDeleteError)
    }
    return nil
}

func (svc *tenantApplicationSvc) Update(ctx *gin.Context, req *dtotenantapplication.UpdateReq) error {
    entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
    if err != nil || entity == nil || entity.ID == 0 {
        return code.GetError(code.OAuthClientNotExistError)
    }
    if entity.TenantID != gincontext.GetTenantID(ctx) {
        return code.GetError(code.OAuthClientNotExistError)
    }
    updateMap := map[string]any{
        "status":     req.Status,
        "config":     datatypes.JSON(req.Config),
        "updated_by": gincontext.GetUserID(ctx),
    }
    if err := dao.NewTenantApplicationDao().UpdateMap(ctx, req.TenantAppID, updateMap); err != nil {
        glog.Errorf(ctx, "[svctenantapplication.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.OAuthClientUpdateError)
    }
    return nil
}

func (svc *tenantApplicationSvc) Detail(ctx *gin.Context, req *dtotenantapplication.DetailReq) (*dtotenantapplication.DetailResp, error) {
    entity, err := dao.NewTenantApplicationDao().GetByID(ctx, req.TenantAppID)
    if err != nil || entity == nil || entity.ID == 0 {
        return nil, code.GetError(code.OAuthClientNotExistError)
    }
    if entity.TenantID != gincontext.GetTenantID(ctx) {
        return nil, code.GetError(code.OAuthClientNotExistError)
    }
    return &dtotenantapplication.DetailResp{
        TenantAppID: entity.ID,
        TenantID:    entity.TenantID,
        AppID:       entity.AppID,
        Status:      entity.Status,
        Config:      string(entity.Config),
        CreatedAt:   entity.CreatedAt.Format("2006-01-02 15:04:05"),
    }, nil
}

func (svc *tenantApplicationSvc) PageList(ctx *gin.Context, req *dtotenantapplication.PageListReq) (*dtotenantapplication.PageListResp, error) {
    cond := &dao.TenantApplicationCond{
        BaseCond: &genericdao.BaseCond{Page: req.Page, PageSize: req.PageSize},
        TenantID: gincontext.GetTenantID(ctx),
        Status:   req.Status,
    }
    list, total, err := dao.NewTenantApplicationDao().GetPageListByCond(ctx, cond)
    if err != nil {
        return nil, code.GetError(code.OAuthClientGetPageListError)
    }
    items := make([]dtotenantapplication.PageListItem, 0, len(list))
    for _, v := range list {
        items = append(items, dtotenantapplication.PageListItem{
            TenantAppID: v.ID,
            TenantID:    v.TenantID,
            AppID:       v.AppID,
            Status:      v.Status,
            CreatedAt:   v.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    return &dtotenantapplication.PageListResp{List: items, Total: total}, nil
}
```

- [ ] **Step 4: 创建 Controller**

```go
// 文件: internal/controller/ctrtenantapplication/tenant_application.go
package ctrtenantapplication

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/internal/dto/dtotenantapplication"
    "github.com/morehao/ark-iam/iam/internal/service/svctenantapplication"
    "github.com/morehao/golib/biz/gcontext/gincontext"
)

type TenantApplicationCtr interface {
    Create(ctx *gin.Context)
    Delete(ctx *gin.Context)
    Update(ctx *gin.Context)
    Detail(ctx *gin.Context)
    PageList(ctx *gin.Context)
}

type tenantApplicationCtr struct {
    svc svctenantapplication.TenantApplicationSvc
}

var _ TenantApplicationCtr = (*tenantApplicationCtr)(nil)

func NewTenantApplicationCtr() TenantApplicationCtr {
    return &tenantApplicationCtr{
        svc: svctenantapplication.NewTenantApplicationSvc(),
    }
}

func (ctr *tenantApplicationCtr) Create(ctx *gin.Context) {
    var req dtotenantapplication.CreateReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    res, err := ctr.svc.Create(ctx, &req)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    gincontext.Success(ctx, res)
}

func (ctr *tenantApplicationCtr) Delete(ctx *gin.Context) {
    var req dtotenantapplication.DeleteReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    if err := ctr.svc.Delete(ctx, &req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    gincontext.Success(ctx, "删除成功")
}

func (ctr *tenantApplicationCtr) Update(ctx *gin.Context) {
    var req dtotenantapplication.UpdateReq
    if err := ctx.ShouldBindJSON(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    if err := ctr.svc.Update(ctx, &req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    gincontext.Success(ctx, "修改成功")
}

func (ctr *tenantApplicationCtr) Detail(ctx *gin.Context) {
    var req dtotenantapplication.DetailReq
    if err := ctx.ShouldBindQuery(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    res, err := ctr.svc.Detail(ctx, &req)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    gincontext.Success(ctx, res)
}

func (ctr *tenantApplicationCtr) PageList(ctx *gin.Context) {
    var req dtotenantapplication.PageListReq
    if err := ctx.ShouldBindQuery(&req); err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    res, err := ctr.svc.PageList(ctx, &req)
    if err != nil {
        gincontext.Fail(ctx, err)
        return
    }
    gincontext.Success(ctx, res)
}
```

- [ ] **Step 5: 创建路由文件**

```go
// 文件: internal/router/tenant_application.go
package router

import (
    "github.com/morehao/ark-iam/iam/internal/controller/ctrtenantapplication"
    "github.com/morehao/golib/biz/gconstant"
    "github.com/morehao/golib/biz/gserver/ginserver"
)

func tenantApplicationRouter(groups *ginserver.RouterGroups) {
    ctr := ctrtenantapplication.NewTenantApplicationCtr()
    v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
    v1RouterGroup.POST("/tenantApplication/create", ctr.Create)
    v1RouterGroup.POST("/tenantApplication/delete", ctr.Delete)
    v1RouterGroup.POST("/tenantApplication/update", ctr.Update)
    v1RouterGroup.GET("/tenantApplication/detail", ctr.Detail)
    v1RouterGroup.GET("/tenantApplication/pageList", ctr.PageList)
}
```

- [ ] **Step 6: 在 router.go 注册路由**

在 `RegisterRouter` 函数中添加 `tenantApplicationRouter(groups)` 调用。

---

### Task 7: 编译验证

- [ ] **Step 1: 编译验证**

```bash
cd /Users/songhao/Documents/practice/go/ark-iam/backend/apps/iam
go build ./...
```

- [ ] **Step 2: 运行测试**

```bash
make test APP=iam
```

- [ ] **Step 3: 修复编译/测试错误**

如果有编译错误，逐个修复直到 `go build ./...` 通过。
