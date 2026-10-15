package code

import "testing"

// TestUserApplicationErrorCodesRegistered 锁定准入层（user_application）错误码的**注册完整性**。
//
// 为什么需要这条测试：`code.GetError(c)` 只是 `errorMap[c]` 取值，**未登记的码返回零值 Error**
// （`Code=0, Msg=""`）。此时业务错误经 `gincontext.Fail` 出去会变成
// `{"code":0,"msg":""}`——前端与调用方会把它当**成功**（这正是 e2e 在活栈上抓到的问题：
// I5 拒绝写入其实生效了，但响应码是 0）。而"用 `code.GetError(x).Error()` 与
// `code.GetError(x).Error()` 比较"的断言**恒真**（两边都是空串），单测抓不到，
// 因此这里直接钉住 Code/Msg 两个字段。
func TestUserApplicationErrorCodesRegistered(t *testing.T) {
	wantMsg := map[int]string{
		UserApplicationReplaceError:            "更新用户应用准入失败",
		UserApplicationGetListError:            "查看用户应用准入清单失败",
		UserApplicationNotSubscribedError:      "目标应用未订阅或已停用，不可准入",
		UserApplicationLastAdminForbiddenError: "撤销该准入会使租户失去最后一个内置管理员，已拒绝",
		AuthAppAccessDeniedError:               "你未被授权使用该应用，请联系所在租户的管理员分配",
	}
	for c, msg := range wantMsg {
		registered, ok := errorMap[c]
		if !ok {
			t.Errorf("错误码 %d 未注册（GetError 会返回零值，对外表现为成功）", c)
			continue
		}
		if registered.Code != c {
			t.Errorf("错误码 %d 的注册项 Code=%d，应为 %d", c, registered.Code, c)
		}
		if registered.Msg == "" {
			t.Errorf("错误码 %d 文案为空", c)
		}
		if registered.Msg != msg {
			t.Errorf("错误码 %d 文案 = %q, want %q", c, registered.Msg, msg)
		}
		got := GetError(c)
		if got.GetCode() != c || got.GetMsg() == "" {
			t.Errorf("GetError(%d) = {code:%d, msg:%q}，未取到已登记的错误", c, got.GetCode(), got.GetMsg())
		}
	}
}
