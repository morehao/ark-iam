// Package notify 是「一次性临时凭据」的投递抽象层（邮件 / 短信）。
//
// 定位：业务服务（建租户 / 建成员 / 重置密码）在事务提交后，把「账号本人的邮箱/手机 + 明文
// 临时密码」交给本层投递；本层负责「按配置选渠道、真正发出去、把失败原因返回」。层内不含任何
// 业务语义，也不感知租户/身份——只做「发一条消息」。
//
// 设计要点：
//   - Sender 是最小接口（Send(ctx, Message) error），SMTP / 短信供应商各自实现，测试用 fake 注入；
//   - Notifier 是渠道选择点：按配置决定走邮件还是短信，选不到可用渠道时返回 ErrNoChannel；
//   - 明文绝不进日志/审计（由调用方保证；本层不打日志打印明文）。
//
// 详见 docs/design/system-design.md §5.8 与执行计划 2026-11-14-notify-delivery-channel-plan.md。
package notify

import (
	"context"
	"errors"
)

// MessageKind 标识消息类型。本期只有临时密码一种，预留扩展。
type MessageKind string

const (
	// KindTempPassword 是「账号初始/重置临时密码」消息。
	KindTempPassword MessageKind = "temp_password"
)

// Message 是待投递的一条消息。
type Message struct {
	// To 收件地址：邮件为邮箱，短信为手机号。**只能来自自然人自带的 primary_email/primary_phone**，
	// 绝不从请求参数取，防乱投 / SSRF。
	To string
	// Subject 主题（短信渠道忽略）。
	Subject string
	// Body 正文（短信渠道通常只取变量值走模板，正文可忽略）。
	Body string
	// Kind 消息类型，决定渠道模板/主题措辞。
	Kind MessageKind
}

// 投递层哨兵错误。调用方用 errors.Is 区分「不可投递」与「投递失败」。
var (
	// ErrNoChannel：没有任何可用渠道（总闸关 / 渠道全关 / 全部 noop）。
	// 语义等价于「通道不可用」，业务层据此走 requireDelivery 分支。
	ErrNoChannel = errors.New("notify: no delivery channel available")
	// ErrNoRecipient：有渠道，但自然人既无邮箱也无对应手机号，无法投递。
	ErrNoRecipient = errors.New("notify: no recipient address (primary_email/primary_phone both empty)")
	// ErrDeliveryFailed：真正尝试投递但失败（供应商报错/网络/超时）。包裹底层错误。
	ErrDeliveryFailed = errors.New("notify: delivery failed")
)

// Sender 是最小的「发一条消息」接口，各渠道实现。
type Sender interface {
	// Send 投递一条消息。实现方需自带超时/重试；返回的错误应可 errors.Is(ErrDeliveryFailed)。
	Send(ctx context.Context, msg Message) error
}
