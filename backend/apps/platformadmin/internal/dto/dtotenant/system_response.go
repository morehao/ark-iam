package dtotenant

import (
	"github.com/morehao/ark-iam/pkg/iam/object/objaudit"
	"github.com/morehao/ark-iam/pkg/iam/object/objtenant"
	"github.com/morehao/golib/biz/gobject"
)

type SystemCreateResp struct {
	SystemID string `json:"systemID"` // 自增ID
}

type SystemDetailResp struct {
	SystemID string `json:"systemID"` // 自增ID
	objtenant.SystemBaseInfo
	gobject.OperatorBaseInfo
}

type SystemPageListItem struct {
	SystemID string `json:"systemID"` // 自增ID
	objtenant.SystemBaseInfo
	gobject.OperatorBaseInfo
}

type SystemPageListResp struct {
	List  []SystemPageListItem `json:"list"`  // 数据列表
	Total int64                `json:"total"` // 数据总条数
}

type LogDetailResp struct {
	LogID string `json:"logID"` // 日志ID
	objaudit.LogBaseInfo
	gobject.OperatorBaseInfo
}

type LogPageListItem struct {
	LogID string `json:"logID"` // 日志ID
	objaudit.LogBaseInfo
	gobject.OperatorBaseInfo
}

type LogPageListResp struct {
	List  []LogPageListItem `json:"list"`  // 数据列表
	Total int64             `json:"total"` // 数据总条数
}
