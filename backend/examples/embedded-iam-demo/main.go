// Command embedded-iam-demo 演示「新项目引入 iam 库即自带账号认证与管理体系」。
//
// 形态 2（内嵌三方单体）：一个 Go 进程同时提供
//   - IAM 后端能力（登录/会话/人员/部门/角色/菜单 + /install 初始化 + /oidc 协议）
//   - 内嵌的登录/初始化页（login-web dist）
//
// 运行：
//
//	cd backend/examples/embedded-iam-demo && go run .
//	# 访问 http://localhost:8099/install 初始化管理员，再从 /login 登录
//
// 可选环境变量：
//	IAM_DSN         数据库 DSN（默认 sqlite 文件；生产改 PostgreSQL）
//	IAM_ISSUER      OIDC issuer（默认 http://localhost:8099/oidc）
//	IAM_REDIS_ADDR  真实 Redis 地址（如 127.0.0.1:6379）；不设则起进程内 miniredis
//	BOOTSTRAP_TOKEN /install 初始化的引导令牌
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam"
	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"
	_ "github.com/morehao/golib/glog/driver/zap"
)

func main() {
	addr := ":8099"
	dsn := envOr("IAM_DSN", "sqlite://embedded-iam-demo.db")
	issuer := envOr("IAM_ISSUER", "http://localhost"+addr+"/oidc")
	_ = issuer

	// Redis：优先用 IAM_REDIS_ADDR 指定的真实 Redis；未配置则起进程内 miniredis 便于零依赖体验。
	// 两者都通过 Config.Redis 注入（iam 会把实例 Redis 桥接为全局，供 OP/SSO 使用）。
	var redisCfg *iam.RedisConfig
	if redisAddr := os.Getenv("IAM_REDIS_ADDR"); redisAddr != "" {
		redisCfg = &iam.RedisConfig{Addr: redisAddr}
	} else {
		mr, merr := miniredis.Run()
		if merr != nil {
			log.Fatalf("miniredis: %v", merr)
		}
		redisCfg = &iam.RedisConfig{Addr: mr.Addr()}
	}

	app, err := iam.New(iam.Config{
		Env:          "dev",
		DB:           iam.DBConfig{DSN: dsn},
		Redis:        redisCfg,
		Domains:      iam.DefaultConsoleDomains,
		SameDatabase: true,
	})
	if err != nil {
		log.Fatalf("iam.New: %v", err)
	}
	defer app.Close()

	ctx := context.Background()
	if err := app.RunMigrations(ctx); err != nil {
		log.Fatalf("RunMigrations: %v", err)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	if err := app.Mount(engine.Group("")); err != nil {
		log.Fatalf("Mount: %v", err)
	}
	// 内嵌登录/初始化页（同源托管在根，前端相对路径 /v1、/oidc 天然正确）。
	if err := app.MountFrontend(engine, "/"); err != nil {
		log.Fatalf("MountFrontend: %v", err)
	}

	fmt.Printf("embedded-iam-demo 启动: http://localhost%s  (先访问 /install 初始化)\n", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("run: %v", err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
