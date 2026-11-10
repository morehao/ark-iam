package internal

import "github.com/morehao/ark-iam/contract"

// KeySource 是 OP 公钥来源的最小接口别名（真源 contract.KeySource）。
// 供独立部署的 platform/tenant 进程远程验签，或同进程 gateway 注入 auth 产出的 key set。
type KeySource = contract.KeySource

// Verifier 是 access token 校验器接口别名（真源 contract.Verifier）。
// 装配层 new 出 sdk/rp 的实现并注入 pkg 中间件，从而切断 pkg → sdk 的 module 依赖边。
type Verifier = contract.Verifier
