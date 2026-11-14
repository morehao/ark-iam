package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SMSProvider 是短信供应商适配器的工厂标识。新增供应商只需：
//  1. 实现 Sender 接口（发送一条短信）；
//  2. 在 smsFactories 注册一个工厂；
// 业务侧/配置侧零改动。本期内置 "tencent" 参考实现，阿里云等按同一接口扩展即可。
type SMSProvider string

const (
	SMSProviderTencent SMSProvider = "tencent"
)

// smsFactory 由短信配置构造具体供应商 Sender。
type smsFactory func(cfg SMSConfig) (Sender, error)

// smsFactories 是可插拔供应商注册表（便于适配不同供应商）。
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

// httpDoer 抽象 *http.Client（测试注入 fake）。
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// tencentSMSSender 是腾讯云短信（SendSms）参考适配器。
//
// 不引第三方云 SDK —— 直接用 TC3-HMAC-SHA256 签名 + HTTP JSON，避免为一条短信拉 SDK 依赖。
// 请求体结构按腾讯云 2021-01-11 版 SendSms API（PhoneNumberSet/TemplateID/SignName/TemplateParamSet）。
type tencentSMSSender struct {
	cfg      SMSConfig
	endpoint string
	client   httpDoer
	now      func() time.Time
}

// NewTencentSMSSender 构造腾讯云短信适配器。
func NewTencentSMSSender(cfg SMSConfig) (Sender, error) {
	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("notify: tencent sms credentials incomplete (accessKeyID/accessKeySecret required)")
	}
	if cfg.SignName == "" || cfg.TemplateCode == "" {
		return nil, fmt.Errorf("notify: tencent sms signName/templateCode required")
	}
	region := cfg.Region
	if region == "" {
		region = "ap-guangzhou"
	}
	cfg.Region = region
	return &tencentSMSSender{
		cfg:      cfg,
		endpoint: "https://sms.tencentcloudapi.com",
		client:   &http.Client{Timeout: 10 * time.Second},
		now:      time.Now,
	}, nil
}

func (s *tencentSMSSender) Send(ctx context.Context, msg Message) error {
	if msg.To == "" {
		return fmt.Errorf("%w: empty phone", ErrDeliveryFailed)
	}
	// 手机号需带国家码（如 +8613800000000）；本层按 "+"+号码 归一。
	phone := msg.To
	if !strings.HasPrefix(phone, "+") {
		phone = "+86" + phone
	}

	// TemplateParamSet：本期模板只有一个变量（临时密码）。正文只取其值。
	payload := map[string]any{
		"PhoneNumberSet":   []string{phone},
		"TemplateID":       s.cfg.TemplateCode,
		"SignName":         s.cfg.SignName,
		"TemplateParamSet": []string{msg.Body},
		"SmsSdkAppId":      "", // 由模板体系约定；生产按需补
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("%w: build request: %v", ErrDeliveryFailed, err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", "sms.tencentcloudapi.com")
	req.Header.Set("X-TC-Action", "SendSms")
	req.Header.Set("X-TC-Version", "2021-01-11")
	req.Header.Set("X-TC-Region", s.cfg.Region)
	s.sign(req, body)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: http: %v", ErrDeliveryFailed, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: http status %d", ErrDeliveryFailed, resp.StatusCode)
	}
	// 腾讯云在 200 里用 Response.Error 表达失败。
	var out struct {
		Response struct {
			Error *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("%w: decode: %v", ErrDeliveryFailed, err)
	}
	if out.Response.Error != nil {
		return fmt.Errorf("%w: %s %s", ErrDeliveryFailed, out.Response.Error.Code, out.Response.Error.Message)
	}
	return nil
}

// sign 按 TC3-HMAC-SHA256 给请求签名（腾讯云 API v3）。
func (s *tencentSMSSender) sign(req *http.Request, body []byte) {
	ts := s.now().Unix()
	date := time.Unix(ts, 0).UTC().Format("2006-01-02")
	service := "sms"

	// 1) 规范请求串
	canonicalHeaders := "content-type:application/json; charset=utf-8\nhost:sms.tencentcloudapi.com\n"
	signedHeaders := "content-type;host"
	hashedPayload := sha256Hex(body)
	canonicalRequest := strings.Join([]string{
		http.MethodPost, "/", "", canonicalHeaders, signedHeaders, hashedPayload,
	}, "\n")

	// 2) 待签字符串
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	stringToSign := strings.Join([]string{
		"TC3-HMAC-SHA256", fmt.Sprintf("%d", ts), credentialScope, sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	// 3) 计算签名
	secretDate := hmacSHA256([]byte("TC3"+s.cfg.AccessKeySecret), date)
	secretService := hmacSHA256(secretDate, service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))

	// 4) 拼 Authorization
	auth := fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.cfg.AccessKeyID, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", auth)
	req.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", ts))
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(data))
	return m.Sum(nil)
}
