# Changelog

## [v0.1.1] - 2026-11-06

### Features

- **iam**: 可嵌入 IAM 库 + 前端单门面包 @morehao/ark-iam-react 与 CI 发布链路 (#77)
- **app**: gate console requests by app access and add audit log query (#75)
- **application**: add per-user application access and token gate (#74)
- **oauth-client**: enforce client type invariants and secret policy (#73)
- **seed**: replace startup seeding with a one-time /install bootstrap (#72)
- **oidc**: add rp sdk, rpapi directory api, and signing key rotation (#70)
- **oidc**: add app-scoped groups claim with app-owned role codes (#68)
- **dbclient**: enforce explicit tenant scope with fail-closed isolation (#67)
- **tenant**: provision builtin tenant admin with one-time password (#53)
- **tenant**: add tenant suspension gates and move member management to tenant console (#52)
- **role**: scope role assignment by app; tenant console admin-only (#51)
- **iam**: rework service account ownership and consolidate platform console IA (#50)
- **menu**: extract shared svcmenu and add platform my menu tree (#46)
- **menu**: add role menu visibility control (#45)
- **auth**: add login config policy to gate registration and tenant creation (#44)
- **auth**: add login rate limit and move OIDC wiring to router (#39)
- **tenantadmin**: 组织容器化重构 + 租户角色管理 + 权限域清理 (#36)
- migrate to string-ID primary keys with PostgreSQL support (#35)
- **platformadmin**: 按应用作用域管理菜单并支持租户侧动态菜单 (#29)
- **frontend**: complete and beautify all admin apps + backend fixes (#27)
- **oidc**: real-time unified logout via back-channel + frontend-split shared packages (#26)
- **oidc**: invalidate OIDC tokens after global SSO logout (#22)
- 适配 golib v1.31.0 破坏性变更且修复 lint (#19)
- **e2e**: OIDC SSO 测试完善 — 从8个扩展到约29个用例 (#18)
- **sso-test-app**: 新增退出登录功能，支持本地退出和全局退出 (#16)
- SSO E2E Playwright 迁移 & 前端应用精简 (#15)
- **frontend**: add unified API response handler with error interceptor (#9)
- **auth**: unify password, tenant selection, and sso login flow (#8)
- **iam**: complete register, api-key auth, and domain management
- **iam**: implement cross-tenant person model
- **iam**: unify connector authentication flow
- **iam**: add application secret management (list/create/delete)
- **iam**: add connector factory list and test functionality
- **iam**: add SSO provider list and IdP config management
- **iam**: register role application management routes
- **iam**: add role application management methods to controller
- **iam**: add role application management methods to service
- **iam**: add role application DTOs
- **iam**: register role user management routes
- **iam**: add role user management methods to controller
- **iam**: add role user management methods to service
- **iam**: add role user DTOs
- **iam**: register application role management routes
- **iam**: add application role management methods to controller
- **iam**: add application role management methods to service
- **iam**: add application role DTOs
- **iam**: implement proper SSO code exchange for user info
- **iam**: implement proper SSO authorization URL building
- **iam**: register session routes in user router
- **iam**: add session controller for user session management
- **iam**: add session service for user session management
- **iam**: add session DAO for user session management
- **iam**: add session DTO for user session management
- **iam**: complete controller/service/dto layer restructuring
- **iam**: complete code layer restructuring
- **iam**: complete router layer restructuring
- **iam**: complete object layer restructuring
- **iam**: remove objsystem directory
- **iam**: merge objorganization into objtenant
- **iam**: migrate system.go to objtenant
- **iam**: create objaudit directory and migrate log.go
- **auth**: add authentication module with refresh token and SSO support
- **iam**: add infrastructure for all modules including routers, error codes and swagger docs

### Fixes

- **menu**: order menus by sort so console ordering takes effect (#65)
- **auth**: enforce allow_join_by_invite gate on tenant join (#64)
- **oauth-client**: correct display, app binding, and detail route (#62)
- **ui**: pin list action column and bound column widths (#54)
- **auth/oidc**: 协议合规与主流设计对齐 (#32)
- **oidc**: fix OIDC provider correctness (#11)
- **iam**: enforce tenant-scoped object access
- **iam**: close audit gaps in routing and query binding
- **makefile**: correct MAIN_DIR path to resolve build error
- **iam**: fix object package imports after restructuring

### Refactors

- **oidc**: extract pkg/identity and consolidate rp oidc capability (#71)
- **model**: enforce explicit json and enum column types (#69)
- **seed**: identify seed rows by immutable seed key (#63)
- **seed**: declare field authority and rename seed identities (#61)
- **pkg**: flatten shared layer and merge credential helpers (#60)
- **iam**: type dictionary enums with named string types (#59)
- **iam**: type enums, rename app source, and serve dropdowns server-side (#58)
- **iam**: retire department and role business codes (#57)
- **iam**: rename organization container to department (#56)
- **role**: rename admin_level to admin_type with admin/normal (#55)
- **user**: merge member management into user module (#49)
- **ui**: unify design tokens and extract shared display components (#48)
- **auth**: self-registration via oidc flow and tenant member rework (#43)
- **role**: redesign role dimension and split org management (#42)
- rename gincontext id getters with String suffix (#41)
- **auth**: overhaul auth session, password and OIDC connector flows (#40)
- **oidc**: consolidate oidcop into core and unify identity read (#38)
- **oidc**: route provider protocols with hint-carrying handlers (#37)
- **dao**: add generic id type to gorm dao (#33)
- unify routing path params, DTO tags and swagger docs (#31)
- split API paths by application identifier (#30)
- **oidc**: drop login-web OIDC client, align to OP-hosted login UI (#28)
- **iam-split**: split monolithic IAM into four services (auth/platformadmin/tenantadmin/gateway) (#25)
- split IAM into auth, platformadmin, tenantadmin apps (#17)
- adapt to golib v1.31.1 gormplugin and testkit APIs (#20)
- **oidc**: move protocol state from process memory to Redis (#13)
- **iam**: add /auth prefix to auth route paths
- **session**: migrate session revocation to condition-based approach
- **dao**: align api_key, application_secret and session DAOs with genericdao pattern
- **user**: consolidate user related modules into single user.go
- **iam**: split permission and auth services into domain-specific files with matching controller updates
- **iam**: split permission and tenant controllers into domain-specific files
- **iam**: split user controller and service into domain-specific files
- **iam**: domain-based controller/service restructuring
- **iam**: reorganize layers by business domain
- **iam**: remove underscores from package names and add swagger docs
- **code**: remove legacy error code definitions
- **iam**: update code_gen config for tenant management
- **config**: restructure project layout and update code_gen config
- move project files from backend/ to root level
- **sql**: use singular table names in iam schema
- **sql**: migrate to unified IAM schema with complete audit fields

### Docs

- **design**: rewrite design docs with SSO/OIDC concepts and guides (#34)
- **iam**: restructure design docs and align with implementation (#24)
- add IAM lightweight optimization implementation plan
- add IAM lightweight optimization design spec
- add logto migration priority documentation
- **iam**: merge IAM and Logto API docs for feature gap analysis
- 添加 IAM 应用领域划分实施计划
- 添加 IAM 应用功能领域划分设计文档
- **iam**: supplement missing logto apis mapping documentation
- **iam**: add Logto API mapping document
- add project documentation and configuration files

### Other Changes

- build+refactor: golib v1.33.0 升级、user 表名回归、service 上下文改造、DAO BuildCondition 改用 clause (#76)
- Revert "refactor: split IAM into auth, platformadmin, tenantadmin apps (#17)" (#21)
- Feature/OIDC sso flow (#14)
- Feat/login debug (#12)
- Perf/optimize table schema (#10)
