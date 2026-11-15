package notify

import (
	"context"
	"errors"
	"fmt"
)

// ErrChannelLessConfig：总闸开（Enabled=true）但没有任何渠道启用——属于配置错误，
// 由进程入口据环境决定「拒绝启动（prod）/ 降级告警（dev）」（见执行计划 R3）。
var ErrChannelLessConfig = errors.New("notify: enabled=true but neither mail nor sms channel is enabled")

// Notifier 是渠道选择点（唯一选渠道的地方），业务层只调它。
//
// 渠道选择规则（见执行计划 §1.4bis）：
//   - 有邮件偏好地址且邮件渠道可用 → 走邮件；
//   - 否则有手机且短信渠道可用 → 走短信；
//   - 两者皆不可用 → ErrNoRecipient（有渠道但无地址）/ ErrNoChannel（无渠道）。
type Notifier struct {
	enabled         bool
	requireDelivery bool
	mail            Sender
	sms             Sender
}

// Enabled 返回总闸是否开启（业务层据此决定是否走投递 vs 回显）。
func (n *Notifier) Enabled() bool {
	return n != nil && n.enabled
}

// RequireDelivery 返回失败语义（true=报错 / false=回退回显）。
func (n *Notifier) RequireDelivery() bool {
	return n != nil && n.requireDelivery
}

// Recipient 是投递目标的地址对（来自自然人自带联系方式）。
type Recipient struct {
	Email string
	Phone string
}

// SendTempPassword 是投递临时密码的便捷入口（业务层用）。
//
// 渠道选择与失败语义在内部完成；返回：
//   - nil：投递成功（明文由调用方丢弃、响应不回显）；
//   - ErrNoChannel / ErrNoRecipient：不可投递（业务层据 requireDelivery 分支）；
//   - 其它（errors.Is ErrDeliveryFailed）：投递失败。
func (n *Notifier) SendTempPassword(ctx context.Context, to Recipient, password string) error {
	if n == nil || !n.enabled {
		return ErrNoChannel
	}
	body := fmt.Sprintf(
		"您的账号初始/重置临时密码为：%s\n\n首次登录必须修改密码。请勿将本密码透露给他人；若非本人操作请及时联系管理员。",
		password,
	)
	// 渠道选择：邮件优先，其次短信（见 §1.4bis）。
	if n.mail != nil && to.Email != "" {
		return n.mail.Send(ctx, Message{To: to.Email, Subject: "【Ark IAM】账号初始密码", Body: body, Kind: KindTempPassword})
	}
	if n.sms != nil && to.Phone != "" {
		return n.sms.Send(ctx, Message{To: to.Phone, Subject: "", Body: password, Kind: KindTempPassword})
	}
	// 有渠道但没有可用地址；或没有渠道。
	if n.mail == nil && n.sms == nil {
		return ErrNoChannel
	}
	return ErrNoRecipient
}

// —— 进程级默认 Notifier ——
//
// 一进程一 IAM 实例语义下，投递通道是进程级依赖（与 db/redis 同）。服务构造时默认取
// notify.Default()；需要单测替换时用 NewXxxSvc 的 WithNotifier 选项覆盖（见 svc 包）。
var defaultNotifier *Notifier

// SetDefault 设置进程级默认 Notifier（由 sdk/server 在启动时按配置装配后调用）。
func SetDefault(n *Notifier) { defaultNotifier = n }

// Default 返回进程级默认 Notifier（未设置时返回 nil，业务层按「通道未启用」处理）。
func Default() *Notifier { return defaultNotifier }

// Disabled 返回一个显式「关闭态」Notifier（SendTempPassword 恒 ErrNoChannel）。
// 用于未配置/降级场景，等价于「通道不可用」。
func Disabled() *Notifier { return &Notifier{enabled: false} }

// DeliveryOutcome 是「投递临时密码」的业务结果，供 service 直接映射到响应。
type DeliveryOutcome struct {
	// Delivered 是否已成功投递（true 时 EchoPassword 必为空）。
	Delivered bool
	// EchoPassword 回退时供响应回显的明文（Delivered=true 或通道未启用时为空）。
	EchoPassword string
}

// DeliverTempPassword 是给业务层的最小集成入口：封装「事务后投递 + 失败按 requireDelivery 分支」
// 的全部判断，业务侧只需把结果映射到 DTO，并用返回的 err（若有）决定错误码。
//
// 语义（见执行计划 §1.4bis / §1.2bis）：
//   - 通道未启用（n==nil 或 !enabled）→ 返回 {EchoPassword: pwd}，不改行为（回显现状）；
//   - 有可用渠道且投递成功 → {Delivered:true}（明文不回显）；
//   - 投递不可达（ErrNoChannel/ErrNoRecipient）或投递失败：
//       requireDelivery=true  → 返回 err（业务层据以返回 NotifyDeliverError）；
//       requireDelivery=false → {EchoPassword: pwd}（回退回显）。
//
// password 只在内存中传递，绝不写日志。
func (n *Notifier) DeliverTempPassword(ctx context.Context, to Recipient, password string) (DeliveryOutcome, error) {
	if n == nil || !n.enabled {
		return DeliveryOutcome{EchoPassword: password}, nil
	}
	err := n.SendTempPassword(ctx, to, password)
	if err == nil {
		return DeliveryOutcome{Delivered: true}, nil
	}
	if n.requireDelivery {
		return DeliveryOutcome{}, err
	}
	// 过渡口径：回退回显，标记未投递。
	return DeliveryOutcome{Delivered: false, EchoPassword: password}, nil
}

// NewForTest 构造一个可直接注入 Sender 的 Notifier（仅供跨包单测使用，生产走 FromConfig）。
// mail/sms 任一可为 nil；两者皆 nil 时等价于无渠道。
func NewForTest(mail, sms Sender, requireDelivery bool) *Notifier {
	return &Notifier{enabled: true, requireDelivery: requireDelivery, mail: mail, sms: sms}
}
