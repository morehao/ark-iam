// Command embedded-iam-demo 演示「新项目引入 iam 库即自带账号认证与管理体系」。
//
// 形态 2（内嵌三方单体）：一个 Go 进程同时提供
//   - IAM 后端能力（登录/会话/人员/部门/角色/菜单 + /install 初始化 + /oidc 协议）
//   - 内嵌的登录/初始化页（login-web dist）
//
// 运行（本地 sqlite 快速体验）：
//
//	cd backend/examples/embedded-iam-demo && go run . 
//	# 访问 http://localhost:8099/install 初始化管理员，再从 /login 登录
//
// 生产请把 DB.DSN 换成 PostgreSQL，并配置 OIDC 签名密钥与 Redis（见 iam 文档）。
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/iam"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/redis/go-redis/v9"
	_ "github.com/morehao/golib/dbaccess/dbgorm/driver/sqlite"
	_ "github.com/morehao/golib/glog/driver/zap"
)

func main() {
	addr := ":8099"
	dsn := envOr("IAM_DSN", "sqlite://embedded-iam-demo.db")
	issuer := envOr("IAM_ISSUER", "http://localhost"+addr+"/oidc")

	app, err := iam.New(iam.Config{
		Env:     "dev",
		DB:      iam.DBConfig{DSN: dsn},
		Domains:      iam.DefaultConsoleDomains,
		SameDatabase: true,
		// dev 演示：用精简配置合成最小 pkgconfig.Config（OP 使用 dev 缺省签名密钥）。
	})
	if err != nil {
		log.Fatalf("iam.New: %v", err)
	}
	defer app.Close()
	_ = issuer

	// dev 演示：无外部 Redis 时启动进程内 miniredis，供 OP 协议态存储使用。
	if dbclient.RedisCli == nil {
		mr, merr := miniredis.Run()
		if merr != nil {
			log.Fatalf("miniredis: %v", merr)
		}
		dbclient.RedisCli = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	}

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
