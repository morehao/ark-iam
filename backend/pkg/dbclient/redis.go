package dbclient

import (
	"github.com/morehao/golib/dbaccess/dbredis"
	"github.com/morehao/golib/glog"
	"github.com/redis/go-redis/v9"
)

var (
	RedisCli *redis.Client
)

// SetRedis 将给定客户端注册为全局 RedisCli（生产用）。可嵌入 IAM 库在实例化时调用它，
// 使仍通过全局 dbclient.RedisCli 取连接的共享代码（OP 协议态存储、SSO 会话校验等）
// 与实例使用同一份 Redis。一进程一 IAM 实例语义下这是安全且必要的桥接。
func SetRedis(client *redis.Client) {
	RedisCli = client
}

func InitRedis(config dbredis.RedisConfig, logConfig *glog.LogConfig) error {
	var opts []dbredis.Option
	if logConfig != nil {
		opts = append(opts, dbredis.WithLogConfig(logConfig))
	}
	client, err := dbredis.New(&config, opts...)
	if err != nil {
		return err
	}
	RedisCli = client
	return nil
}
