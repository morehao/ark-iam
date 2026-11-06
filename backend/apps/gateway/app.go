package gateway

import (
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam"
	"github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/rpapi"
)

const AppName = "gateway"

// Init 装配 gateway 聚合应用。R4 起 platformadmin/tenantadmin/auth 三域改由
// 可嵌入 IAM 库（iam.New + Mount）统一装配；rpapi 仍是独立的只读目录 API，单独挂载。
//
// 行为与重构前等价：同一套路由布局、同一套鉴权中间件（OIDCAuth + RequireAppAccess）、
// 同一套 back-channel logout 接收端。
func Init(engine *gin.Engine, Conf *config.Config) {
	app, err := iam.New(iam.Config{
		Raw: Conf,
		Env: envOf(Conf),
		DB: iam.DBConfig{
			DSN:     dsnFromConf(Conf),
			Service: "iam",
		},
		Redis: redisFromConf(Conf),
		// Domains=nil → 全量（形态 1：gateway 独立部署挂全部域）。
	})
	if err != nil {
		panic("[gateway.Init] iam.New fail: " + err.Error())
	}
	if err := app.Mount(engine.Group("")); err != nil {
		panic("[gateway.Init] iam.Mount fail: " + err.Error())
	}

	// rpapi 不在 iam 核心库内（独立只读目录 API），仍单独挂载。复用 iam 装配的 auth 域
	// 已产出的进程内 KeySource（app.KeySource()），使 rpapi 本地验签、零网络调用，
	// 与重构前 gateway 注入 auth keySource 的行为一致。
	rpapi.Init(engine, Conf, app.KeySource())
}

func envOf(c *config.Config) string {
	if c == nil {
		return "dev"
	}
	return c.Server.Env
}

func dsnFromConf(c *config.Config) string {
	if c == nil {
		return ""
	}
	for _, dbc := range c.DBConfigs {
		if dbc.Service == "iam" || dbc.Service == "" {
			return dbc.URL
		}
	}
	if len(c.DBConfigs) > 0 {
		return c.DBConfigs[0].URL
	}
	return ""
}

func redisFromConf(c *config.Config) *iam.RedisConfig {
	if c == nil || c.RedisConfig.Addr == "" {
		return nil
	}
	return &iam.RedisConfig{
		Service:  c.RedisConfig.Service,
		Addr:     c.RedisConfig.Addr,
		Password: c.RedisConfig.Password,
		DB:       c.RedisConfig.DB,
	}
}
