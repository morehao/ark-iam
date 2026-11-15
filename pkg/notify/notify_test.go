package notify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gnotify "github.com/morehao/golib/gnotify"
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

// ---- 发送适配层（gnotify 适配）----

// fakeGNotifier 是 gnotify.Notifier 的测试桩，记录收到的 gnotify.Message 与 Channel。
type fakeGNotifier struct {
	got     gnotify.Message
	sent    bool
	err     error
	channel string
}

func (f *fakeGNotifier) Send(_ context.Context, msg gnotify.Message) error {
	f.got = msg
	f.sent = true
	return f.err
}
func (f *fakeGNotifier) Channel() string { return f.channel }

// email 适配：Message 字段映射（To→[]string、Subject、Body），失败归一为 ErrDeliveryFailed。
func TestEmailSenderAdapterMapsMessage(t *testing.T) {
	f := &fakeGNotifier{}
	s := &emailSender{n: f}
	err := s.Send(context.Background(), Message{To: "u@example.com", Subject: "【Ark IAM】账号初始密码", Body: "PW123", Kind: KindTempPassword})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(f.got.To) != 1 || f.got.To[0] != "u@example.com" {
		t.Fatalf("To mapping wrong: %v", f.got.To)
	}
	if f.got.Subject != "【Ark IAM】账号初始密码" || f.got.Body != "PW123" {
		t.Fatalf("subject/body wrong: %+v", f.got)
	}
}

func TestEmailSenderAdapterNormalizesError(t *testing.T) {
	f := &fakeGNotifier{err: gnotify.ErrSendFailed}
	s := &emailSender{n: f}
	if err := s.Send(context.Background(), Message{To: "u@example.com", Body: "x"}); !errors.Is(err, ErrDeliveryFailed) {
		t.Fatalf("want ErrDeliveryFailed, got %v", err)
	}
	// 空收件人 → gnotify.ErrNoRecipient → notify.ErrNoRecipient
	f2 := &fakeGNotifier{err: gnotify.ErrNoRecipient}
	s2 := &emailSender{n: f2}
	if err := s2.Send(context.Background(), Message{To: "", Body: "x"}); !errors.Is(err, ErrNoRecipient) {
		t.Fatalf("want ErrNoRecipient, got %v", err)
	}
}

// email 适配：构造校验（host/port/from 必填）。
func TestNewSMTPSenderValidation(t *testing.T) {
	if _, err := NewSMTPSender(MailConfig{Host: ""}); err == nil {
		t.Fatal("want error on missing host")
	}
	if _, err := NewSMTPSender(MailConfig{Host: "smtp.example.com", Port: 0, From: "a@b.com"}); err == nil {
		t.Fatal("want error on missing port")
	}
}

// sms 适配：Message.Body 作为模板单值参数注入 Params["param"]，To→[]string；错误归一。
func TestTencentSenderAdapterMapsMessage(t *testing.T) {
	f := &fakeGNotifier{channel: "tencent"}
	s := &smsSender{n: f}
	if err := s.Send(context.Background(), Message{To: "13800000000", Body: "PW123", Kind: KindTempPassword}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if len(f.got.To) != 1 || f.got.To[0] != "13800000000" {
		t.Fatalf("To mapping wrong: %v", f.got.To)
	}
	if f.got.Params["param"] != "PW123" {
		t.Fatalf("template param wrong: %v", f.got.Params)
	}
}

func TestTencentSenderAdapterNormalizesError(t *testing.T) {
	f := &fakeGNotifier{err: gnotify.ErrSendFailed}
	s := &smsSender{n: f}
	if err := s.Send(context.Background(), Message{To: "13800000000", Body: "x"}); !errors.Is(err, ErrDeliveryFailed) {
		t.Fatalf("want ErrDeliveryFailed, got %v", err)
	}
}

// sms 适配：构造校验（凭证/签名/SDKAppID/模板必填）。
func TestNewTencentSMSSenderValidation(t *testing.T) {
	if _, err := NewTencentSMSSender(SMSConfig{Provider: "tencent"}); err == nil {
		t.Fatal("want error on empty credentials")
	}
	// 缺 SdkAppID 也应报错（官方 SDK 必填）。
	if _, err := NewTencentSMSSender(SMSConfig{
		Provider: "tencent", AccessKeyID: "id", AccessKeySecret: "secret",
		SignName: "ArkIAM", TemplateCode: "12345",
	}); err == nil {
		t.Fatal("want error on missing sdkAppId")
	}
}

// sms：未注册供应商 → 报错（可插拔注册表）。
func TestNewSMSSenderUnknownProvider(t *testing.T) {
	if _, err := NewSMSSender(SMSConfig{Provider: "aliyun"}); err == nil {
		t.Fatal("want error for unregistered provider")
	}
}

// 端到端：经真实腾讯云 SDK + httptest endpoint，验证 smsSender.Send 全链路（适配→SDK→HTTP）
// 与成功/失败归一路径。用 newTencentSMSSender 的 endpoint 注入把请求打到本地测试服务器。
func TestTencentSenderEndToEndViaSDK(t *testing.T) {
	var hit bool
	var actionVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		actionVersion = r.Header.Get("X-TC-Action") + "/" + r.Header.Get("X-TC-Version")
		_, _ = w.Write([]byte(`{"Response":{"SendStatusSet":[{"Code":"Ok","Message":"ok"}],"RequestId":"r1"}}`))
	}))
	defer srv.Close()

	s, err := newTencentSMSSender(SMSConfig{
		Provider: "tencent", AccessKeyID: "id", AccessKeySecret: "secret",
		SignName: "ArkIAM", SdkAppID: "1400000000", TemplateCode: "12345", Region: "ap-guangzhou",
	}, srv.URL)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := s.Send(context.Background(), Message{To: "13800000000", Body: "PW123", Kind: KindTempPassword}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !hit {
		t.Fatal("httptest endpoint not hit (SDK path broken)")
	}
	if actionVersion != "SendSms/2021-01-11" {
		t.Fatalf("unexpected action/version: %q", actionVersion)
	}
}
