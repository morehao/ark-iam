package internal

import (
	"fmt"

	"github.com/morehao/golib/gsetting"
)

// initSetting 初始化运行时配置中心实例（golib gsetting），挂在 App 上供各消费者共享。
//
// 一进程一实例语义：页面写（platformadmin）与业务读（notify/security.login）在同一进程内
// 共用同一 *gsetting.Setting，避免 gsetting 包级 AdminAPI 被多次 New 覆盖（见设计 §10 R2）。
//
// 默认自动建表 core_setting（幂等 AutoMigrate）。若部署要求 DDL 由发布流程执行，可改为
// gsetting.New(a.dbGetter, gsetting.WithoutAutoMigrate()) 并在发布侧显式 gsetting.Migrate。
func (a *App) initSetting() error {
	if a.storeRaw == nil {
		return fmt.Errorf("sdk/server: init setting failed: db not initialized")
	}
	// gsetting.New 接收 *gorm.DB（内部按其 DBGetter 包装）；用实例底层连接，保证与业务同库。
	s, err := gsetting.New(a.storeRaw)
	if err != nil {
		return fmt.Errorf("sdk/server: init setting failed: %w", err)
	}
	a.setting = s
	return nil
}

// Setting 返回实例持有的运行时配置中心句柄；未初始化时为 nil。
func (a *App) Setting() *gsetting.Setting {
	return a.setting
}
