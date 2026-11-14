package internal

import (
	"os"

	pkgconfig "github.com/morehao/ark-iam/pkg/config"
	"github.com/morehao/ark-iam/pkg/notify"
)

// notifyConfigFrom 把 pkgconfig.NotifyConfig（配置解析产物）映射为 pkg/notify.Config（运行时）。
// 敏感字段优先从环境变量读（NOTIFY_MAIL_PASSWORD / NOTIFY_SMS_ACCESS_KEY_SECRET），
// 配置仅作回落（见执行计划 §1.2 敏感字段约定）。
func notifyConfigFrom(c pkgconfig.NotifyConfig) notify.Config {
	mailPassword := firstNonEmpty(envOr("NOTIFY_MAIL_PASSWORD", ""), c.Mail.Password)
	smsSecret := firstNonEmpty(envOr("NOTIFY_SMS_ACCESS_KEY_SECRET", ""), c.SMS.AccessKeySecret)

	return notify.Config{
		Enabled:         c.Enabled,
		RequireDelivery: c.RequireDelivery,
		Mail: notify.MailConfig{
			Enabled:  c.Mail.Enabled,
			Host:     c.Mail.Host,
			Port:     c.Mail.Port,
			Username: c.Mail.Username,
			Password: mailPassword,
			From:     c.Mail.From,
			TLS:      c.Mail.TLS,
		},
		SMS: notify.SMSConfig{
			Enabled:         c.SMS.Enabled,
			Provider:        c.SMS.Provider,
			AccessKeyID:     c.SMS.AccessKeyID,
			AccessKeySecret: smsSecret,
			SignName:        c.SMS.SignName,
			TemplateCode:    c.SMS.TemplateCode,
			Region:          c.SMS.Region,
		},
	}
}

// envOr 读环境变量，未设置时返回 def。
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// firstNonEmpty 返回首个非空字符串。
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
