package server

import (
	pkconfig "github.com/morehao/ark-iam/pkg/config"
)

// Config 是嵌入方提供的配置（比 pkgconfig.Config 精简，缺省安全）。
type Config struct {
	// Domains 选择挂载的业务域；空 = 全量（形态 1 / gateway 用）。
	Domains []Domain

	// Env 运行环境：dev / prod。
	Env string

	// DB 数据库连接（必填）。
	DB DBConfig

	// Redis 缓存连接（可空）。
	Redis *RedisConfig

	// SameDatabase 表示 IAM 表与三方业务表同库（内嵌形态）。为 true 时 RunMigrations 会先做
	// 撞名检测（防止 IAM 表与三方同名业务表冲突）。独立部署（全新库）可保持 false。
	SameDatabase bool

	// AllowExistingTables 是撞名检测的白名单（可选）：列出的表名即使已存在且结构不符 IAM
	// 特征，也允许 RunMigrations 继续（默认零值即严格 fail-fast）。
	AllowExistingTables []string

	// Raw 是完整 pkgconfig.Config（可选）。auth 域的 OP（签名密钥/issuer/cookie/控制台
	// 地址等）依赖完整配置，无法用上面的精简字段表达；提供 Raw 时 auth 域使用它，
	// 未提供时 New 会基于精简字段合成一份最小 Config（仅含 dev 可用的安全缺省）。
	Raw *pkconfig.Config

	// —— 独立部署（非 auth 进程）远程验签 ——

	// JWKSURL 是远程 auth 的 JWKS 地址；空则由 Issuer 推导（标准 discovery）。
	// 独立部署的 platform/tenant 进程应配置它（或 Issuer）以远程拉 JWKS 验签。
	JWKSURL string
	// Issuer 是远程 OP issuer；JWKSURL 为空时据此推导 JWKS 端点。
	Issuer string
	// KeySource 可直接注入 KeySource（同进程 gateway 用，或测试）；非空时优先于远程解析。
	KeySource KeySource
}

// DBConfig 是 IAM 的数据源配置。
type DBConfig struct {
	// DSN 数据库连接串，例如：
	//   postgres://user:pass@host:5432/iam?sslmode=disable&TimeZone=Asia/Shanghai
	//   sqlite://:memory:（测试）
	DSN string

	// Service 连接的服务名（用于日志/追踪标识），缺省 "iam"。
	Service string
}

// RedisConfig 是 IAM 的 Redis 连接配置。
type RedisConfig struct {
	Service  string
	Addr     string
	Password string
	// DB 逻辑库序号；0 为默认。
	DB int
}

// FromPkgConfig 从完整 pkgconfig.Config 派生门面 Config（进程入口便捷构造用）：
//   - Raw 直接引用 conf（auth 域深度依赖完整配置）；
//   - DB.DSN 取 conf.DBConfigs 中 service=="iam"/空 的第一条（否则取第一条）；
//   - Redis 取 conf.RedisConfig（addr 空则不启用）；
//   - Env 取 conf.Server.Env。
//
// domains 为选定的域集合（空=全量）；jwksURL/issuer 用于独立部署的远程验签（同进程可都空）。
func FromPkgConfig(conf *pkconfig.Config, domains []Domain, jwksURL, issuer string) Config {
	cfg := Config{
		Raw:     conf,
		Domains: domains,
		JWKSURL: jwksURL,
		Issuer:  issuer,
	}
	if conf == nil {
		return cfg
	}
	cfg.Env = conf.Server.Env
	// DB：优先 service==iam，否则第一条。
	for _, dbc := range conf.DBConfigs {
		if dbc.Service == "iam" || dbc.Service == "" {
			cfg.DB = DBConfig{DSN: dbc.URL, Service: "iam"}
			break
		}
	}
	if cfg.DB.DSN == "" && len(conf.DBConfigs) > 0 {
		cfg.DB = DBConfig{DSN: conf.DBConfigs[0].URL, Service: "iam"}
	}
	// Redis：addr 非空才启用。
	if conf.RedisConfig.Addr != "" {
		cfg.Redis = &RedisConfig{
			Service:  "iam",
			Addr:     conf.RedisConfig.Addr,
			Password: conf.RedisConfig.Password,
			DB:       conf.RedisConfig.DB,
		}
	}
	return cfg
}
