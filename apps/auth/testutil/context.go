package testutil

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/testsetup"
	"github.com/morehao/golib/biz/gcontext"
	"github.com/morehao/golib/biz/gcontext/gincontext"
)

// WithIamContext 构造一个把「用户 + 其租户作用域」注入测试上下文的选项。
//
// 返回 testsetup.CtxOption：历史上这里是 golib `biz/testkit` 的 Option 别名，golib v1.34.5
// 删除 `biz/testkit` 后改为本仓库 testsetup 自带的选项类型，调用方写法不变。
func WithIamContext(userID string) testsetup.CtxOption {
	return func(gc *gin.Context) {
		// 测试助手按自然人反查其归属租户，此刻租户未知：显式声明「全部租户」作用域。
		user, err := dao.NewUserDao().GetByID(dbclient.CrossTenantContext(context.Background()), userID)
		if err != nil {
			panic(err)
		}
		if user == nil || user.ID == "" {
			panic("user not found")
		}

		gc.Set(gcontext.KeyUserID, user.ID)
		// 租户作用域：类型化值 + gin Keys 投影，与生产中间件写法完全一致。
		gincontext.SetTenantScope(gc, gcontext.CurrentScope(user.TenantID))
		gc.Set(gcontext.KeyPersonID, user.PersonID)
	}
}
