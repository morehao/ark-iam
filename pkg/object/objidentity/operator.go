// Package objidentity 承载"操作人身份"这一业务输入的数据载具。
//
// 身份**不是**横切状态，而是业务输入（谁在操作）：controller 从鉴权结果里取出身份，
// 填进请求 DTO 的 Operator 字段，service 只读 DTO。因此本包是纯数据，
// **不 import gin、不 import 任何上下文包**——service 可以用纯 context.Context 单测，
// 同一次调用在"gin ctx"与"纯 ctx"下行为完全一致。
//
// 与 pkg/identity 的分工：
//   - pkg/identity 回答"这个请求是怎么被鉴权的"（gin Keys 的读写契约，只给
//     middleware / controller / 策略层用）；
//   - objidentity.Operator 回答"这次业务操作是谁发起的"（跟随 DTO 进入 service）。
//
// 边界依据见 golib v1.33.0 `biz/gcontext/README.md` 的「身份不经过 context」。
package objidentity

// Operator 是发起本次业务操作的主体，以及审计/登录日志需要记录的请求元数据。
//
// 字段全部 `json:"-"`：**禁止客户端伪造**。`ShouldBindJSON`/`ShouldBindQuery` 不会写入
// 这些字段，只有 controller 能填；service 拿到空值时必须按既有规则处理
// （需要身份的写操作应 fail-closed，纯读操作可忽略）。
//
// 一律**值嵌入**（不是指针）：DTO 里写 `Operator objidentity.Operator`，
// 未填时是零值而不是 nil，service 不需要判空。
type Operator struct {
	// UserID 是租户成员主键（user.id）：审计的 ActorUserID / created_by / updated_by 口径。
	UserID string `json:"-"`
	// PersonID 是自然人主键（person.id）：跨租户的全局身份。
	PersonID string `json:"-"`
	// ClientIP 是发起请求的客户端 IP（登录日志、审计落库用）。
	ClientIP string `json:"-"`
	// UserAgent 是客户端 UA（登录日志、审计落库用）。
	UserAgent string `json:"-"`
}

// IsEmpty 判断是否未携带任何身份（controller 未填或非 HTTP 入口）。
func (o Operator) IsEmpty() bool {
	return o.UserID == "" && o.PersonID == ""
}
