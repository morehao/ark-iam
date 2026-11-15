package notify

import (
	"context"
	"fmt"
	"strings"

	gnotify "github.com/morehao/golib/gnotify"
	gnotifysms "github.com/morehao/golib/gnotify/sms/tencent"
)

// 短信渠道：复用 golib gnotify/sms/tencent（底层腾讯云官方 Go SDK）。
//
// 历史实现是手写 TC3-HMAC-SHA256 签名 + 手拼 HTTP JSON（"为一条短信不引 SDK"）。
// v1.36.2 起 golib 官方方向转为「用官方云 SDK」，手写签名易错且需跟随供应商维护。
// 现改为薄适配：把 Sender 适配到 gnotify 的 tencent Notifier，签名/重试/响应解析交给 SDK。

// SMSProvider 保留为业务标识（配置值）。当前仅内置 tencent；新增供应商只需在
// smsFactories 注册一个「用对应 gnotify 渠道构造 Sender」的工厂，业务侧零改动。
type SMSProvider string

const (
	SMSProviderTencent SMSProvider = "tencent"
)

// smsFactory 由短信配置构造具体供应商 Sender。
type smsFactory func(cfg SMSConfig) (Sender, error)

// smsFactories 是可插拔供应商注册表。
var smsFactories = map[SMSProvider]smsFactory{
	SMSProviderTencent: NewTencentSMSSender,
}

// NewSMSSender 按 cfg.Provider 选供应商适配器构造 Sender。
func NewSMSSender(cfg SMSConfig) (Sender, error) {
	p := SMSProvider(strings.ToLower(strings.TrimSpace(cfg.Provider)))
	if p == "" {
		return nil, fmt.Errorf("notify: sms provider not set")
	}
	factory, ok := smsFactories[p]
	if !ok {
		return nil, fmt.Errorf("notify: unsupported sms provider %q", cfg.Provider)
	}
	return factory(cfg)
}

// smsSender 实现 Sender，包装 gnotify/sms/tencent 的 Notifier。
type smsSender struct {
	n gnotify.Notifier
}

// NewTencentSMSSender 构造腾讯云短信适配器（经 gnotify/sms/tencent + 官方 SDK）。
func NewTencentSMSSender(cfg SMSConfig) (Sender, error) {
	return newTencentSMSSender(cfg, "")
}

// newTencentSMSSender 构造适配器；endpoint 非空时覆盖默认 host（单测指向 httptest）。
func newTencentSMSSender(cfg SMSConfig, endpoint string) (Sender, error) {
	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("notify: tencent sms credentials incomplete (accessKeyID/accessKeySecret required)")
	}
	if cfg.SignName == "" || cfg.SdkAppID == "" || cfg.TemplateCode == "" {
		return nil, fmt.Errorf("notify: tencent sms signName/sdkAppId/templateCode required")
	}
	region := cfg.Region
	if region == "" {
		region = "ap-guangzhou"
	}
	gn := gnotifysms.TencentConfig{
		SecretID:   cfg.AccessKeyID,
		SecretKey:  cfg.AccessKeySecret,
		SdkAppID:   cfg.SdkAppID,
		SignName:   cfg.SignName,
		TemplateID: cfg.TemplateCode,
		Region:     region,
		Endpoint:   endpoint,
	}
	n, err := gnotifysms.New(gn)
	if err != nil {
		return nil, fmt.Errorf("notify: build tencent sms notifier: %w", err)
	}
	return &smsSender{n: n}, nil
}

func (s *smsSender) Send(ctx context.Context, msg Message) error {
	// 临时密码短信模板只有一个变量，正文即该变量的值；gnotify tencent 从 Params["param"]
	// 取单值模板参数（或 Params["params"] 逗号分隔多值）。这里把 msg.Body 作为单值注入。
	err := s.n.Send(ctx, gnotify.Message{
		To:     []string{msg.To},
		Params: map[string]string{"param": msg.Body},
	})
	return normalizeGnotifyErr(err)
}
