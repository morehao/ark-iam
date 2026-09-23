package svcinstall

import (
	"crypto/subtle"
	"os"
	"strings"

	"github.com/morehao/ark-iam/auth/config"
)

// envBootstrapToken 读取引导令牌。
//
// 声明为 var（而非直接调用 os.Getenv）是为了给测试留一个可替换的接缝：
// 环境变量是进程级全局状态，测试里 t.Setenv 也能改，但把它做成变量能让
// "未配置 token" 这条 fail-closed 路径被独立、显式地覆盖。
var envBootstrapToken = func() string { return os.Getenv(EnvBootstrapToken) }

// configBootstrapToken 读取配置文件里的引导令牌（`install.bootstrapToken`）。
//
// 与 envBootstrapToken 同样做成 var，便于测试独立覆盖两个来源而不必真的去改全局配置。
var configBootstrapToken = func() string {
	if config.Conf == nil {
		return ""
	}
	return config.Conf.Install.BootstrapToken
}

// effectiveBootstrapToken 解析**生效**的引导令牌：环境变量优先，配置文件回落。
//
// 为什么是 env 优先而不是反过来：两者等价于"创建平台管理员"的权限，而环境变量是
// 更不容易被提交/复制的那条通道（本仓库的 config.yaml 与其内的库口令都是入库的）。
// env 优先让生产可以用环境变量/密钥管理覆盖掉文件里的值，而不必先改文件；
// 同时"env 为空则回落到配置"让本地开发不必每次 export。
//
// 空白即视为"此来源未提供"，继续回落——这样 `BOOTSTRAP_TOKEN=" "` 不会把配置里的
// 有效令牌遮成未配置。
func effectiveBootstrapToken() string {
	if token := trimToken(envBootstrapToken()); token != "" {
		return token
	}
	return trimToken(configBootstrapToken())
}

// tokenConfigured 报告服务端是否配置了引导令牌（空串与纯空白都算未配置）。
func tokenConfigured() bool { return effectiveBootstrapToken() != "" }

// trimToken 归一令牌：部署时通过脚本注入的值很容易带尾随换行（`export TOKEN=$(cat file)`
// 或 docker secret 末尾换行），把这类噪声当成"不匹配"会让运维排查很久。
func trimToken(s string) string { return strings.TrimSpace(s) }

// subtleCompare 常量时间比较，避免调用方通过响应时间逐字节推断令牌。
//
// 这里的威胁模型不是"本机计时攻击"，而是**部署期**：/install 暴露在网络上，
// 逐字节可猜的令牌等于没有令牌。长度不同时 ConstantTimeCompare 直接返回 0（长度本身泄露，
// 但长度不是秘密）。
func subtleCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
