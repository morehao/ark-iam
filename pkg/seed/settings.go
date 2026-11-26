package seed

import (
	"context"
	"fmt"

	"github.com/morehao/ark-iam/pkg/loginsetting"
	"github.com/morehao/ark-iam/pkg/notify"
	"github.com/morehao/golib/gsetting"
	"gorm.io/gorm"
)

// RuntimeSettings supplies the initial configuration for installation.
type RuntimeSettings struct {
	Notify notify.Config
	Login  loginsetting.Config
}

func (s RuntimeSettings) seed(ctx context.Context, tx *gorm.DB) error {
	db := func(ctx context.Context) *gorm.DB { return tx.WithContext(ctx) }
	for _, group := range []string{notify.SettingGroup, loginsetting.SettingGroup} {
		var count int64
		if err := tx.Model(&gsetting.SettingEntity{}).Where(map[string]any{"group": group, "app_id": ""}).Count(&count).Error; err != nil {
			return fmt.Errorf("seed settings: count %s: %w", group, err)
		}
		if count != 0 {
			continue
		}
		if group == notify.SettingGroup {
			if err := notify.SaveConfig(ctx, db, s.Notify); err != nil {
				return err
			}
			continue
		}
		if err := loginsetting.Save(ctx, db, s.Login); err != nil {
			return err
		}
	}
	return nil
}
