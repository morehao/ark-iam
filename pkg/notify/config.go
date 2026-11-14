package notify

// Config 是投递层的配置（对应配置文件 notify 段，见 docs/design/configuration-reference.md）。
//
// 三层开关（语义见执行计划 §1.2bis「开关与生效矩阵」）：
//   1. Enabled           总闸：false = 完全走现状（回显明文、零外部调用）
//   2. Mail.Enabled / SMS.Enabled  渠道开关：总闸开时决定哪些渠道参与投递
//   3. RequireDelivery   失败语义：全部渠道投递失败时，true=报错 / false=回退回显
type Config struct {
	// Enabled 总闸。出厂默认 false（仅首启灰度验证）；验证通道可用后应置 true 成为目标默认。
	Enabled bool
	// RequireDelivery 投递失败语义：true=返回功能错误码（账号已建、需重置重试）；
	// false=回退为响应回显明文（delivered=false）。仅在 Enabled=true 时有意义。
	RequireDelivery bool
	// Mail 邮件渠道配置。
	Mail MailConfig
	// SMS 短信渠道配置。
	SMS SMSConfig
}

// MailConfig 是 SMTP 邮件渠道配置。
type MailConfig struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	// Password 建议走环境变量 NOTIFY_MAIL_PASSWORD，配置里留空。
	Password string
	From     string
	// TLS 是否用 SMTPS/STARTTLS。
	TLS bool
}

// SMSConfig 是短信渠道配置。本期单供应商（可插拔，通过 Provider 选适配器）。
type SMSConfig struct {
	Enabled  bool
	Provider string // 供应商标识；本期实现 "tencent"。空=不启用。
	// 供应商相关凭证与签名/模板。
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	// Region 供应商地域（腾讯云用），如 ap-guangzhou。
	Region string
}

// FromConfig 由配置构造 Notifier。
//
// 校验（对应 §1.2bis「总闸开但无渠道=配置错误」）：
//   - Enabled=false：返回一个「关闭态」Notifier（Send 恒返回 ErrNoChannel），不报错；
//   - Enabled=true 且 mail/sms 全关：返回 ErrNoChannelChannellessConfig（进程入口据环境决定拒绝启动/降级告警）。
//
// mail/sms 的具体 Sender 构造见 smtp.go / sms.go（各自 FromMailConfig / FromSMSConfig）。
// 本函数会调用它们；任一构造失败即返回错误（fail-closed，避免「以为开了但发不出」）。
func FromConfig(cfg Config) (*Notifier, error) {
	if !cfg.Enabled {
		return &Notifier{enabled: false, requireDelivery: cfg.RequireDelivery}, nil
	}

	n := &Notifier{enabled: true, requireDelivery: cfg.RequireDelivery}

	if cfg.Mail.Enabled {
		s, err := NewSMTPSender(cfg.Mail)
		if err != nil {
			return nil, err
		}
		n.mail = s
	}
	if cfg.SMS.Enabled {
		s, err := NewSMSSender(cfg.SMS)
		if err != nil {
			return nil, err
		}
		n.sms = s
	}

	if n.mail == nil && n.sms == nil {
		return nil, ErrChannelLessConfig
	}
	return n, nil
}
