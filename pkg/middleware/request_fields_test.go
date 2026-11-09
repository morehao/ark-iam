package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/gconstant"
)

// TestWithRequestFieldsPropagatesRequestID 守护 C6：requestID 必须能在 controller 交给
// service 的**纯 context.Context** 上取到（glog 的 extra_keys 用 ctx.Value("requestID") 取值）。
//
// 边界调整后 controller 用 ctx.Request.Context() 下调，而 requestID 历史上只写在 gin Keys；
// 若不投影，service/dao 的日志会静默丢掉 requestID 字段。
func TestWithRequestFieldsPropagatesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name     string
		setupCtx func(c *gin.Context)
		want     string
	}{
		{
			name:     "gin Keys 的 requestID 投影到请求上下文",
			setupCtx: func(c *gin.Context) { c.Set(gcontext.KeyRequestID, "req-123") },
			want:     "req-123",
		},
		{
			name:     "回落 gconstant.KeyAppRequestID（老写法）",
			setupCtx: func(c *gin.Context) { c.Set(gconstant.KeyAppRequestID, "app-req-9") },
			want:     "app-req-9",
		},
		{
			name:     "都没有时保持为空，不写入",
			setupCtx: func(c *gin.Context) {},
			want:     "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runRequestFieldsProbe(t, tc.setupCtx)
			if got != tc.want {
				t.Fatalf("requestID = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestWithRequestFieldsKeepsGinGetter 投影后 gin 侧 GetRequestID 不受影响。
func TestWithRequestFieldsKeepsGinGetter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotGin string
	r := gin.New()
	// 在中间件链**之前**写入 requestID（模拟 AccessLog 的位置）。
	r.Use(func(c *gin.Context) { c.Set(gcontext.KeyRequestID, "req-777") }, WithRequestFields())
	r.GET("/x", func(c *gin.Context) {
		gotGin = gincontext.GetRequestID(c)
	})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))

	if gotGin != "req-777" {
		t.Fatalf("gin GetRequestID = %q, want req-777", gotGin)
	}
}

// runRequestFieldsProbe 走真实 HTTP 请求，读取 handler 里"纯 ctx"上的 requestID。
func runRequestFieldsProbe(t *testing.T, setup func(c *gin.Context)) string {
	t.Helper()
	var got string
	r := gin.New()
	if setup != nil {
		r.Use(setup)
	}
	r.Use(WithRequestFields())
	r.GET("/x", func(c *gin.Context) {
		// 与 controller 下调 service 的方式完全一致：取纯 ctx。
		got, _ = c.Request.Context().Value("requestID").(string)
	})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	return got
}
