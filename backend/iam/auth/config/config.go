// Package config 承载 auth 域运行时配置。可嵌入 IAM 库下，Conf 由 iam.App 在初始化
// 时注入（一进程一 IAM 实例语义），auth 各 service/controller 通过 config.Conf 读取，
// 避免逐处改函数签名（auth 的 OP 装配深度依赖配置）。
package config

import pkconfig "github.com/morehao/ark-iam/pkg/config"

// Conf 是 auth 域当前生效的配置；由 iam.App 在 Mount 前设置。
var Conf *pkconfig.Config

// Set 由 iam.App 注入配置。
func Set(c *pkconfig.Config) { Conf = c }
