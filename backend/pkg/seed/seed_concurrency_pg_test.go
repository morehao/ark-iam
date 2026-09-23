//go:build pg

package seed_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/morehao/golib/gcrypto"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
)

// TestBootstrapConcurrentOnlyOneWins 并发引导只允许一次真实执行。
//
// **为什么必须在 PG 上测**：互斥完全依赖 `pg_advisory_xact_lock`
// （`lockSeed` 对非 postgres dialector 直接 no-op，见 seed.go），SQLite 上这个用例
// 无论实现是否正确都会通过——那是一条毫无证据力的绿灯。真实部署是多副本/多次点击提交，
// 这条路径必须用真库验证。
//
// 失效时会怎样（这就是本用例要防的）：两个事务都读到"未初始化"，于是都去插入平台租户。
// 唯一索引会让其中一个**报数据库错误**而不是返回 `StatusAlreadyInitialized`，
// 于是运维在页面上看到的是 500（`107003`），而库里已经有一份写好的数据——
// "重复提交只报已初始化"这个交互承诺当场失效。因此本用例断言的核心是
// **"恰好一个 created、其余全部 already_initialized、零 error"**，三者缺一不可。
func TestBootstrapConcurrentOnlyOneWins(t *testing.T) {
	const concurrency = 8

	db, err := gorm.Open(postgres.Open(pgTestDSN(t)), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	// 让连接池真的能同时开出 N 条连接，否则并发在池里被串行化，测不到锁。
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(concurrency)
	sqlDB.SetMaxIdleConns(concurrency)

	dropAll := func() {
		for _, tbl := range []string{
			"user_role", "role_menu", "tenant_application",
			"application_client_secret", "application_client", "menu",
			"role", "user_identity", "user_login_log",
			"department_user", "department", "tenant_user", "person",
			"refresh_token", "session", "audit_log", "api_key", "connector",
			"domain", "system", "log", "application", "tenant",
		} {
			_ = db.Exec("DROP TABLE IF EXISTS " + tbl).Error
		}
	}
	dropAll()
	t.Cleanup(dropAll)

	if err := model.AutoMigrateAll(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	hash, err := gcrypto.GeneratePasswordHash("Admin123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}

	type outcome struct {
		idx    int
		rep    seed.Report
		status seed.Status
		err    error
	}
	results := make([]outcome, concurrency)

	// 屏障：让 N 个 goroutine 尽量同时冲进 Bootstrap，制造真实争用。
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// 每个调用方都用**自己的**定义（不同管理员用户名与租户名）：
			// 这样"库里的数据属于哪个调用方"可以反查，能证明赢家只有一份数据、
			// 而不是各写各的拼在一起。
			def := seed.Definition{
				TenantName:        fmt.Sprintf("并发租户-%d", i),
				AdminUsername:     fmt.Sprintf("admin-%d", i),
				AdminName:         fmt.Sprintf("并发管理员-%d", i),
				AdminEmail:        fmt.Sprintf("admin-%d@example.com", i),
				AdminPasswordHash: hash,
			}
			<-start
			rep, status, err := seed.Bootstrap(context.Background(), db, def)
			results[i] = outcome{idx: i, rep: rep, status: status, err: err}
		}(i)
	}
	close(start)
	wg.Wait()

	var created, already, failed []int
	for _, r := range results {
		switch {
		case r.err != nil:
			failed = append(failed, r.idx)
		case r.status == seed.StatusCreated:
			created = append(created, r.idx)
		case r.status == seed.StatusAlreadyInitialized:
			already = append(already, r.idx)
		default:
			t.Errorf("调用方 %d 返回未预期的 status=%q（既非 created 也非 already_initialized）", r.idx, r.status)
		}
	}
	if len(failed) != 0 {
		for _, r := range results {
			if r.err != nil {
				t.Errorf("并发调用方 %d 报错（互斥失效会让重复提交变成数据库错误）：%v", r.idx, r.err)
			}
		}
	}
	if len(created) != 1 {
		t.Fatalf("created 调用方数 = %d（%v），want 恰好 1", len(created), created)
	}
	if len(already) != concurrency-1 {
		t.Errorf("already_initialized 调用方数 = %d，want %d", len(already), concurrency-1)
	}

	winner := created[0]

	// 只有赢家的 Report 有内容：输家的 Report 必须为空，
	// 否则意味着"先写了数据才发现已初始化"（事务边界破了）。
	for _, r := range results {
		if r.idx == winner {
			if len(r.rep.Changes) == 0 {
				t.Error("赢家的 Report 为空（页面拿不到变更清单）")
			}
			continue
		}
		if len(r.rep.Changes) != 0 {
			t.Errorf("输家 %d 也报告了 %d 条变更，want 0（数据由赢家独占写入）", r.idx, len(r.rep.Changes))
		}
	}

	// 只存在一份数据，且全部属于赢家。
	assertCount := func(tbl string, want int64) {
		t.Helper()
		var n int64
		if err := db.Table(tbl).Count(&n).Error; err != nil {
			t.Fatalf("count %s: %v", tbl, err)
		}
		if n != want {
			t.Errorf("table %s: want %d rows, got %d", tbl, want, n)
		}
	}
	assertCount("tenant", 1)
	assertCount("application", 2)
	assertCount("role", 2)
	assertCount("menu", 14)
	assertCount("person", 1)
	assertCount("tenant_user", 1)
	assertCount("application_client", 2)
	assertCount("user_role", 2)
	assertCount("role_menu", 14)
	assertCount("tenant_application", 2)
	assertCount("department", 1)
	assertCount("department_user", 1)

	var tenant model.TenantEntity
	if err := db.Where("code = ?", model.SeedPlatformTenantCode).First(&tenant).Error; err != nil {
		t.Fatalf("query platform tenant: %v", err)
	}
	if want := fmt.Sprintf("并发租户-%d", winner); tenant.Name != want {
		t.Errorf("平台租户名 = %q，want %q（赢家 %d 的数据；不符说明并发下发生了交叉写入）",
			tenant.Name, want, winner)
	}

	var person model.PersonEntity
	if err := db.Where("username = ?", fmt.Sprintf("admin-%d", winner)).First(&person).Error; err != nil {
		t.Fatalf("赢家 %d 的管理员账号不存在（并发下管理员与租户可能来自不同调用方）: %v", winner, err)
	}
	for i := 0; i < concurrency; i++ {
		if i == winner {
			continue
		}
		var other model.PersonEntity
		if err := db.Where("username = ?", fmt.Sprintf("admin-%d", i)).First(&other).Error; err == nil {
			t.Errorf("输家 %d 的管理员账号也被写入了（单次生效被破坏）", i)
		}
	}

	// 收尾：赢家之后再单独跑一次，必须稳定返回 already_initialized（锁释放后状态一致）。
	if _, status, err := seed.Bootstrap(context.Background(), db, seed.Definition{AdminPasswordHash: hash}); err != nil {
		t.Fatalf("并发结束后的再次引导报错: %v", err)
	} else if status != seed.StatusAlreadyInitialized {
		t.Errorf("并发结束后的再次引导 status = %q，want %q", status, seed.StatusAlreadyInitialized)
	}
}
