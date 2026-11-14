package notify

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"testing"
)

// fakeSender 记录投递调用，可配置返回错误。
type fakeSender struct {
	calls []Message
	err   error
}

func (f *fakeSender) Send(_ context.Context, msg Message) error {
	f.calls = append(f.calls, msg)
	return f.err
}

// T1: 未启用总闸 → SendTempPassword 返回 ErrNoChannel。
func TestNotifierDisabledReturnsNoChannel(t *testing.T) {
	n, err := FromConfig(Config{Enabled: false})
	if err != nil {
		t.Fatalf("FromConfig disabled: %v", err)
	}
	if n.Enabled() {
		t.Fatal("disabled notifier should report Enabled()==false")
	}
	if err := n.SendTempPassword(context.Background(), Recipient{Email: "a@b.com"}, "pw"); !errors.Is(err, ErrNoChannel) {
		t.Fatalf("want ErrNoChannel, got %v", err)
	}
}

// T1: 总闸开但无渠道 → FromConfig 报 ErrChannelLessConfig。
func TestFromConfigEnabledButChannelLess(t *testing.T) {
	_, err := FromConfig(Config{Enabled: true})
	if !errors.Is(err, ErrChannelLessConfig) {
		t.Fatalf("want ErrChannelLessConfig, got %v", err)
	}
}

// T1: 渠道选择——邮件优先；无邮件地址走短信；皆无 → ErrNoRecipient。
func TestNotifierChannelSelection(t *testing.T) {
	mail := &fakeSender{}
	sms := &fakeSender{}
	n := &Notifier{enabled: true, mail: mail, sms: sms}

	// 有邮箱 + 手机 → 走邮件
	if err := n.SendTempPassword(context.Background(), Recipient{Email: "a@b.com", Phone: "13800000000"}, "pw"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(mail.calls) != 1 || len(sms.calls) != 0 {
		t.Fatalf("want mail chosen, got mail=%d sms=%d", len(mail.calls), len(sms.calls))
	}
	if mail.calls[0].Kind != KindTempPassword || !strings.Contains(mail.calls[0].Body, "pw") {
		t.Fatalf("mail message wrong: %+v", mail.calls[0])
	}

	// 只有手机 → 走短信
	mail.calls, sms.calls = nil, nil
	if err := n.SendTempPassword(context.Background(), Recipient{Phone: "13800000000"}, "pw2"); err != nil {
		t.Fatalf("send sms: %v", err)
	}
	if len(sms.calls) != 1 || len(mail.calls) != 0 {
		t.Fatalf("want sms chosen, got mail=%d sms=%d", len(mail.calls), len(sms.calls))
	}

	// 有渠道但无任何地址 → ErrNoRecipient
	mail.calls, sms.calls = nil, nil
	if err := n.SendTempPassword(context.Background(), Recipient{}, "pw3"); !errors.Is(err, ErrNoRecipient) {
		t.Fatalf("want ErrNoRecipient, got %v", err)
	}
}

// T1: 投递失败错误向上传播（可 errors.Is ErrDeliveryFailed）。
func TestNotifierPropagatesDeliveryFailure(t *testing.T) {
	mail := &fakeSender{err: errors.Join(ErrDeliveryFailed, errors.New("boom"))}
	n := &Notifier{enabled: true, requireDelivery: true, mail: mail}
	err := n.SendTempPassword(context.Background(), Recipient{Email: "a@b.com"}, "pw")
	if !errors.Is(err, ErrDeliveryFailed) {
		t.Fatalf("want ErrDeliveryFailed, got %v", err)
	}
	if !n.RequireDelivery() {
		t.Fatal("RequireDelivery should be true")
	}
}

// T2: SMTP sender 组装 MIME 并调用 sendMail（注入 fake）。
func TestSMTPSenderBuildsAndSends(t *testing.T) {
	var gotAddr, gotFrom string
	var gotTo []string
	var gotRaw []byte
	s := &smtpSender{
		cfg:     MailConfig{Host: "smtp.example.com", Port: 587, From: "IAM <no-reply@example.com>"},
		timeout: defaultSMTPTimeout,
		sendMail: func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
			gotAddr, gotFrom, gotTo, gotRaw = addr, from, to, msg
			return nil
		},
	}
	err := s.Send(context.Background(), Message{To: "u@example.com", Subject: "【Ark IAM】账号初始密码", Body: "PW123", Kind: KindTempPassword})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotAddr != "smtp.example.com:587" {
		t.Fatalf("addr %q", gotAddr)
	}
	if gotFrom != "IAM <no-reply@example.com>" || len(gotTo) != 1 || gotTo[0] != "u@example.com" {
		t.Fatalf("from/to wrong: %q %v", gotFrom, gotTo)
	}
	raw := string(gotRaw)
	for _, want := range []string{"To: u@example.com", "Subject: ", "PW123", "Content-Type: text/plain"} {
		if !strings.Contains(raw, want) {
			t.Fatalf("MIME missing %q:\n%s", want, raw)
		}
	}
}

// T2: SMTP sender 构造校验。
func TestNewSMTPSenderValidation(t *testing.T) {
	if _, err := NewSMTPSender(MailConfig{Host: ""}); err == nil {
		t.Fatal("want error on missing host")
	}
}

// fakeHTTP 返回固定响应。
type fakeHTTP struct {
	status int
	body   string
	req    *http.Request
}

func (f *fakeHTTP) Do(req *http.Request) (*http.Response, error) {
	f.req = req
	return &http.Response{StatusCode: f.status, Body: io.NopCloser(strings.NewReader(f.body))}, nil
}

// T3: 腾讯云短信 sender 组装请求 + 签名 + 解析成功/失败。
func TestTencentSMSSender(t *testing.T) {
	s, err := NewTencentSMSSender(SMSConfig{
		Provider: "tencent", AccessKeyID: "id", AccessKeySecret: "secret",
		SignName: "ArkIAM", TemplateCode: "12345", Region: "ap-guangzhou",
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	ts := s.(*tencentSMSSender)
	fh := &fakeHTTP{status: 200, body: `{"Response":{"SendStatusSet":[{"Code":"Ok"}],"RequestId":"x"}}`}
	ts.client = fh

	if err := ts.Send(context.Background(), Message{To: "13800000000", Body: "PW123", Kind: KindTempPassword}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if fh.req == nil {
		t.Fatal("no request issued")
	}
	if fh.req.Header.Get("X-TC-Action") != "SendSms" || fh.req.Header.Get("X-TC-Version") != "2021-01-11" {
		t.Fatalf("headers wrong: %v", fh.req.Header)
	}
	if !strings.HasPrefix(fh.req.Header.Get("Authorization"), "TC3-HMAC-SHA256 ") {
		t.Fatalf("authorization not signed: %q", fh.req.Header.Get("Authorization"))
	}

	// 供应商返回 Error → ErrDeliveryFailed
	ts.client = &fakeHTTP{status: 200, body: `{"Response":{"Error":{"Code":"AuthFailure.SignatureFailure","Message":"bad"}}}`}
	if err := ts.Send(context.Background(), Message{To: "13800000000", Body: "x"}); !errors.Is(err, ErrDeliveryFailed) {
		t.Fatalf("want ErrDeliveryFailed on provider error, got %v", err)
	}
}

// T3: 未注册供应商 → 报错（可插拔注册表）。
func TestNewSMSSenderUnknownProvider(t *testing.T) {
	if _, err := NewSMSSender(SMSConfig{Provider: "aliyun"}); err == nil {
		t.Fatal("want error for unregistered provider")
	}
}
