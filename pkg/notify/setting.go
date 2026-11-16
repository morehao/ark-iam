package notify

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gsetting"
)

// instanceCache 缓存「每个 *gorm.DB 对应一个 gsetting 实例」，避免每次投递都重新 Migrate。
// gsetting.New 会隐式 AutoMigrate，重复调用虽幂等但有开销；这里按 db 指针去重。
var (
	instanceMu    sync.Mutex
	instanceCache = map[any]*gsetting.Setting{}
)

func settingFor(db gormdao.DBGetter) (*gsetting.Setting, error) {
	raw := db(context.Background())
	instanceMu.Lock()
	defer instanceMu.Unlock()
	if s, ok := instanceCache[raw]; ok {
		return s, nil
	}
	s, err := gsetting.New(raw)
	if err != nil {
		return nil, err
	}
	instanceCache[raw] = s
	return s, nil
}

// envOverride 读环境变量覆盖值，未设置为空串。
func envOverride(key string) string {
	return os.Getenv(key)
}

// 运行时配置中心中 notify 分组及其字段键（禁止散落裸字符串）。
const (
	// SettingGroup 是 notify 在 gsetting 中的配置分组名。
	SettingGroup = "notify"

	SettingKeyEnabled         = "enabled"
	SettingKeyRequireDelivery = "requireDelivery"

	SettingKeyMailEnabled  = "mail.enabled"
	SettingKeyMailHost     = "mail.host"
	SettingKeyMailPort     = "mail.port"
	SettingKeyMailUsername = "mail.username"
	SettingKeyMailPassword = "mail.password" // 加密
	SettingKeyMailFrom     = "mail.from"
	SettingKeyMailTLS      = "mail.tls"

	SettingKeySMSEnabled         = "sms.enabled"
	SettingKeySMSProvider        = "sms.provider"
	SettingKeySMSAccessKeyID     = "sms.accessKeyID"
	SettingKeySMSAccessKeySecret = "sms.accessKeySecret" // 加密
	SettingKeySMSSignName        = "sms.signName"
	SettingKeySMSSdkAppID        = "sms.sdkAppId"
	SettingKeySMSTemplateCode    = "sms.templateCode"
	SettingKeySMSRegion          = "sms.region"
)

// LoadConfig 从 gsetting 读取 notify 分组并拼装为 Config。
//
// 语义（见设计 §5.2 / §8）：
//   - 字段缺失/未配置 → 取零值（enabled=false 安全）；
//   - 加密字段由 gsetting 自动解密；解密失败按该字段为空处理（不报错，渠道视为不可用）；
//   - 仅当 gsetting 查询本身返回系统错误（db 故障）时返回 error，由调用方降级。
//
// 环境变量覆盖（NOTIFY_MAIL_PASSWORD / NOTIFY_SMS_ACCESS_KEY_SECRET）在此合并：
// env 优先于 DB（应急覆盖），见设计 §5.5。
func LoadConfig(ctx context.Context, db gormdao.DBGetter) (Config, error) {
	s, err := settingFor(db)
	if err != nil {
		return Config{}, fmt.Errorf("notify: init gsetting: %w", err)
	}
	return loadFrom(ctx, s)
}

// loadFrom 用已初始化的 gsetting 实例读取并拼装（供测试与上层共享实例复用）。
func loadFrom(ctx context.Context, s *gsetting.Setting) (Config, error) {
	var cfg Config

	// 顶层开关。
	if v, err := getBoolSafe(ctx, s, SettingKeyEnabled); err != nil {
		return Config{}, err
	} else {
		cfg.Enabled = v
	}
	if v, err := getBoolSafe(ctx, s, SettingKeyRequireDelivery); err != nil {
		return Config{}, err
	} else {
		cfg.RequireDelivery = v
	}

	// 邮件渠道。
	mail, err := loadMail(ctx, s)
	if err != nil {
		return Config{}, err
	}
	cfg.Mail = mail

	// 短信渠道。
	sms, err := loadSMS(ctx, s)
	if err != nil {
		return Config{}, err
	}
	cfg.SMS = sms

	// 环境变量应急覆盖（最高优先级）。
	if v := envOverride("NOTIFY_MAIL_PASSWORD"); v != "" {
		cfg.Mail.Password = v
	}
	if v := envOverride("NOTIFY_SMS_ACCESS_KEY_SECRET"); v != "" {
		cfg.SMS.AccessKeySecret = v
	}
	return cfg, nil
}

func loadMail(ctx context.Context, s *gsetting.Setting) (MailConfig, error) {
	var m MailConfig
	var err error
	if m.Enabled, err = getBoolSafe(ctx, s, SettingKeyMailEnabled); err != nil {
		return m, err
	}
	if m.Host, err = getStrSafe(ctx, s, SettingKeyMailHost); err != nil {
		return m, err
	}
	if m.Port, err = getIntSafe(ctx, s, SettingKeyMailPort); err != nil {
		return m, err
	}
	if m.Username, err = getStrSafe(ctx, s, SettingKeyMailUsername); err != nil {
		return m, err
	}
	if m.Password, err = getStrSafe(ctx, s, SettingKeyMailPassword); err != nil {
		return m, err
	}
	if m.From, err = getStrSafe(ctx, s, SettingKeyMailFrom); err != nil {
		return m, err
	}
	if m.TLS, err = getBoolSafe(ctx, s, SettingKeyMailTLS); err != nil {
		return m, err
	}
	return m, nil
}

func loadSMS(ctx context.Context, s *gsetting.Setting) (SMSConfig, error) {
	var c SMSConfig
	var err error
	if c.Enabled, err = getBoolSafe(ctx, s, SettingKeySMSEnabled); err != nil {
		return c, err
	}
	if c.Provider, err = getStrSafe(ctx, s, SettingKeySMSProvider); err != nil {
		return c, err
	}
	if c.AccessKeyID, err = getStrSafe(ctx, s, SettingKeySMSAccessKeyID); err != nil {
		return c, err
	}
	if c.AccessKeySecret, err = getStrSafe(ctx, s, SettingKeySMSAccessKeySecret); err != nil {
		return c, err
	}
	if c.SignName, err = getStrSafe(ctx, s, SettingKeySMSSignName); err != nil {
		return c, err
	}
	if c.SdkAppID, err = getStrSafe(ctx, s, SettingKeySMSSdkAppID); err != nil {
		return c, err
	}
	if c.TemplateCode, err = getStrSafe(ctx, s, SettingKeySMSTemplateCode); err != nil {
		return c, err
	}
	if c.Region, err = getStrSafe(ctx, s, SettingKeySMSRegion); err != nil {
		return c, err
	}
	return c, nil
}

// getStrSafe 读取字符串键；键不存在/解密失败一律视为空串（不视为错误）。
func getStrSafe(ctx context.Context, s *gsetting.Setting, key string) (string, error) {
	v, err := s.GetString(ctx, SettingGroup, key)
	if err != nil {
		// gsetting 的 Get 在「行不存在」时返回零值实体、不报错；报错基本是 db 故障。
		return "", err
	}
	return v, nil
}

func getBoolSafe(ctx context.Context, s *gsetting.Setting, key string) (bool, error) {
	v, err := s.GetBool(ctx, SettingGroup, key)
	if err != nil {
		// 行不存在时 gsetting 返回空串 -> ParseBool 报错；按 false 处理（安全默认）。
		return false, nil
	}
	return v, nil
}

func getIntSafe(ctx context.Context, s *gsetting.Setting, key string) (int, error) {
	v, err := s.GetInt64(ctx, SettingGroup, key)
	if err != nil {
		// 行不存在返回空串 -> ParseInt 报错；按 0 处理。
		return 0, nil
	}
	return int(v), nil
}

// SaveConfig 把 Config 逐字段写回 gsetting 分组 notify。
//
// 敏感字段（邮件密码、短信 Secret）以 encrypted=true 加密落库（见设计 §4.1）。
func SaveConfig(ctx context.Context, db gormdao.DBGetter, cfg Config) error {
	s, err := settingFor(db)
	if err != nil {
		return fmt.Errorf("notify: init gsetting: %w", err)
	}
	return saveTo(ctx, s, cfg)
}

func saveTo(ctx context.Context, s *gsetting.Setting, cfg Config) error {
	admin := gsetting.GetAdmin()
	type entry struct {
		key       string
		valueType gsetting.ValueType
		value     string
		encrypted bool
		desc      string
	}
	b := func(v bool) string {
		if v {
			return "true"
		}
		return "false"
	}
	entries := []entry{
		{SettingKeyEnabled, gsetting.ValueTypeBool, b(cfg.Enabled), false, "总闸：关闭=不发任何通知（回退为响应回显一次明文）；开启=按渠道投递"},
		{SettingKeyRequireDelivery, gsetting.ValueTypeBool, b(cfg.RequireDelivery), false, "投递失败语义：开启=投递失败报错（需重置重试）；关闭=回退为回显明文"},

		{SettingKeyMailEnabled, gsetting.ValueTypeBool, b(cfg.Mail.Enabled), false, "邮件渠道开关（总闸开时生效）"},
		{SettingKeyMailHost, gsetting.ValueTypeString, cfg.Mail.Host, false, "SMTP 服务器地址"},
		{SettingKeyMailPort, gsetting.ValueTypeInt, fmt.Sprintf("%d", cfg.Mail.Port), false, "SMTP 服务器端口（如 587）"},
		{SettingKeyMailUsername, gsetting.ValueTypeString, cfg.Mail.Username, false, "SMTP 认证用户名"},
		{SettingKeyMailPassword, gsetting.ValueTypeString, cfg.Mail.Password, true, "SMTP 认证密码（加密存储；可用环境变量 NOTIFY_MAIL_PASSWORD 覆盖）"},
		{SettingKeyMailFrom, gsetting.ValueTypeString, cfg.Mail.From, false, "发件人，形如 IAM <no-reply@example.com>"},
		{SettingKeyMailTLS, gsetting.ValueTypeBool, b(cfg.Mail.TLS), false, "是否使用 SMTPS/STARTTLS"},

		{SettingKeySMSEnabled, gsetting.ValueTypeBool, b(cfg.SMS.Enabled), false, "短信渠道开关（总闸开时生效）"},
		{SettingKeySMSProvider, gsetting.ValueTypeString, cfg.SMS.Provider, false, "短信供应商标识（tencent；空=不启用）"},
		{SettingKeySMSAccessKeyID, gsetting.ValueTypeString, cfg.SMS.AccessKeyID, false, "短信供应商 AccessKeyID（腾讯云 SecretID）"},
		{SettingKeySMSAccessKeySecret, gsetting.ValueTypeString, cfg.SMS.AccessKeySecret, true, "短信供应商 AccessKeySecret（腾讯云 SecretKey，加密存储；可用环境变量 NOTIFY_SMS_ACCESS_KEY_SECRET 覆盖）"},
		{SettingKeySMSSignName, gsetting.ValueTypeString, cfg.SMS.SignName, false, "短信签名"},
		{SettingKeySMSSdkAppID, gsetting.ValueTypeString, cfg.SMS.SdkAppID, false, "短信应用 SDKAppID（腾讯云必填，如 1400000000）"},
		{SettingKeySMSTemplateCode, gsetting.ValueTypeString, cfg.SMS.TemplateCode, false, "短信模板 ID"},
		{SettingKeySMSRegion, gsetting.ValueTypeString, cfg.SMS.Region, false, "短信供应商地域（腾讯云默认 ap-guangzhou）"},
	}
	for _, e := range entries {
		// gsetting 不接受空值（errValueEmpty）；空字段跳过不写，读取端按零值处理即可。
		if e.value == "" {
			continue
		}
		// Create 对同 (group,key) 走 upsert；Encrypted=true 时 gsetting 内部加密落库。
		if err := admin.Create(ctx, &gsetting.CreateReq{
			Group:       SettingGroup,
			Key:         e.key,
			ValueType:   string(e.valueType),
			Value:       e.value,
			Encrypted:   e.encrypted,
			Description: e.desc,
		}); err != nil {
			return fmt.Errorf("notify: save %s: %w", e.key, err)
		}
	}
	return nil
}

// DeliverTempPasswordViaSetting 是业务投递点的最小集成入口：现读 gsetting → 现构造 Notifier → 投递。
//
// 语义（见设计 §5.3 / §8）：配置读失败 / 构造失败一律降级为「回显明文」，绝不阻断主流程；
// 不依赖任何进程级单例（gsetting 是唯一事实源）。
func DeliverTempPasswordViaSetting(ctx context.Context, db gormdao.DBGetter, to Recipient, password string) (DeliveryOutcome, error) {
	cfg, err := LoadConfig(ctx, db)
	if err != nil {
		// 配置中心不可用：降级回显，不报错。
		return DeliveryOutcome{EchoPassword: password}, nil
	}
	n, err := FromConfig(cfg)
	if err != nil {
		// 配置非法（如 enabled=true 但渠道全关）：降级回显，不报错。
		return DeliveryOutcome{EchoPassword: password}, nil
	}
	return n.DeliverTempPassword(ctx, to, password)
}
