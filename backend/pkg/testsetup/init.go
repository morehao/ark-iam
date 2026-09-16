package testsetup

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/morehao/golib/biz/testkit"
	"github.com/morehao/golib/gconstant"
	"github.com/morehao/golib/gutil"
)

type Initializer = testkit.Initializer
type InitializerFunc = testkit.InitializerFunc

var (
	mu                  sync.RWMutex
	initializerCreators = map[string]InitializerFunc{
		AppNameDemo: newDemoappInitializer,
		AppNameAuth: newAuthappInitializer,
	}
	registeredApps = make(map[string]bool)
)

func init() {
	gin.SetMode(gin.TestMode)
}

func RegisterApp(appName string, initFunc InitializerFunc) {
	mu.Lock()
	defer mu.Unlock()
	if registeredApps[appName] {
		return
	}
	testkit.RegisterInitializer(appName, initFunc)
	registeredApps[appName] = true
}

func Initialize(appName string) {
	mu.Lock()
	if !registeredApps[appName] {
		if creator, ok := initializerCreators[appName]; ok {
			testkit.RegisterInitializer(appName, creator)
			registeredApps[appName] = true
		}
	}
	mu.Unlock()
	testkit.Initialize(appName)
}

func Close(appName string) {
	testkit.Close(appName)
}

func Init(appName string) {
	Initialize(appName)
}

func Done(appName string) {
	Close(appName)
}

func NewCtx(opts ...testkit.Option) *gin.Context {
	ginCtx, engine := gin.CreateTestContext(nil)
	// 与生产引擎保持一致：开启 ContextWithFallback，gin 上下文才能当作 context.Context
	// 使用（取租户作用域、透传 Done/Err），否则类型化作用域在 gin ctx 上不可见。
	engine.ContextWithFallback = true
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
