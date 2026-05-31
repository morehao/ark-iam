# Application 模型拆分实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将当前 `application` 表拆分为三张独立表（全局应用定义、租户订阅、OIDC 客户端），使菜单和角色具备应用维度隔离。

**Architecture:** 当前 `application` 表的数据通过迁移脚本拆分为 `application`（全局，存在命名冲突需用临时名）+ `tenant_application` + `oauth_client`。旧表最终被删除。`menu` 从 tenant 作用域改为 application 作用域。`role` 新增 `application_id` 字段。`application_role` 表被移除。OIDC 集成代码改为查询 `oauth_client` 表。

**Tech Stack:** Go, GORM, MySQL, gin, OIDC (zitadel/oidc)

---

### Task 1：新增 oauth_client + oauth_client_secret 表（非破坏性）

**目的：** 新增 OIDC 客户端表，这是全新的实体，与旧 `application` 表无冲突。当前 `application` 表名被旧表占用，所以新全局 `application` 表先创建为 `app_definition`（Task 4 重命名）。

**Files:**
- Create: `backend/scripts/sql/iam_schema.sql` (追加新表 DDL)
- Create: `backend/apps/iam/model/oauth_client.go`
- Create: `backend/apps/iam/model/oauth_client_secret.go`
- Create: `backend/apps/iam/dao/oauth_client.go`
- Create: `backend/apps/iam/dao/oauth_client_secret.go`

- [ ] **Step 1: 在 schema 中追加 oauth_client 表 DDL**

在 `backend/scripts/sql/iam_schema.sql` 末尾追加：

```sql
CREATE TABLE `oauth_client` (
    `id`                            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '客户端ID',
    `tenant_id`                     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `application_id`                BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属应用id（迁移后填充）',
    `client_id`                     VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'OIDC客户端ID',
    `name`                          VARCHAR(256) NOT NULL DEFAULT '' COMMENT '客户端名称',
    `redirect_uris`                 JSON NOT NULL DEFAULT ('[]') COMMENT '授权回调地址',
    `post_logout_redirect_uris`     JSON NOT NULL DEFAULT ('[]') COMMENT '登出回调地址',
    `grant_types`                   JSON NOT NULL DEFAULT ('["authorization_code"]') COMMENT '授权类型',
    `response_types`                JSON NOT NULL DEFAULT ('["code"]') COMMENT '响应类型',
    `token_endpoint_auth_method`    VARCHAR(32) NOT NULL DEFAULT 'client_secret_basic' COMMENT '令牌端点认证方式',
    `allowed_origins`               JSON NOT NULL DEFAULT ('[]') COMMENT 'CORS白名单',
    `require_pkce`                  TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否强制PKCE',
    `require_auth_time`             TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否需要auth_time声明',
    `default_scopes`                JSON NOT NULL DEFAULT ('["openid","profile"]') COMMENT '默认权限范围',
    `access_token_ttl`              BIGINT NOT NULL DEFAULT 3600 COMMENT '访问令牌有效期(秒)',
    `refresh_token_ttl`             BIGINT NOT NULL DEFAULT 2592000 COMMENT '刷新令牌有效期(秒)',
    `type`                          VARCHAR(32) NOT NULL DEFAULT 'first_party' COMMENT '客户端类型',
    `is_third_party`                TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否第三方应用',
    `status`                        VARCHAR(32) NOT NULL DEFAULT 'enable' COMMENT '状态',
    `created_at`                    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`                    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`                    DATETIME DEFAULT NULL,
    `created_by`                    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`                    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`                    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_client_id` (`client_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_application_id` (`application_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='OIDC客户端表';
```

- [ ] **Step 2: 追加 oauth_client_secret 表 DDL**

```sql
CREATE TABLE `oauth_client_secret` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `oauth_client_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '客户端ID',
    `name`            VARCHAR(256) NOT NULL DEFAULT '' COMMENT '密钥名称',
    `value_hash`      VARCHAR(256) NOT NULL DEFAULT '' COMMENT '密钥哈希',
    `value_prefix`    VARCHAR(16) NOT NULL DEFAULT '' COMMENT '密钥前缀',
    `expired_at`      DATETIME DEFAULT NULL COMMENT '过期时间',
    `revoked_at`      DATETIME DEFAULT NULL COMMENT '撤销时间',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    KEY `idx_oauth_client_id` (`oauth_client_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='OIDC客户端密钥表';
```

- [ ] **Step 3: 创建 model/oauth_client.go**

```go
package model

import (
    "gorm.io/datatypes"
    "gorm.io/gorm"
)

const TableNameOAuthClient = "oauth_client"

type OAuthClientEntity struct {
    gorm.Model
    TenantID        uint   `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
    ApplicationID   uint   `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:所属应用id" json:"applicationID"`
    ClientID        string `gorm:"column:client_id;type:varchar(64);not null;default '';uniqueIndex;comment:OIDC客户端ID" json:"clientID"`
    Name            string `gorm:"column:name;type:varchar(256);not null;default '';comment:客户端名称" json:"name"`

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
    Type                    string         `gorm:"column:type;type:varchar(32);not null;default 'first_party';comment:客户端类型" json:"type"`
    IsThirdParty            int8           `gorm:"column:is_third_party;type:tinyint(1);not null;default 0;comment:是否第三方应用" json:"isThirdParty"`
    Status                  string         `gorm:"column:status;type:varchar(32);not null;default 'enable';comment:状态" json:"status"`

    CreatedBy uint `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
    UpdatedBy uint `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
    DeletedBy uint `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (OAuthClientEntity) TableName() string { return TableNameOAuthClient }

type OAuthClientEntityList []OAuthClientEntity

func (l OAuthClientEntityList) ToMap() map[uint]OAuthClientEntity {
    m := make(map[uint]OAuthClientEntity)
    for _, v := range l {
        m[v.ID] = v
    }
    return m
}
```

- [ ] **Step 4: 创建 model/oauth_client_secret.go**

```go
package model

import (
    "time"
    "gorm.io/gorm"
)

const TableNameOAuthClientSecret = "oauth_client_secret"

type OAuthClientSecretEntity struct {
    gorm.Model
    OAuthClientID uint       `gorm:"column:oauth_client_id;type:bigint unsigned;not null;default 0;comment:客户端ID" json:"oauthClientID"`
    Name          string     `gorm:"column:name;type:varchar(256);not null;default '';comment:密钥名称" json:"name"`
    ValueHash     string     `gorm:"column:value_hash;type:varchar(256);not null;default '';comment:密钥哈希" json:"-"`
    ValuePrefix   string     `gorm:"column:value_prefix;type:varchar(16);not null;default '';comment:密钥前缀" json:"valuePrefix"`
    ExpiredAt     *time.Time `gorm:"column:expired_at;type:datetime;comment:过期时间" json:"expiresAt"`
    RevokedAt     *time.Time `gorm:"column:revoked_at;type:datetime;comment:撤销时间" json:"-"`
    CreatedBy     uint       `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
    UpdatedBy     uint       `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
    DeletedBy     uint       `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (OAuthClientSecretEntity) TableName() string { return TableNameOAuthClientSecret }

type OAuthClientSecretEntityList []OAuthClientSecretEntity

func (l OAuthClientSecretEntityList) ToMap() map[uint]OAuthClientSecretEntity {
    m := make(map[uint]OAuthClientSecretEntity)
    for _, v := range l {
        m[v.ID] = v
    }
    return m
}
```

- [ ] **Step 5: 创建 dao/oauth_client.go**

```go
package dao

import (
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/dbclient"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

type OAuthClientCond struct {
    *genericdao.BaseCond
    TenantID      uint
    ApplicationID uint
    ClientID      string
    Name          string
    Status        string
}

func (c *OAuthClientCond) BuildCondition(db *gorm.DB, tableName string) {
    if c.BaseCond != nil {
        c.BaseCond.BuildCondition(db, tableName)
    }
    if c.TenantID != 0 {
        db.Where(tableName + ".tenant_id = ?", c.TenantID)
    }
    if c.ApplicationID != 0 {
        db.Where(tableName + ".application_id = ?", c.ApplicationID)
    }
    if c.ClientID != "" {
        db.Where(tableName + ".client_id = ?", c.ClientID)
    }
    if c.Name != "" {
        db.Where(tableName + ".name = ?", c.Name)
    }
    if c.Status != "" {
        db.Where(tableName + ".status = ?", c.Status)
    }
}

type OAuthClientDao struct {
    *genericdao.GenericDao[model.OAuthClientEntity, model.OAuthClientEntityList]
}

func NewOAuthClientDao() *OAuthClientDao {
    return &OAuthClientDao{
        GenericDao: genericdao.NewGenericDao[model.OAuthClientEntity, model.OAuthClientEntityList](
            model.TableNameOAuthClient, "OAuthClientDao",
            dbclient.IamDB,
        ),
    }
}
```

- [ ] **Step 6: 创建 dao/oauth_client_secret.go**

```go
package dao

import (
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/dbclient"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

type OAuthClientSecretCond struct {
    *genericdao.BaseCond
    OAuthClientID uint
    Name          string
}

func (c *OAuthClientSecretCond) BuildCondition(db *gorm.DB, tableName string) {
    if c.BaseCond != nil {
        c.BaseCond.BuildCondition(db, tableName)
    }
    if c.OAuthClientID != 0 {
        db.Where(tableName + ".oauth_client_id = ?", c.OAuthClientID)
    }
    if c.Name != "" {
        db.Where(tableName + ".name = ?", c.Name)
    }
}

type OAuthClientSecretDao struct {
    *genericdao.GenericDao[model.OAuthClientSecretEntity, model.OAuthClientSecretEntityList]
}

func NewOAuthClientSecretDao() *OAuthClientSecretDao {
    return &OAuthClientSecretDao{
        GenericDao: genericdao.NewGenericDao[model.OAuthClientSecretEntity, model.OAuthClientSecretEntityList](
            model.TableNameOAuthClientSecret, "OAuthClientSecretDao",
            dbclient.IamDB,
        ),
    }
}
```

---

### Task 2：新增 app_definition + tenant_application 表（非破坏性）

**目的：** 创建全局应用定义表（临时名为 `app_definition`，避免与旧 `application` 表名冲突）+ 租户订阅表。迁移完成后 `app_definition` 重命名为 `application`。

**Files:**
- Modify: `backend/scripts/sql/iam_schema.sql`
- Create: `backend/apps/iam/model/app_definition.go`
- Create: `backend/apps/iam/model/tenant_application.go`
- Create: `backend/apps/iam/dao/app_definition.go`
- Create: `backend/apps/iam/dao/tenant_application.go`

- [ ] **Step 1: 追加 app_definition + tenant_application 表 DDL**

```sql
CREATE TABLE `app_definition` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '应用ID',
    `code`            VARCHAR(64) NOT NULL DEFAULT '' COMMENT '应用编码（唯一）',
    `name`            VARCHAR(128) NOT NULL DEFAULT '' COMMENT '应用名称',
    `description`     TEXT DEFAULT NULL COMMENT '应用描述',
    `logo_url`        VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '应用logo',
    `homepage_url`    VARCHAR(2048) NOT NULL DEFAULT '' COMMENT '应用主页',
    `type`            VARCHAR(32) NOT NULL DEFAULT 'first_party' COMMENT '应用类型',
    `status`          VARCHAR(32) NOT NULL DEFAULT 'enable' COMMENT '状态',
    `sort`            INT NOT NULL DEFAULT 0 COMMENT '排序',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='应用定义表';

CREATE TABLE `tenant_application` (
    `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
    `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户id',
    `application_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '应用id',
    `status`          VARCHAR(32) NOT NULL DEFAULT 'enable' COMMENT '状态',
    `config`          JSON NOT NULL DEFAULT ('{}') COMMENT '租户级应用配置',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at`      DATETIME DEFAULT NULL,
    `created_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    `deleted_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_tenant_app` (`tenant_id`, `application_id`),
    KEY `idx_tenant_id` (`tenant_id`),
    KEY `idx_application_id` (`application_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户应用订阅表';
```

- [ ] **Step 2: 创建 model/app_definition.go**

```go
package model

import "gorm.io/gorm"

const TableNameAppDefinition = "app_definition"

type AppDefinitionEntity struct {
    gorm.Model
    Code        string `gorm:"column:code;type:varchar(64);not null;default '';uniqueIndex;comment:应用编码" json:"code"`
    Name        string `gorm:"column:name;type:varchar(128);not null;default '';comment:应用名称" json:"name"`
    Description string `gorm:"column:description;type:text;comment:应用描述" json:"description"`
    LogoURL     string `gorm:"column:logo_url;type:varchar(2048);not null;default '';comment:应用logo" json:"logoURL"`
    HomepageURL string `gorm:"column:homepage_url;type:varchar(2048);not null;default '';comment:应用主页" json:"homepageURL"`
    Type        string `gorm:"column:type;type:varchar(32);not null;default 'first_party';comment:应用类型" json:"type"`
    Status      string `gorm:"column:status;type:varchar(32);not null;default 'enable';comment:状态" json:"status"`
    Sort        int    `gorm:"column:sort;type:int;not null;default 0;comment:排序" json:"sort"`
    CreatedBy   uint   `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
    UpdatedBy   uint   `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
    DeletedBy   uint   `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (AppDefinitionEntity) TableName() string { return TableNameAppDefinition }

type AppDefinitionEntityList []AppDefinitionEntity

func (l AppDefinitionEntityList) ToMap() map[uint]AppDefinitionEntity {
    m := make(map[uint]AppDefinitionEntity)
    for _, v := range l {
        m[v.ID] = v
    }
    return m
}
```

- [ ] **Step 3: 创建 model/tenant_application.go**

```go
package model

import (
    "gorm.io/datatypes"
    "gorm.io/gorm"
)

const TableNameTenantApplication = "tenant_application"

type TenantApplicationEntity struct {
    gorm.Model
    TenantID      uint           `gorm:"column:tenant_id;type:bigint unsigned;not null;default 0;comment:租户id" json:"tenantID"`
    ApplicationID uint           `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:应用id" json:"applicationID"`
    Status        string         `gorm:"column:status;type:varchar(32);not null;default 'enable';comment:状态" json:"status"`
    Config        datatypes.JSON `gorm:"column:config;type:json;not null;default ('{}');comment:租户级应用配置" json:"config"`
    CreatedBy     uint           `gorm:"column:created_by;type:bigint unsigned;not null;default 0;comment:创建人id" json:"createdBy"`
    UpdatedBy     uint           `gorm:"column:updated_by;type:bigint unsigned;not null;default 0;comment:更新人id" json:"updatedBy"`
    DeletedBy     uint           `gorm:"column:deleted_by;type:bigint unsigned;not null;default 0;comment:删除人id" json:"deletedBy"`
}

func (TenantApplicationEntity) TableName() string { return TableNameTenantApplication }

type TenantApplicationEntityList []TenantApplicationEntity

func (l TenantApplicationEntityList) ToMap() map[uint]TenantApplicationEntity {
    m := make(map[uint]TenantApplicationEntity)
    for _, v := range l {
        m[v.ID] = v
    }
    return m
}
```

- [ ] **Step 4: 创建 dao/app_definition.go**

```go
package dao

import (
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/dbclient"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

type AppDefinitionCond struct {
    *genericdao.BaseCond
    Name   string
    Type   string
    Status string
    Code   string
}

func (c *AppDefinitionCond) BuildCondition(db *gorm.DB, tableName string) {
    if c.BaseCond != nil {
        c.BaseCond.BuildCondition(db, tableName)
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
    if c.Code != "" {
        db.Where(tableName + ".code = ?", c.Code)
    }
}

type AppDefinitionDao struct {
    *genericdao.GenericDao[model.AppDefinitionEntity, model.AppDefinitionEntityList]
}

func NewAppDefinitionDao() *AppDefinitionDao {
    return &AppDefinitionDao{
        GenericDao: genericdao.NewGenericDao[model.AppDefinitionEntity, model.AppDefinitionEntityList](
            model.TableNameAppDefinition, "AppDefinitionDao",
            dbclient.IamDB,
        ),
    }
}
```

- [ ] **Step 5: 创建 dao/tenant_application.go**

```go
package dao

import (
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/dbclient"
    "github.com/morehao/golib/biz/genericdao"
    "gorm.io/gorm"
)

type TenantApplicationCond struct {
    *genericdao.BaseCond
    TenantID      uint
    ApplicationID uint
    Status        string
}

func (c *TenantApplicationCond) BuildCondition(db *gorm.DB, tableName string) {
    if c.BaseCond != nil {
        c.BaseCond.BuildCondition(db, tableName)
    }
    if c.TenantID != 0 {
        db.Where(tableName + ".tenant_id = ?", c.TenantID)
    }
    if c.ApplicationID != 0 {
        db.Where(tableName + ".application_id = ?", c.ApplicationID)
    }
    if c.Status != "" {
        db.Where(tableName + ".status = ?", c.Status)
    }
}

type TenantApplicationDao struct {
    *genericdao.GenericDao[model.TenantApplicationEntity, model.TenantApplicationEntityList]
}

func NewTenantApplicationDao() *TenantApplicationDao {
    return &TenantApplicationDao{
        GenericDao: genericdao.NewGenericDao[model.TenantApplicationEntity, model.TenantApplicationEntityList](
            model.TableNameTenantApplication, "TenantApplicationDao",
            dbclient.IamDB,
        ),
    }
}
```

---

### Task 3：Service + Controller + DTO + Router — 全局应用管理

**目的：** 为 `app_definition`（全局应用定义）添加完整的 CRUD 层，这部分由平台管理员使用。

**Files:**
- Create: `backend/apps/iam/internal/dto/dtoappdefinition/request.go`
- Create: `backend/apps/iam/internal/dto/dtoappdefinition/response.go`
- Create: `backend/apps/iam/internal/dto/dtoappdefinition/obj.go`
- Create: `backend/apps/iam/internal/service/svcappdefinition/application.go`
- Create: `backend/apps/iam/internal/controller/ctrappdefinition/application.go`
- Create: `backend/apps/iam/internal/router/application.go`
- Modify: `backend/apps/iam/internal/router/router.go`

- [ ] **Step 1: 创建 DTO**

`backend/apps/iam/internal/dto/dtoappdefinition/request.go`：
```go
package dtoappdefinition

type CreateReq struct {
    Code        string `json:"code" binding:"required"`
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
    LogoURL     string `json:"logoUrl"`
    HomepageURL string `json:"homepageUrl"`
    Type        string `json:"type"`
    Sort        int    `json:"sort"`
}

type UpdateReq struct {
    AppDefID    uint   `json:"appDefId" binding:"required"`
    Name        string `json:"name"`
    Description string `json:"description"`
    LogoURL     string `json:"logoUrl"`
    HomepageURL string `json:"homepageUrl"`
    Type        string `json:"type"`
    Status      string `json:"status"`
    Sort        int    `json:"sort"`
}

type DetailReq struct {
    AppDefID uint `form:"appDefId" binding:"required"`
}

type DeleteReq struct {
    AppDefID uint `json:"appDefId" binding:"required"`
}

type PageListReq struct {
    Page     int    `form:"page"`
    PageSize int    `form:"pageSize"`
    Name     string `form:"name"`
    Type     string `form:"type"`
    Status   string `form:"status"`
}
```

`backend/apps/iam/internal/dto/dtoappdefinition/response.go`：
```go
package dtoappdefinition

type CreateResp struct {
    AppDefID uint   `json:"appDefId"`
    Code     string `json:"code"`
}

type DetailResp struct {
    AppDefID    uint   `json:"appDefId"`
    Code        string `json:"code"`
    Name        string `json:"name"`
    Description string `json:"description"`
    LogoURL     string `json:"logoUrl"`
    HomepageURL string `json:"homepageUrl"`
    Type        string `json:"type"`
    Status      string `json:"status"`
    Sort        int    `json:"sort"`
    CreatedAt   string `json:"createdAt"`
}

type PageListItem struct {
    AppDefID    uint   `json:"appDefId"`
    Code        string `json:"code"`
    Name        string `json:"name"`
    Description string `json:"description"`
    Type        string `json:"type"`
    Status      string `json:"status"`
    Sort        int    `json:"sort"`
    CreatedAt   string `json:"createdAt"`
}

type PageListResp struct {
    List  []PageListItem `json:"list"`
    Total int64          `json:"total"`
}
```

- [ ] **Step 2: 创建 Service**

`backend/apps/iam/internal/service/svcappdefinition/application.go`：
```go
package svcappdefinition

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/dao"
    "github.com/morehao/ark-iam/iam/internal/dto/dtoappdefinition"
    "github.com/morehao/ark-iam/iam/model"
    "github.com/morehao/ark-iam/pkg/code"
    "github.com/morehao/golib/biz/gcontext/gincontext"
    "github.com/morehao/golib/biz/genericdao"
    "github.com/morehao/golib/glog"
    "github.com/morehao/golib/gutil"
)

type AppDefinitionSvc interface {
    Create(ctx *gin.Context, req *dtoappdefinition.CreateReq) (*dtoappdefinition.CreateResp, error)
    Update(ctx *gin.Context, req *dtoappdefinition.UpdateReq) error
    Delete(ctx *gin.Context, req *dtoappdefinition.DeleteReq) error
    Detail(ctx *gin.Context, req *dtoappdefinition.DetailReq) (*dtoappdefinition.DetailResp, error)
    PageList(ctx *gin.Context, req *dtoappdefinition.PageListReq) (*dtoappdefinition.PageListResp, error)
}

type appDefinitionSvc struct{}

var _ AppDefinitionSvc = (*appDefinitionSvc)(nil)

func NewAppDefinitionSvc() AppDefinitionSvc {
    return &appDefinitionSvc{}
}

func (svc *appDefinitionSvc) Create(ctx *gin.Context, req *dtoappdefinition.CreateReq) (*dtoappdefinition.CreateResp, error) {
    entity := &model.AppDefinitionEntity{
        Code:        req.Code,
        Name:        req.Name,
        Description: req.Description,
        LogoURL:     req.LogoURL,
        HomepageURL: req.HomepageURL,
        Type:        req.Type,
        Sort:        req.Sort,
        CreatedBy:   gincontext.GetUserID(ctx),
    }
    if err := dao.NewAppDefinitionDao().Insert(ctx, entity); err != nil {
        glog.Errorf(ctx, "[svcappdefinition.Create] dao Insert fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.ApplicationCreateError)
    }
    return &dtoappdefinition.CreateResp{
        AppDefID: entity.ID,
        Code:     entity.Code,
    }, nil
}

func (svc *appDefinitionSvc) Update(ctx *gin.Context, req *dtoappdefinition.UpdateReq) error {
    updateMap := map[string]any{
        "name":         req.Name,
        "description":  req.Description,
        "logo_url":     req.LogoURL,
        "homepage_url": req.HomepageURL,
        "type":         req.Type,
        "status":       req.Status,
        "sort":         req.Sort,
        "updated_by":   gincontext.GetUserID(ctx),
    }
    if err := dao.NewAppDefinitionDao().UpdateMap(ctx, req.AppDefID, updateMap); err != nil {
        glog.Errorf(ctx, "[svcappdefinition.Update] dao UpdateMap fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.ApplicationUpdateError)
    }
    return nil
}

func (svc *appDefinitionSvc) Delete(ctx *gin.Context, req *dtoappdefinition.DeleteReq) error {
    userID := gincontext.GetUserID(ctx)
    if err := dao.NewAppDefinitionDao().Delete(ctx, req.AppDefID, userID); err != nil {
        glog.Errorf(ctx, "[svcappdefinition.Delete] dao Delete fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return code.GetError(code.ApplicationDeleteError)
    }
    return nil
}

func (svc *appDefinitionSvc) Detail(ctx *gin.Context, req *dtoappdefinition.DetailReq) (*dtoappdefinition.DetailResp, error) {
    entity, err := dao.NewAppDefinitionDao().GetByID(ctx, req.AppDefID)
    if err != nil || entity == nil || entity.ID == 0 {
        glog.Errorf(ctx, "[svcappdefinition.Detail] dao GetByID fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.ApplicationGetDetailError)
    }
    return &dtoappdefinition.DetailResp{
        AppDefID:    entity.ID,
        Code:        entity.Code,
        Name:        entity.Name,
        Description: entity.Description,
        LogoURL:     entity.LogoURL,
        HomepageURL: entity.HomepageURL,
        Type:        entity.Type,
        Status:      entity.Status,
        Sort:        entity.Sort,
        CreatedAt:   entity.CreatedAt.Format("2006-01-02 15:04:05"),
    }, nil
}

func (svc *appDefinitionSvc) PageList(ctx *gin.Context, req *dtoappdefinition.PageListReq) (*dtoappdefinition.PageListResp, error) {
    cond := &dao.AppDefinitionCond{
        BaseCond: &genericdao.BaseCond{
            Page:     req.Page,
            PageSize: req.PageSize,
        },
        Name:   req.Name,
        Type:   req.Type,
        Status: req.Status,
    }
    list, total, err := dao.NewAppDefinitionDao().GetPageListByCond(ctx, cond)
    if err != nil {
        glog.Errorf(ctx, "[svcappdefinition.PageList] dao GetPageListByCond fail, err:%v, req:%s", err, gutil.ToJsonString(req))
        return nil, code.GetError(code.ApplicationGetPageListError)
    }
    items := make([]dtoappdefinition.PageListItem, 0, len(list))
    for _, v := range list {
        items = append(items, dtoappdefinition.PageListItem{
            AppDefID:    v.ID,
            Code:        v.Code,
            Name:        v.Name,
            Description: v.Description,
            Type:        v.Type,
            Status:      v.Status,
            Sort:        v.Sort,
            CreatedAt:   v.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    return &dtoappdefinition.PageListResp{List: items, Total: total}, nil
}
```

- [ ] **Step 3: 创建 Controller**

`backend/apps/iam/internal/controller/ctrappdefinition/application.go`：
```go
package ctrappdefinition

import (
    "github.com/gin-gonic/gin"
    "github.com/morehao/ark-iam/iam/internal/dto/dtoappdefinition"
    "github.com/morehao/ark-iam/iam/internal/service/svcappdefinition"
    "github.com/morehao/golib/biz/gcontext/gincontext"
)

type AppDefinitionCtr interface {
    Create(ctx *gin.Context)
    Update(ctx *gin.Context)
    Delete(ctx *gin.Context)
    Detail(ctx *gin.Context)
    PageList(ctx *gin.Context)
}

type appDefinitionCtr struct {
    svc svcappdefinition.AppDefinitionSvc
}

var _ AppDefinitionCtr = (*appDefinitionCtr)(nil)

func NewAppDefinitionCtr() AppDefinitionCtr {
    return &appDefinitionCtr{svc: svcappdefinition.NewAppDefinitionSvc()}
}

func (ctr *appDefinitionCtr) Create(ctx *gin.Context) {
    var req dtoappdefinition.CreateReq
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

func (ctr *appDefinitionCtr) Update(ctx *gin.Context) {
    var req dtoappdefinition.UpdateReq
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

func (ctr *appDefinitionCtr) Delete(ctx *gin.Context) {
    var req dtoappdefinition.DeleteReq
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

func (ctr *appDefinitionCtr) Detail(ctx *gin.Context) {
    var req dtoappdefinition.DetailReq
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

func (ctr *appDefinitionCtr) PageList(ctx *gin.Context) {
    var req dtoappdefinition.PageListReq
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

- [ ] **Step 4: 创建路由 + 注册到 router.go**

`backend/apps/iam/internal/router/application.go`：
```go
package router

import (
    "github.com/morehao/ark-iam/iam/internal/controller/ctrappdefinition"
    "github.com/morehao/golib/biz/gconstant"
    "github.com/morehao/golib/biz/gserver/ginserver"
)

func appDefinitionRouter(groups *ginserver.RouterGroups) {
    ctr := ctrappdefinition.NewAppDefinitionCtr()
    v1RouterGroup := groups.MustGetGroup(gconstant.ApiVersionV1)
    v1RouterGroup.POST("/app-definition/create", ctr.Create)
    v1RouterGroup.POST("/app-definition/delete", ctr.Delete)
    v1RouterGroup.POST("/app-definition/update", ctr.Update)
    v1RouterGroup.GET("/app-definition/detail", ctr.Detail)
    v1RouterGroup.GET("/app-definition/pageList", ctr.PageList)
}
```

在 `backend/apps/iam/internal/router/router.go` 中追加：
```go
appDefinitionRouter(groups)
```

- [ ] **Step 5: 验证编译**

```bash
make build APP=iam
```
Expected: 编译成功，无错误。

---

### Task 4：重构 OIDC 客户端 Service（从当前 application 代码拆分）

**目的：** 当前 `svcapplication` 的 CRUD + 密钥逻辑改为操作 `oauth_client` + `oauth_client_secret` 表。当前 API 路径 `/v1/application/*` 保留，但底层改查新表。

**Files:**
- Create: `backend/apps/iam/internal/dto/dtooauthclient/request.go`
- Create: `backend/apps/iam/internal/dto/dtooauthclient/response.go`
- Create: `backend/apps/iam/object/objoauthclient/oauth_client.go`
- Create: `backend/apps/iam/internal/service/svcoauthclient/oauth_client.go`
- Create: `backend/apps/iam/internal/controller/ctroauthclient/oauth_client.go`
- Create (or keep and modify): `backend/apps/iam/internal/router/permission.go` 中的 applicationRouter

- [ ] **Step 1: 创建 Object**

`backend/apps/iam/object/objoauthclient/oauth_client.go`：
```go
package objoauthclient

type OAuthClientBaseInfo struct {
    TenantID       uint   `json:"tenantID"`
    ApplicationID  uint   `json:"applicationID"`
    ClientID       string `json:"clientID"`
    Name           string `json:"name"`
    Type           string `json:"type"`
    Status         string `json:"status"`
    IsThirdParty   int8   `json:"isThirdParty"`
}
```

- [ ] **Step 2: 创建 DTO**

`backend/apps/iam/internal/dto/dtooauthclient/request.go`：包含 CreateReq、UpdateReq、DeleteReq、DetailReq、PageListReq、SecretListReq、CreateSecretReq、DeleteSecretReq、AssignRolesReq、RemoveRoleReq

DTO 结构参考当前 `dtoapplication/request.go` 中所有请求结构体，将 `ApplicationID` 改为 `OAuthClientID`，路径类型不变。

`backend/apps/iam/internal/dto/dtooauthclient/response.go`：包含 CreateResp、DetailResp、PageListItem、PageListResp、SecretResp、SecretListResp、RoleResp、RoleListResp

- [ ] **Step 3: 创建 Service**

`backend/apps/iam/internal/service/svcoauthclient/oauth_client.go`：

将当前 `svcapplication/application.go` 中所有逻辑复制过来，做以下变更：
- 所有 `model.ApplicationEntity` → `model.OAuthClientEntity`
- 所有 `dao.NewApplicationDao()` → `dao.NewOAuthClientDao()`
- 所有 `model.ApplicationSecretEntity` → `model.OAuthClientSecretEntity`
- 所有 `dao.NewApplicationSecretDao()` → `dao.NewOAuthClientSecretDao()`
- `applicationVisibleToTenant` 保留（逻辑不变）
- `ListRoles` / `AssignRoles` / `RemoveRole` 仍操作 `application_role` 表（尚未移除）
- 接口名从 `ApplicationSvc` → `OAuthClientSvc`

- [ ] **Step 4: 创建 Controller**

`backend/apps/iam/internal/controller/ctroauthclient/oauth_client.go`：

将当前 `ctrpermission/application.go` 中所有逻辑复制过来，将 `dtoapplication` → `dtooauthclient`，`svcapplication` → `svcoauthclient`。

- [ ] **Step 5: 更新路由**

修改 `backend/apps/iam/internal/router/permission.go` 中的 `applicationRouter`：

将 `ctrpermission.NewApplicationCtr()` → `ctroauthclient.NewOAuthClientCtr()`，路径 `/application/` 保留不变。

导入路径改为 `ctroauthclient "github.com/morehao/ark-iam/iam/internal/controller/ctroauthclient"`

- [ ] **Step 6: 验证编译**

```bash
make build APP=iam
```
Expected: 编译成功。

---

### Task 5：数据迁移脚本 + 重命名

**目的：** 将当前 `application` 表数据拆分到新表，删除旧表，将 `app_definition` 重命名为 `application`。

**Files:**
- Create: `backend/scripts/sql/migration_v1_v2.sql`

- [ ] **Step 1: 编写迁移脚本**

```sql
-- Step 1: 从旧 application 表提取去重应用定义到 app_definition
INSERT INTO app_definition (code, name, type, status, sort, created_at, updated_at, created_by)
SELECT DISTINCT
    LOWER(REPLACE(name, ' ', '_')) AS code,
    name, type, 'enable' AS status, 0 AS sort,
    NOW(), NOW(), 0
FROM application
WHERE deleted_at IS NULL;

-- Step 2: 为每个 tenant 的每个 app 创建 tenant_application
INSERT INTO tenant_application (tenant_id, application_id, status, created_at, updated_at, created_by)
SELECT DISTINCT a.tenant_id, ad.id, 'enable', NOW(), NOW(), 0
FROM application a
JOIN app_definition ad ON ad.name = a.name
WHERE a.deleted_at IS NULL;

-- Step 3: 将旧 application 记录迁移到 oauth_client
INSERT INTO oauth_client (
    tenant_id, application_id, client_id, name,
    redirect_uris, post_logout_redirect_uris, grant_types, response_types,
    token_endpoint_auth_method, allowed_origins, require_pkce, require_auth_time,
    default_scopes, access_token_ttl, refresh_token_ttl, type, is_third_party, status,
    created_at, updated_at, created_by, updated_by
)
SELECT
    a.tenant_id, ad.id, a.client_id, a.name,
    a.redirect_uris, a.post_logout_redirect_uris, a.grant_types, a.response_types,
    a.token_endpoint_auth_method, a.allowed_origins, a.require_pkce, a.require_auth_time,
    a.default_scopes, a.access_token_ttl, a.refresh_token_ttl, a.type, a.is_third_party, a.status,
    a.created_at, a.updated_at, a.created_by, a.updated_by
FROM application a
JOIN app_definition ad ON ad.name = a.name
WHERE a.deleted_at IS NULL;

-- Step 4: 迁移 application_secret 到 oauth_client_secret
INSERT INTO oauth_client_secret (oauth_client_id, name, value_hash, value_prefix, expired_at, revoked_at, created_at, updated_at, created_by, updated_by)
SELECT oc.id, s.name, s.value_hash, s.value_prefix, s.expired_at, s.revoked_at, s.created_at, s.updated_at, s.created_by, s.updated_by
FROM application_secret s
JOIN oauth_client oc ON oc.client_id = (
    SELECT a.client_id FROM application a WHERE a.id = s.application_id
)
WHERE s.deleted_at IS NULL;

-- Step 5: 更新 application_role 中 application_id → 新 application_id
-- 这里 application_role.application_id 指向旧 application.id，需要更新为 app_definition.id
UPDATE application_role ar
JOIN application a ON a.id = ar.application_id
JOIN app_definition ad ON ad.name = a.name
SET ar.application_id = ad.id
WHERE ar.deleted_at IS NULL;

-- Step 6: 更新 refresh_token 中 application_id → oauth_client.id
ALTER TABLE refresh_token CHANGE COLUMN application_id oauth_client_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
UPDATE refresh_token rt
JOIN oauth_client oc ON oc.client_id = (
    SELECT a.client_id FROM application a WHERE a.id = rt.oauth_client_id
)
SET rt.oauth_client_id = oc.id
WHERE rt.deleted_at IS NULL;

-- Step 7: 更新 menu 表
ALTER TABLE menu ADD COLUMN application_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属应用id' AFTER id;
-- 将所有现有菜单分配给一个默认应用（如果只有一个 app_definition 的话）
-- 手动分配或设置为 0 待后续处理

-- Step 8: 删除旧表
DROP TABLE IF EXISTS application_secret;
DROP TABLE IF EXISTS application;
-- application_role 暂保留，等 Task 7 处理

-- Step 9: 重命名 app_definition → application
RENAME TABLE app_definition TO application;
```

- [ ] **Step 2: 验证迁移**

运行迁移脚本后，验证新表数据完整性：
```sql
SELECT COUNT(*) FROM application;           -- 应等于旧表 DISTINCT name 数量
SELECT COUNT(*) FROM tenant_application;     -- 应等于旧表非空记录数
SELECT COUNT(*) FROM oauth_client;           -- 应等于旧表非空记录数
SELECT COUNT(*) FROM oauth_client_secret;    -- 应等于旧 application_secret 非空记录数
```

- [ ] **Step 3: 更新 Go 模型中的表名常量**

DB 迁移后表名变为 `application`，但 Go model 的 `TableNameAppDefinition` 仍指向 `"app_definition"`。需要更新：

1. 将 `model/app_definition.go` 重命名为 `model/application.go`
2. 结构体名 `AppDefinitionEntity` → `ApplicationEntity`
3. 常量名 `TableNameAppDefinition` → `TableNameApplication`，值从 `"app_definition"` → `"application"`
4. 更新 `dao/app_definition.go` 中所有引用：`AppDefinitionCond` → `ApplicationCond`，`AppDefinitionDao` → `ApplicationDao`
5. 更新 `service/svcappdefinition/`、`controller/ctrappdefinition/`、`dto/dtoappdefinition/` 中所有本地变量名

注意：旧的 `model/application.go`（OIDC client model）将在 Task 9 中删除，所以这里不会冲突。

- [ ] **Step 4: 编译验证**

```bash
make build APP=iam
```

---

### Task 6：角色模型调整（新增 application_id + 移除 application_role）

**目的：** `role` 表新增 `application_id` 字段，使角色支持按应用隔离。移除 `application_role` 表及相关代码。

**Files:**
- Modify: `backend/apps/iam/model/role.go`
- Modify: `backend/apps/iam/dao/role.go`
- Modify: `backend/scripts/sql/iam_schema.sql`
- Remove: `backend/apps/iam/model/application_role.go`
- Remove: `backend/apps/iam/dao/application_role.go`
- Modify: `backend/apps/iam/internal/service/svcoauthclient/oauth_client.go`（移除 ListRoles/AssignRoles/RemoveRole 中对 application_role 的引用）
- Modify: `backend/apps/iam/internal/service/svcpermission/role.go`（移除 ListApplications/AssignApplications）
- Create: `backend/apps/iam/internal/service/svcoauthclient/role.go`（新增基于 role.application_id 的角色管理）

- [ ] **Step 1: 更新 role 表 DDL**

```sql
ALTER TABLE role ADD COLUMN application_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '所属应用id' AFTER tenant_id;
CREATE INDEX idx_tenant_application ON role (tenant_id, application_id);
```

- [ ] **Step 2: 更新 model/role.go**

在 `RoleEntity` 中新增字段：
```go
ApplicationID uint `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:所属应用id" json:"applicationID"`
```

- [ ] **Step 3: 更新 dao/role.go**

在 `RoleCond` 中新增：
```go
ApplicationID uint
```
在 `BuildCondition` 中新增：
```go
if c.ApplicationID != 0 {
    db.Where(tableName + ".application_id = ?", c.ApplicationID)
}
```

- [ ] **Step 4: 更新 OAuthClient Svc 中的角色管理**

当前 `svcapplication.ListRoles`/`AssignRoles`/`RemoveRole` 操作 `application_role` 表。改为通过 `oauth_client.application_id` 找到全局应用，再操作 `role` 表（基于 `role.application_id`）。

关键逻辑：OAuthClient 上的角色管理，实际上是管理该 OAuthClient 所属 `application` 下的角色。所以需要先通过 OAuthClientID 查到 `application_id`，再用 `application_id` 查角色。

```go
// ListRoles — 查询 role 表中 application_id 匹配的记录
func (svc *oAuthClientSvc) ListRoles(ctx *gin.Context, req *dtooauthclient.RoleListReq) (*dtooauthclient.RoleListResp, error) {
    // 1. 先查 OAuthClient 拿到 application_id
    oauthClient, err := dao.NewOAuthClientDao().GetByID(ctx, uint(req.OAuthClientID))
    if err != nil || oauthClient == nil || oauthClient.ID == 0 {
        return nil, code.GetError(code.ApplicationNotExistError)
    }
    // 2. 通过 role.application_id 查角色
    roleDao := dao.NewRoleDao()
    cond := &dao.RoleCond{
        BaseCond:      &genericdao.BaseCond{Page: 1, PageSize: 100},
        TenantID:      gincontext.GetTenantID(ctx),
        ApplicationID: oauthClient.ApplicationID,
    }
    list, total, err := roleDao.GetPageListByCond(ctx, cond)
    if err != nil {
        glog.Errorf(ctx, "[oAuthClientSvc.ListRoles] dao GetPageListByCond fail, err:%v", err)
        return nil, code.GetError(code.RoleApplicationGetListError)
    }
    roles := make([]dtooauthclient.RoleResp, 0, len(list))
    for _, v := range list {
        roles = append(roles, dtooauthclient.RoleResp{
            RoleID:   uint64(v.ID),
            RoleName: v.Name,
            RoleCode: v.Code,
            CreatedAt: v.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }
    return &dtooauthclient.RoleListResp{
        Total: total,
        Roles: roles,
    }, nil
}

// AssignRoles — 不再需要 application_role 表，直接创建 role 时设 application_id
// 但这里 AssignRoles 的语义从"给应用分配已有角色"变为"直接在应用下创建角色"
// 保持接口不变，改为创建 role 并设 application_id
func (svc *oAuthClientSvc) AssignRoles(ctx *gin.Context, req *dtooauthclient.AssignRolesReq) error {
    oauthClient, err := dao.NewOAuthClientDao().GetByID(ctx, uint(req.OAuthClientID))
    if err != nil || oauthClient == nil || oauthClient.ID == 0 {
        return code.GetError(code.ApplicationNotExistError)
    }
    for _, roleID := range req.RoleIDs {
        existing, err := dao.NewRoleDao().GetByID(ctx, uint(roleID))
        if err != nil || existing == nil || existing.ID == 0 {
            continue
        }
        // 更新 role 的 application_id
        _ = dao.NewRoleDao().UpdateMap(ctx, uint(roleID), map[string]any{
            "application_id": oauthClient.ApplicationID,
            "updated_by":     gincontext.GetUserID(ctx),
        })
    }
    return nil
}

// RemoveRole — 取消角色和应用的关联（清空 application_id）
func (svc *oAuthClientSvc) RemoveRole(ctx *gin.Context, req *dtooauthclient.RemoveRoleReq) error {
    existing, err := dao.NewRoleDao().GetByID(ctx, uint(req.RoleID))
    if err != nil || existing == nil || existing.ID == 0 {
        return code.GetError(code.RoleApplicationNotExistError)
    }
    if err := dao.NewRoleDao().UpdateMap(ctx, uint(req.RoleID), map[string]any{
        "application_id": 0,
        "updated_by":     gincontext.GetUserID(ctx),
    }); err != nil {
        return code.GetError(code.RoleApplicationDeleteError)
    }
    return nil
}
```

- [ ] **Step 5: 更新 role svc，移除 application_role 相关方法**

`svcpermission/role.go` 中的 `ListApplications` 和 `AssignApplications` 方法需要改为基于 `role.application_id` 操作，或移除以简化。建议移除，因为角色直接挂在应用下后不需要另外分配应用。

- [ ] **Step 6: 移除旧文件**

删除 `model/application_role.go` 和 `dao/application_role.go`。

- [ ] **Step 7: 编译验证**

```bash
make build APP=iam
```

---

### Task 7：菜单模型调整（tenant_id → application_id）

**目的：** `menu` 表从租户作用域改为应用作用域，所有菜单由平台管理员在应用下定义。

**Files:**
- Modify: `backend/apps/iam/model/menu.go`
- Modify: `backend/apps/iam/dao/menu.go`
- Modify: `backend/scripts/sql/iam_schema.sql`

- [ ] **Step 1: 更新 menu 表结构**

```sql
ALTER TABLE menu DROP COLUMN tenant_id;
-- application_id 已在 Task 5 迁移脚本中新增
```

- [ ] **Step 2: 更新 model/menu.go**

将 `TenantID` 字段替换为 `ApplicationID`：
```go
type MenuEntity struct {
    gorm.Model
    ApplicationID uint   `gorm:"column:application_id;type:bigint unsigned;not null;default 0;comment:所属应用id" json:"applicationID"`
    ParentID     uint   `gorm:"column:parent_id;..."`
    // ... 其他字段不变
}
```

- [ ] **Step 3: 更新 dao/menu.go**

`MenuCond` 中 `TenantID` → `ApplicationID`，`BuildCondition` 中相应修改。

- [ ] **Step 4: 更新 Menu Controller + Menu Service**

`internal/controller/ctrpermission/menu.go` 和 `internal/service/svcpermission/menu.go` 中所有使用 `TenantID` 的地方改为 `ApplicationID`：
- 创建菜单时 `TenantID` → `ApplicationID`（从请求参数获取）
- 分页查询条件 `TenantID` → `ApplicationID`
- 菜单树查询 `TenantID` → `ApplicationID`
- 租户隔离校验改为应用校验（仅平台管理员可操作）

- [ ] **Step 5: 编译验证**

```bash
make build APP=iam
```

---

### Task 8：更新 OIDC 集成代码

**目的：** OIDC 认证流程中查询 `application` 表的地方改为查询 `oauth_client` 表。

**Files:**
- Modify: `backend/apps/iam/internal/service/svcoidc/storage.go`
- Modify: `backend/apps/iam/internal/service/svcoidc/client.go`

- [ ] **Step 1: 更新 GetClientByClientID**

`svcoidc/storage.go:81` 中：
```go
// 修改前
appEntity, err := dao.NewApplicationDao().GetByCond(ctx, &dao.ApplicationCond{ClientID: clientID})

// 修改后
clientEntity, err := dao.NewOAuthClientDao().GetByCond(ctx, &dao.OAuthClientCond{ClientID: clientID})
```

- [ ] **Step 2: 更新 AuthorizeClientIDSecret**

`svcoidc/storage.go:89` 中：
```go
// 修改 oauth_client 和 oauth_client_secret 的查询
clientEntity, err := dao.NewOAuthClientDao().GetByCond(ctx, &dao.OAuthClientCond{ClientID: clientID})
secrets, _, err := dao.NewOAuthClientSecretDao().GetPageListByCond(ctx, &dao.OAuthClientSecretCond{OAuthClientID: clientEntity.ID})
```

- [ ] **Step 3: 更新 CreateAccessAndRefreshTokens**

将 `dao.NewApplicationDao().GetByCond(...)` 改为 `dao.NewOAuthClientDao().GetByCond(...)`，`ApplicationID` 字段改为 `OAuthClientID`。

- [ ] **Step 4: 更新 TokenRequestByRefreshToken**

将 `dao.NewApplicationDao().GetByID(...)` 改为 `dao.NewOAuthClientDao().GetByID(...)`。

- [ ] **Step 5: 编译验证**

```bash
make build APP=iam
```

---

### Task 9：清理旧代码 + 更新 Test

**目的：** 移除所有旧 `application` 相关代码，确保测试通过。

**Files:**
- Remove: `backend/apps/iam/internal/service/svcapplication/`（整个目录）
- Remove: `backend/apps/iam/internal/controller/ctrpermission/application.go`
- Remove: `backend/apps/iam/internal/dto/dtoapplication/`（整个目录，或保留一部分作为 oauth_client DTO 的引用）
- Remove: `backend/apps/iam/object/objapplication/`（整个目录）
- Remove: `backend/apps/iam/model/application.go`（旧表 model）
- Remove: `backend/apps/iam/dao/application.go`（旧表 DAO）

- [ ] **Step 1: 移除旧文件和目录**

删除以下文件/目录：
- `backend/apps/iam/internal/service/svcapplication/`
- `backend/apps/iam/internal/controller/ctrpermission/application.go`
- `backend/apps/iam/internal/dto/dtoapplication/`
- `backend/apps/iam/object/objapplication/`

注意：迁移前确认 `internal/router/oidc.go` 中是否还引用了旧 application 相关包。

- [ ] **Step 2: 清理 router/permission.go 和 router/router.go 中旧 application 引用**

`permission.go` 中 `applicationRouter` 已改为使用 `ctroauthclient`，移除旧的 `ctrpermission.NewApplicationCtr` 引用。

- [ ] **Step 3: 更新 tests**

`svcapplication/application_test.go` 和 `svcapplication/application_tenant_scope_test.go` → 迁移到 `svcoauthclient/` 下，改为测试 `oauth_client` 表和 `OAuthClientSvc`。
`ctrpermission/role_application_controller_test.go` → 更新为适配新路由路径或移除。

- [ ] **Step 4: 运行全部测试**

```bash
make test APP=iam
```
Expected: 所有测试通过。

- [ ] **Step 5: 运行 lint**

```bash
make lint
```
Expected: 无 lint 错误。

---
