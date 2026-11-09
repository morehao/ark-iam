package dao

import (
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/golib/dbaccess/gormdao"
)

// DaoOption 定制 DAO 行为；目前仅支持测试注入自定义 DB。
// 统一的测试注入助手：各 DAO 构造器接收 opts ...DaoOption，
// 替代此前按 DAO 各自定义的 NewXXXDaoWithDB 变体。
type DaoOption func(*daoOptions)

type daoOptions struct {
	dbGetter gormdao.DBGetter
}

// WithDBGetter 注入自定义数据库访问器（测试专用）：
// 将被测 DAO 隔离到独立数据库（如内存 sqlite），其余走全局 iam 库的路径不受影响。
func WithDBGetter(getter gormdao.DBGetter) DaoOption {
	return func(o *daoOptions) {
		o.dbGetter = getter
	}
}

// resolveDBGetter 解析最终数据库访问器：指定了 WithDBGetter 则用之，否则回退全局 iam 库。
func resolveDBGetter(opts ...DaoOption) gormdao.DBGetter {
	o := &daoOptions{}
	for _, opt := range opts {
		opt(o)
	}
	if o.dbGetter != nil {
		return o.dbGetter
	}
	return dbclient.IamDB
}
