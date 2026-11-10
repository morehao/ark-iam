package server

import "github.com/morehao/ark-iam/pkg/oidckit"

// KeySource 是 OP 公钥来源的最小接口别名（真源 sdk/rp.KeySource，经 pkg/oidckit 别名）。
// 供独立部署的 platform/tenant 进程远程验签，或同进程 gateway 注入 auth 产出的 key set。
type KeySource = oidckit.KeySource
