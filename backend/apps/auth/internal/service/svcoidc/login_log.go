package svcoidc

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/glog"
)

// insertOIDCLoginLog 是登录日志的落库点，抽成包级函数变量仅为单测注入"写库失败/DB 不可用"
// （与 svcauth.authLoginRecorder 同构的测试 seam，不引入 repository 接口）：
// 服务层直连 dao.NewUserLoginLogDao()，生产路径不依赖任何中间层。
var insertOIDCLoginLog = func(ctx *gin.Context, entity *model.UserLoginLogEntity) error {
	return dao.NewUserLoginLogDao().Insert(ctx, entity)
}

// recordOIDCLoginLog 记录一次「已确实成功」的 OIDC/SSO 登录（best-effort）。
//
// 为什么必须 best-effort：登录日志是旁路观测数据，任何写入异常都不得反转登录结论。
// DB 未初始化时 dao 拿到 nil *gorm.DB 会直接 panic（不是返回 error），故与
// pkg/audit.WriteAudit 同款：defer recover() 兜住 panic，写库失败只记 glog.Errorf。
//
// 作用域：user_login_log 含 tenant_id 列且不在 tenantScopeSkipTables 里，但租户隔离插件只注册
// query/update/delete 三个回调、**不覆盖 INSERT**（见 pkg/dbclient/tenant_scope.go 引用的
// golib gormplugin.ScopePlugin.Initialize/addScope 注释），租户字段由实体显式赋值，
// 因此这里直接 Insert(ctx, entity)，无需再声明作用域；SELECT 侧断言才需要显式作用域。
//
// 调用时机：只在授权票据已 done=true（将继续签发授权码）之后调用，被拒/待选租户/待改密
// 等未完成路径一律不写，避免把失败尝试记成成功登录。
func recordOIDCLoginLog(ctx *gin.Context, tenantID, userID, personID string) {
	if ctx == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf(ctx, "[svcoidc.recordOIDCLoginLog] panic recovered, tenantID:%s, userID:%s, personID:%s, panic:%v",
				tenantID, userID, personID, r)
		}
	}()

	entity := &model.UserLoginLogEntity{
		PersonID:  personID,
		TenantID:  tenantID,
		UserID:    userID,
		LoginType: model.LoginTypeOIDC,
		LoginIP:   gincontext.GetClientIP(ctx),
		UserAgent: ctx.GetHeader("User-Agent"),
		LoginTime: time.Now(),
		// 与既有写入点（svcauth.defaultRecordLoginLog）同口径：OIDC 登录由自然人本人发起，
		// 此时请求上下文尚未注入鉴权身份（没有 userID 可作 created_by），故留空。
		CreatedBy: "",
	}
	if err := insertOIDCLoginLog(ctx, entity); err != nil {
		glog.Errorf(ctx, "[svcoidc.recordOIDCLoginLog] insert login log fail, tenantID:%s, userID:%s, personID:%s, err:%v",
			tenantID, userID, personID, err)
	}
}
