package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// TestNotifyConfigParses 校验 notify 配置段能正确解析（三层开关 + 渠道字段）。
func TestNotifyConfigParses(t *testing.T) {
	const raw = `
notify:
  enabled: true
  requireDelivery: true
  mail:
    enabled: true
    host: smtp.example.com
    port: 587
    username: u
    password: p
    from: "IAM <no-reply@example.com>"
    tls: true
  sms:
    enabled: true
    provider: tencent
    accessKeyID: ak
    accessKeySecret: sk
    signName: ArkIAM
    templateCode: "12345"
    region: ap-guangzhou
`
	var c Config
	if err := yaml.Unmarshal([]byte(raw), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !c.Notify.Enabled || !c.Notify.RequireDelivery {
		t.Fatalf("notify toggles wrong: %+v", c.Notify)
	}
	if !c.Notify.Mail.Enabled || c.Notify.Mail.Host != "smtp.example.com" || c.Notify.Mail.Port != 587 {
		t.Fatalf("mail config wrong: %+v", c.Notify.Mail)
	}
	if !c.Notify.SMS.Enabled || c.Notify.SMS.Provider != "tencent" || c.Notify.SMS.TemplateCode != "12345" {
		t.Fatalf("sms config wrong: %+v", c.Notify.SMS)
	}
}
