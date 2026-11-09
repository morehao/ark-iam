package testsetup

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/morehao/ark-iam/pkg/object/objidentity"
	"github.com/morehao/golib/biz/gcontext"
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
