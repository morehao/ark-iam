package notify

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// defaultSMTPTimeout 是单次投递的超时（含连接/握手/写）。参照 logoutMaxAttempts 前例设置上限。
const defaultSMTPTimeout = 10 * time.Second

// smtpSender 是 SMTP 邮件渠道实现（标准库 net/smtp）。
type smtpSender struct {
	cfg     MailConfig
	timeout time.Duration
	// sendMail 可在测试中注入，断言主题/收件人/超时；默认真实 smtp.SendMail。
	sendMail func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

// NewSMTPSender 由邮件配置构造 Sender。Host/Port/From 必填，缺失即报错（fail-closed）。
func NewSMTPSender(cfg MailConfig) (Sender, error) {
	if cfg.Host == "" || cfg.Port == 0 || cfg.From == "" {
		return nil, fmt.Errorf("notify: smtp config incomplete (host/port/from required)")
	}
	return &smtpSender{cfg: cfg, timeout: defaultSMTPTimeout, sendMail: smtp.SendMail}, nil
}

func (s *smtpSender) Send(ctx context.Context, msg Message) error {
	if msg.To == "" {
		return fmt.Errorf("%w: empty recipient", ErrDeliveryFailed)
	}
	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	raw := buildMIME(s.cfg.From, msg)

	// 用 goroutine + ctx 实现超时：smtp.SendMail 本身不可取消，超时后返回（后台连接随进程回收）。
	done := make(chan error, 1)
	go func() {
		done <- s.sendMail(addr, auth, s.cfg.From, []string{msg.To}, raw)
	}()
	select {
	case <-ctx.Done():
		return fmt.Errorf("%w: %v", ErrDeliveryFailed, ctx.Err())
	case <-time.After(s.timeout):
		return fmt.Errorf("%w: smtp timeout after %s", ErrDeliveryFailed, s.timeout)
	case err := <-done:
		if err != nil {
			// 不把密码等敏感信息带入错误信息（smtp 错误不含正文，安全）。
			return fmt.Errorf("%w: %v", ErrDeliveryFailed, err)
		}
		return nil
	}
}

// buildMIME 组装一封纯文本 UTF-8 邮件（含必要头部）。
func buildMIME(from string, msg Message) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + msg.To + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.Body)
	b.WriteString("\r\n")
	return []byte(b.String())
}
