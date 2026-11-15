package notify

import (
	"context"
	"errors"
	"fmt"
	"net"

	gnotify "github.com/morehao/golib/gnotify"
	gnotifyemail "github.com/morehao/golib/gnotify/email"
)

// 邮件渠道：复用 golib gnotify/email（底层 wneessen/go-mail）。
//
// 历史实现是手搓 net/smtp + 自拼 MIME，缺中文主题 MIME 编码、无 HTML、超时后连接泄漏。
// 现改为薄适配：把 pkg/notify 的 Sender 接口适配到 gnotify.Notifier，发送/MIME/TLS 全交给
// gnotify/email，本层只做「Message ↔ gnotify.Message」与「错误归一」两件事。

// emailSender 实现 Sender，包装 gnotify/email 的 Notifier。
type emailSender struct {
	n gnotify.Notifier
}

// NewSMTPSender 由邮件配置构造 Sender。Host/Port/From 必填，缺失即报错（fail-closed）。
func NewSMTPSender(cfg MailConfig) (Sender, error) {
	return newEmailSender(cfg, nil)
}

// newEmailSender 构造 emailSender；dial 非 nil 时注入自定义拨号（仅单测用）。
func newEmailSender(cfg MailConfig, dial func(network, addr string) (net.Conn, error)) (Sender, error) {
	if cfg.Host == "" || cfg.Port == 0 || cfg.From == "" {
		return nil, fmt.Errorf("notify: smtp config incomplete (host/port/from required)")
	}
	gn := gnotifyemail.EmailConfig{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
		From:     cfg.From,
		TLS:      cfg.TLS,
	}
	var n gnotify.Notifier
	var err error
	if dial != nil {
		n, err = gnotifyemail.NewWithDialer(gn, dial)
	} else {
		n, err = gnotifyemail.New(gn)
	}
	if err != nil {
		return nil, fmt.Errorf("notify: build email notifier: %w", err)
	}
	return &emailSender{n: n}, nil
}

func (s *emailSender) Send(ctx context.Context, msg Message) error {
	err := s.n.Send(ctx, gnotify.Message{
		To:      []string{msg.To},
		Subject: msg.Subject,
		Body:    msg.Body,
	})
	return normalizeGnotifyErr(err)
}

// normalizeGnotifyErr 把 gnotify 的哨兵错误归一为 pkg/notify 的哨兵错误，
// 保持上层 errors.Is 语义不变（ErrNoRecipient 原样保留；发送失败/配置非法归 ErrDeliveryFailed）。
func normalizeGnotifyErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gnotify.ErrNoRecipient) {
		return fmt.Errorf("%w: %v", ErrNoRecipient, err)
	}
	return fmt.Errorf("%w: %v", ErrDeliveryFailed, err)
}
