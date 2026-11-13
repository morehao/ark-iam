package config

import pkconfig "github.com/morehao/ark-iam/pkg/config"

// Set 注入 auth 域运行时配置（由 sdk/server 在 Mount 前设置，一进程一 IAM 实例语义）。
func Set(c *pkconfig.Config) { Conf = c }
