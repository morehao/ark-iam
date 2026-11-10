// Package server 是 sdk 的门面 / 编排层（对外唯一集成入口）。
//
// 分层（见 .dsh/docs/specs/2026-11-10-final-contract-sdk-architecture.md §3）：
//
//   - 本包（sdk/server）对外公开 API，**只暴露 contract 类型与 framework-free 类型**，
//     不 import apps/* 与 pkg 的实现包；实现全部下沉 sdk/server/internal。
//   - sdk/server/internal 是唯一 import apps/* + pkg 的地方，负责 DB/Redis/迁移、
//     各业务域装配与验签器（contract.Verifier）注入。
//   - 调用方进程入口 → sdk/server → sdk/server/internal → apps 各域库 → pkg → contract。
//
// 公开形态：New(Config) / Mount / MountConfig / RunMigrations / KeySource / DBGetter /
// Redis / Close。使用者只需 go get ark-iam/sdk。
package server

import "github.com/morehao/ark-iam/sdk/server/internal"

// App 是门面实例（定义在 internal，此处为别名，方法随别名透出）。
type App = internal.App

// Config 是嵌入方配置。
type Config = internal.Config

// DBConfig 是数据源配置。
type DBConfig = internal.DBConfig

// RedisConfig 是 Redis 连接配置。
type RedisConfig = internal.RedisConfig

// Domain 是 IAM 业务域枚举。
type Domain = internal.Domain

// KeySource 是 OP 公钥来源接口（真源 contract.KeySource）。
type KeySource = internal.KeySource

// ErrTableConflict 表示同名非 IAM 表冲突。
type ErrTableConflict = internal.ErrTableConflict

// ErrDomainDependency 表示域依赖不闭合。
type ErrDomainDependency = internal.ErrDomainDependency

// ErrDBDSNRequired 表示 Config.DB.DSN 缺失。
var ErrDBDSNRequired = internal.ErrDBDSNRequired

// DefaultConsoleDomains 是简易用户体系最小域集（形态 2 推荐）。
var DefaultConsoleDomains = internal.DefaultConsoleDomains

// —— 域常量透出（供调用方选择 Config.Domains）——
const (
	DomainCoreAuth = internal.DomainCoreAuth
	DomainInstall  = internal.DomainInstall

	DomainPerson     = internal.DomainPerson
	DomainUser       = internal.DomainUser
	DomainDepartment = internal.DomainDepartment
	DomainRole       = internal.DomainRole
	DomainMenu       = internal.DomainMenu

	DomainTenant      = internal.DomainTenant
	DomainApplication = internal.DomainApplication
	DomainOAuthClient = internal.DomainOAuthClient
	DomainConnector   = internal.DomainConnector
	DomainDomain      = internal.DomainDomain
	DomainAudit       = internal.DomainAudit
	DomainAPIKey      = internal.DomainAPIKey
	DomainMachineUser = internal.DomainMachineUser
	DomainInvite      = internal.DomainInvite
	DomainTenantApp   = internal.DomainTenantApp
	DomainSelfService = internal.DomainSelfService
)

// New 构造门面实例（转发 internal.New）。
func New(cfg Config) (*App, error) { return internal.New(cfg) }

// FromPkgConfig 从完整 pkgconfig.Config 派生门面 Config（转发 internal.FromPkgConfig）。
//
// 参数用 any 接收以保持 sdk/server 顶层公开文件**不 import pkg**（config 属 pkg）；
// 内部按 *pkgconfig.Config 断言，nil 或其它类型按 nil 处理。调用方照常传
// config.Conf（*pkgconfig.Config）即可，无需感知此细节。
func FromPkgConfig(conf any, domains []Domain, jwksURL, issuer string) Config {
	return internal.FromPkgConfig(conf, domains, jwksURL, issuer)
}
