package testsetup

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/gconstant"
	"github.com/morehao/golib/gutil"
)

// Initializer 是测试环境的初始化器：Initialize 建立依赖（配置/日志/DB/Redis），Close 释放资源。
//
// 这两个类型与 CtxOption 历史上是 golib `biz/testkit` 的别名。golib v1.34.5（#134）把测试
// 基建迁到 `gtest` 并删除了 `biz/testkit`，而 gtest 的新 API 以 `testing.TB` + AppBoot 为中心，
// 与本包「按 appName 注册、包内按需初始化」的调用方式不是一对一替代。这里改为本包自包含实现，
// 保持既有调用方（Initialize / Close / NewCtx / RegisterApp）零改动，也不再依赖已被上游删除的包。
type Initializer interface {
	Initialize() error
	Close() error
}

// InitializerFunc 按需构造 Initializer（首次 Initialize 时调用一次）。
type InitializerFunc func() (Initializer, error)

// CtxOption 在构造测试 Gin 上下文后追加定制（如注入租户/用户身份）。
type CtxOption func(ctx *gin.Context)

var (
	mu sync.RWMutex
	// initializerCreators 是「应用名 → 初始化器构造器」的注册表（内置 demo/auth，其余按需 RegisterApp）。
	initializerCreators = map[string]InitializerFunc{
		AppNameDemo: newDemoappInitializer,
		AppNameAuth: newAuthappInitializer,
	}
	// appInitializers 保存已初始化的实例，供 Close 释放。
	appInitializers = make(map[string]Initializer)
	// initOnce 保证每个应用名的初始化只执行一次。
	initOnce = make(map[string]*sync.Once)
)

func init() {
	gin.SetMode(gin.TestMode)
}

// RegisterApp 注册应用名到初始化函数的映射（已注册过则幂等跳过）。
func RegisterApp(appName string, initFunc InitializerFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := initializerCreators[appName]; exists {
		return
	}
	initializerCreators[appName] = initFunc
}

// Initialize 初始化指定应用名对应的测试环境（并发安全，仅首次生效）。
// 创建或初始化失败直接 panic——测试环境起不来时继续跑没有意义。
func Initialize(appName string) {
	once := onceFor(appName)

	once.Do(func() {
		mu.RLock()
		creator, ok := initializerCreators[appName]
		mu.RUnlock()
		if !ok {
			panic(fmt.Sprintf("app %s not registered", appName))
		}

		initializer, err := creator()
		if err != nil {
			panic(fmt.Sprintf("create initializer for app %s failed: %v", appName, err))
		}
		if err := initializer.Initialize(); err != nil {
			panic(fmt.Sprintf("initialize app %s failed: %v", appName, err))
		}

		mu.Lock()
		appInitializers[appName] = initializer
		mu.Unlock()
	})
}

// onceFor 取（必要时创建）指定应用名的 sync.Once。
func onceFor(appName string) *sync.Once {
	mu.Lock()
	defer mu.Unlock()

	once, ok := initOnce[appName]
	if !ok {
		once = &sync.Once{}
		initOnce[appName] = once
	}
	return once
}

// Close 关闭指定应用名的测试环境资源；appName 为空时关闭全部。
func Close(appName string) {
	if appName == "" {
		closeAll()
		return
	}

	mu.RLock()
	initializer, ok := appInitializers[appName]
	mu.RUnlock()
	if !ok {
		return
	}

	_ = initializer.Close()

	mu.Lock()
	defer mu.Unlock()
	delete(appInitializers, appName)
	delete(initOnce, appName)
}

// closeAll 关闭全部已初始化实例（用于 TestMain 收尾）。
func closeAll() {
	mu.Lock()
	defer mu.Unlock()

	for _, initializer := range appInitializers {
		_ = initializer.Close()
	}
	appInitializers = make(map[string]Initializer)
	initOnce = make(map[string]*sync.Once)
}

func Init(appName string) {
	Initialize(appName)
}

func Done(appName string) {
	Close(appName)
}

func NewCtx(opts ...CtxOption) *gin.Context {
	ginCtx, _ := gin.CreateTestContext(nil)
	// 与生产引擎保持一致：**不开启** ContextWithFallback。生产 controller 用
	// ctx.Request.Context() 做唯一一次转换，service/dao 只收纯 context.Context；
	// 夹具同样不能把 *gin.Context 当 context.Context 传下游，否则测试会掩盖改造遗漏。
	ginCtx.Request = &http.Request{
		URL:    &url.URL{},
		Header: http.Header{},
	}

	for _, opt := range opts {
		opt(ginCtx)
	}

	if _, exists := ginCtx.Get(gconstant.KeyAppRequestID); !exists {
		ginCtx.Set(gconstant.KeyAppRequestID, gutil.GenUUID())
	}

	return ginCtx
}

// RequestCtx 返回夹具 gin.Context 对应的纯请求上下文（等价于生产 controller 的
// ctx.Request.Context()），供测试调用只收 context.Context 的 service。
func RequestCtx(ginCtx *gin.Context) context.Context {
	if ginCtx == nil || ginCtx.Request == nil {
		return context.Background()
	}
	return ginCtx.Request.Context()
}

// OperatorFromCtx 按生产 controller 的方式从 gin 夹具取操作人，填 DTO 的 Operator 字段。
func OperatorFromCtx(ginCtx *gin.Context) objidentity.Operator {
	if ginCtx == nil {
		return objidentity.Operator{}
	}
	op := objidentity.Operator{
		UserID:   ginCtx.GetString(gcontext.KeyUserID),
		PersonID: ginCtx.GetString(gcontext.KeyPersonID),
	}
	if ginCtx.Request != nil {
		op.UserAgent = ginCtx.Request.UserAgent()
	}
	return op
}
