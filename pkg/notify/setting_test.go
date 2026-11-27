package notify

import (
	"context"
	"testing"

	"github.com/morehao/golib/dbaccess/gormdao"
	"github.com/morehao/golib/gsetting"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) gormdao.DBGetter {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:notify_setting_"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := gsetting.Migrate(db); err != nil {
		t.Fatal(err)
	}
	return func(ctx context.Context) *gorm.DB { return db.WithContext(ctx) }
}

func TestSaveLoadRoundTrip(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	want := Config{
		Enabled:         true,
		RequireDelivery: true,
		Mail: MailConfig{
			Enabled:  true,
			Host:     "smtp.example.com",
			Port:     587,
			Username: "u",
			Password: "secret-pwd",
			From:     "IAM <no-reply@example.com>",
			TLS:      true,
		},
		SMS: SMSConfig{
			Enabled:         true,
			Provider:        "tencent",
			AccessKeyID:     "ak",
			AccessKeySecret: "sk-secret",
			SignName:        "sign",
			SdkAppID:        "1400000000",
			TemplateCode:    "123",
			Region:          "ap-guangzhou",
		},
	}
	if err := SaveConfig(ctx, db, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := LoadConfig(ctx, db)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != want {
		t.Fatalf("roundtrip mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestLoadEmptyReturnsZero(t *testing.T) {
	db := setupDB(t)
	got, err := LoadConfig(context.Background(), db)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != (Config{}) {
		t.Fatalf("expected zero config on empty, got %+v", got)
	}
}

func TestEnvOverrideSecret(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if err := SaveConfig(ctx, db, Config{Mail: MailConfig{Password: "fromdb"}, SMS: SMSConfig{AccessKeySecret: "dbsecret"}}); err != nil {
		t.Fatalf("save: %v", err)
	}
	t.Setenv("NOTIFY_MAIL_PASSWORD", "fromenv")
	t.Setenv("NOTIFY_SMS_ACCESS_KEY_SECRET", "envsecret")
	got, err := LoadConfig(ctx, db)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Mail.Password != "fromenv" || got.SMS.AccessKeySecret != "envsecret" {
		t.Fatalf("env override not applied: %+v", got)
	}
}

func TestDeliverViaSettingFallsBackToEcho(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	// 空配置（enabled=false）-> 回显明文，不报错。
	out, err := DeliverTempPasswordViaSetting(ctx, db, Recipient{Email: "a@b.com"}, "pwd123")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.Delivered {
		t.Fatal("should not be delivered when disabled")
	}
	if out.EchoPassword != "pwd123" {
		t.Fatalf("expected echo pwd123, got %q", out.EchoPassword)
	}
}
