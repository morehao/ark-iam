package svcinstall

import (
	"crypto/subtle"
	"os"
	"strings"
)

// envBootstrapToken 读取引导令牌。
//
// 声明为 var（而非直接调用 os.Getenv）是为了给测试留一个可替换的接缝：
// 环境变量是进程级全局状态，测试里 t.Setenv 也能改，但把它做成变量能让
// "未配置 token" 这条 fail-closed 路径被独立、显式地覆盖。
var envBootstrapToken = func() string { return os.Getenv(EnvBootstrapToken) }

// tokenConfigured 报告服务端是否配置了引导令牌（空串与纯空白都算未配置）。
func tokenConfigured() bool { return trimToken(envBootstrapToken()) != "" }

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
