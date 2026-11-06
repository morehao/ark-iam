package gateway

import (
	"testing"

	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/golib/dbaccess/dbgorm"
	"github.com/morehao/golib/dbaccess/dbredis"
)

// TestConfigAdapters 校验 gateway 从 pkgconfig.Config 提取 DB/Redis 的适配逻辑。
func TestConfigAdapters(t *testing.T) {
	c := &pkgconfig.Config{
		Server: pkgconfig.Server{Env: "dev"},
		DBConfigs: []dbgorm.Config{
			{URL: "postgres://x/y", Service: "other"},
			{URL: "postgres://iam/db", Service: "iam"},
		},
		RedisConfig: dbredis.RedisConfig{Service: "iam", Addr: "127.0.0.1:6379"},
	}
	if got := envOf(c); got != "dev" {
		t.Fatalf("envOf: want dev, got %q", got)
	}
	if got := dsnFromConf(c); got != "postgres://iam/db" {
		t.Fatalf("dsnFromConf: want iam db, got %q", got)
	}
	rc := redisFromConf(c)
	if rc == nil || rc.Addr != "127.0.0.1:6379" {
		t.Fatalf("redisFromConf: unexpected %+v", rc)
	}

	// nil 安全
	if envOf(nil) != "dev" || dsnFromConf(nil) != "" || redisFromConf(nil) != nil {
		t.Fatal("nil config adapters not nil-safe")
	}
}
