package code

import "github.com/morehao/golib/gerror"

const (
	DepartmentCreateError      = 100400
	DepartmentDeleteError      = 100401
	DepartmentUpdateError      = 100402
	DepartmentGetDetailError   = 100403
	DepartmentGetPageListError = 100404
	DepartmentNotExistError    = 100405
)

var departmentErrorMsgMap = gerror.CodeMsgMap{
	DepartmentCreateError:      "创建部门失败",
	DepartmentDeleteError:      "删除部门失败",
	DepartmentUpdateError:      "修改部门失败",
	DepartmentGetDetailError:   "查看部门详情失败",
	DepartmentGetPageListError: "查看部门列表失败",
	DepartmentNotExistError:    "部门不存在",
}