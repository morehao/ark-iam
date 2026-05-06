package code

import "github.com/morehao/golib/gerror"

const (
	MenuCreateError      = 100600
	MenuDeleteError      = 100601
	MenuUpdateError      = 100602
	MenuGetDetailError   = 100603
	MenuGetPageListError = 100604
	MenuNotExistError    = 100605
)

var menuErrorMsgMap = gerror.CodeMsgMap{
	MenuCreateError:      "创建菜单失败",
	MenuDeleteError:      "删除菜单失败",
	MenuUpdateError:      "修改菜单失败",
	MenuGetDetailError:   "查看菜单详情失败",
	MenuGetPageListError: "查看菜单列表失败",
	MenuNotExistError:    "菜单不存在",
}