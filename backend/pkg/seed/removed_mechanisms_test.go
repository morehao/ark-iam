package seed_test

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/seed"
)

// TestBootstrap_DeletedBuiltinMenuStaysDeleted 运维在控制台删除内置菜单后，菜单不得被"建回来"。
//
// 这是**替代**已删除的墓碑机制（menuSeedKeyRemoved）的行为用例。旧机制要解决的问题是
// "种子每次启动都执行 → 删除会被下次启动撤销"，因此软删行必须充当墓碑让种子跳过。
// 现在 L1 只在首次初始化执行一次、之后永久自锁，所以删除天然持久——本用例把这个
// 结构性保证钉住：即使再调一次 Bootstrap（不可能的真实路径，但接口仍暴露），也不得写入。
//
// 注意用例同时断言"第二次 Bootstrap 返回 already_initialized 且零写入"，因为
// "删除不被撤销"与"不再写任何数据"是同一件事的两面。
func TestBootstrap_DeletedBuiltinMenuStaysDeleted(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	if _, _, err := seed.Bootstrap(ctx, db, testDefinition(t)); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if got := countRows(t, db, model.TableNameMenu); got != 14 {
		t.Fatalf("首次引导后菜单数 = %d, want 14", got)
	}

	// 模拟控制台删除「菜单管理」：先解除授权绑定，再软删菜单行（与控制台同一路径）。
	var target model.MenuEntity
	if err := db.Where("seed_key = ?", "menu").First(&target).Error; err != nil {
		t.Fatalf("query menu: %v", err)
	}
	if err := db.Where("menu_id = ?", target.ID).Delete(&model.RoleMenuEntity{}).Error; err != nil {
		t.Fatalf("delete role_menu: %v", err)
	}
	if err := db.Delete(&target).Error; err != nil {
		t.Fatalf("soft delete menu: %v", err)
	}
	beforeCounts := map[string]int64{}
	for _, table := range []string{model.TableNameMenu, model.TableNameRoleMenu, "tenant", "application", "application_client"} {
		beforeCounts[table] = countRows(t, db, table)
	}

	// 二次引导：自锁，零写入，被删除的菜单不得复活
	rep, status, err := seed.Bootstrap(ctx, db, testDefinition(t))
	if err != nil {
		t.Fatalf("bootstrap (2nd): %v", err)
	}
	if status != seed.StatusAlreadyInitialized {
		t.Errorf("二次引导 status = %q, want %q", status, seed.StatusAlreadyInitialized)
	}
	if len(rep.Changes) != 0 {
		t.Errorf("二次引导必须零写入，实际 %d 条变更: %+v", len(rep.Changes), rep.Changes)
	}
	for table, want := range beforeCounts {
		if got := countRows(t, db, table); got != want {
			t.Errorf("二次引导后表 %s 行数 = %d, want %d（不得有任何写入）", table, got, want)
		}
	}
	// 被删除的菜单仍是"软删且未被重建"：活跃行少一个、全量行仍是 14。
	// 注意必须用 Model 查询才能带上软删条件（countRows 走 Table()，不带 DeletedAt 语义）。
	var activeMenus, allMenus int64
	if err := db.Model(&model.MenuEntity{}).Count(&activeMenus).Error; err != nil {
		t.Fatalf("count active menus: %v", err)
	}
	if activeMenus != 13 {
		t.Errorf("活跃菜单数 = %d, want 13（被删除的菜单不得复活）", activeMenus)
	}
	if err := db.Unscoped().Model(&model.MenuEntity{}).Count(&allMenus).Error; err != nil {
		t.Fatalf("count all menus: %v", err)
	}
	if allMenus != 14 {
		t.Errorf("全量菜单行数 = %d, want 14（既不复活也不重复建行）", allMenus)
	}
}

// TestSeedExportedSurfaceIsPinned 钉住 pkg/seed 的导出面：**只有一条写通道**。
//
// 这是"启动期零写入、写入只发生在 L1 引导"这一契约的结构性保证——只要导出面不变，
// 就不可能有第二个包外的写入入口。新增导出符号必须是有意的改动，故用测试强制 review。
func TestSeedExportedSurfaceIsPinned(t *testing.T) {
	got := exportedSurface(t)

	// 唯一的写通道 + 只读查询 + 只读清单
	want := []string{
		// 类型
		"BuiltinMenu", "Change", "ConsoleRedirects", "ConsolesConfig", "Definition", "Report", "Status",
		// 常量
		"ErrAdminPasswordHashRequired", "StatusAlreadyInitialized", "StatusCreated",
		// 函数
		//   Bootstrap        唯一写通道（L1 首次引导）
		//   IsInitialized    只读：库是否已引导
		//   SchemaReady      只读：表结构是否就绪
		//   BuiltinMenus     只读：内置菜单清单（make print-builtin-menus）
		//   Run / SeedIam    过渡期启动播种入口，P5（切断启动期播种）删除
		"Bootstrap", "BuiltinMenus", "IsInitialized", "Run", "SchemaReady", "SeedIam",
	}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("pkg/seed 导出面 = %v\nwant %v\n（写通道必须只有 Bootstrap；新增导出请同步本列表并说明理由）", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("pkg/seed 导出面 = %v\nwant %v", got, want)
		}
	}
}

// TestRemovedMechanismsLeaveNoResidue 四个跨版本机制必须**彻底**删除，不留半截实现。
//
// 判据是"不留可执行残留"：声明与调用都不允许存在。注释里作为历史说明提及是允许的
// （见 seed.go 与 pkg/model/seed_authority.go 的文件头），故这里只查 Go 语法层面
// 的声明/调用形态。
func TestRemovedMechanismsLeaveNoResidue(t *testing.T) {
	root := repoRoot(t)
	// token 由片段拼接，避免本文件自身命中
	forbidden := []string{
		"var retired" + "Menus",
		"func prune" + "RetiredMenus(",
		"var seed" + "Migrations",
		"func migrate" + "String(",
		"func reconcile" + "Fields(",
		"func menuSeedKey" + "Removed(",
		"func findMenuByApp" + "AndCode(",
		"func SeedOwns" + "Field(",
		"func SeedReconcile" + "Fields(",
		"SeedFieldRecon" + "cile",
		"SeedFieldMigrate" + "Once",
		"rep.migra" + "ted(",
		"rep.upd" + "ated(",
	}
	var offenders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "node_modules", ".gocache", "dist", "output":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, rErr := os.ReadFile(path)
		if rErr != nil {
			return rErr
		}
		content := string(data)
		for _, token := range forbidden {
			if strings.Contains(content, token) {
				rel, _ := filepath.Rel(root, path)
				offenders = append(offenders, rel+" 命中 "+token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("已删除的机制仍有残留（应彻底删除，不做兼容分支）:\n  %s", strings.Join(offenders, "\n  "))
	}

	// 退役清单文件本身也必须不存在
	if _, statErr := os.Stat(filepath.Join(root, "backend/pkg/seed/retired_menu.go")); statErr == nil {
		t.Error("backend/pkg/seed/retired_menu.go 仍存在：菜单下线不再需要版本级清单")
	}
}

// exportedSurface 解析 pkg/seed 的全部非测试源码，返回排序后的导出顶层符号名。
func exportedSurface(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse package dir: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("期望 1 个包，实际 %d 个", len(pkgs))
	}
	var names []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Recv == nil && d.Name.IsExported() {
						names = append(names, d.Name.Name)
					}
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						switch sp := spec.(type) {
						case *ast.TypeSpec:
							if sp.Name.IsExported() {
								names = append(names, sp.Name.Name)
							}
						case *ast.ValueSpec:
							for _, id := range sp.Names {
								if id.IsExported() {
									names = append(names, id.Name)
								}
							}
						}
					}
				}
			}
		}
	}
	sort.Strings(names)
	return names
}

// repoRoot 返回仓库根目录（测试的工作目录是 backend/pkg/seed）。
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatalf("repo root 判定失败（%s 下没有 AGENTS.md）: %v", root, err)
	}
	return root
}
