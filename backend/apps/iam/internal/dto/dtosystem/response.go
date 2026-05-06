package dtosystem

import (
	"github.com/morehao/ark-iam/iam/object/objsystem"
	"github.com/morehao/golib/biz/gobject"
)

type SystemCreateResp struct {
	SystemID uint `json:"systemID"` // 自增ID
}

type SystemDetailResp struct {
	SystemID uint `json:"systemID"` // 自增ID
	objsystem.SystemBaseInfo
	gobject.OperatorBaseInfo
}

type SystemPageListItem struct {
	SystemID uint `json:"systemID"` // 自增ID
	objsystem.SystemBaseInfo
	gobject.OperatorBaseInfo
}

type SystemPageListResp struct {
	List  []SystemPageListItem `json:"list"`  // 数据列表
	Total int64               `json:"total"` // 数据总条数
}