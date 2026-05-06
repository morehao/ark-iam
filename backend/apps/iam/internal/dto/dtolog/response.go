package dtolog

import (
	"github.com/morehao/ark-iam/iam/object/objlog"
	"github.com/morehao/golib/biz/gobject"
)

type LogDetailResp struct {
	LogID uint `json:"logID"` // 日志ID
	objlog.LogBaseInfo
	gobject.OperatorBaseInfo
}

type LogPageListItem struct {
	LogID uint `json:"logID"` // 日志ID
	objlog.LogBaseInfo
	gobject.OperatorBaseInfo
}

type LogPageListResp struct {
	List  []LogPageListItem `json:"list"`  // 数据列表
	Total int64            `json:"total"` // 数据总条数
}