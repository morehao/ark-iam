package dtoapplication

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/morehao/ark-iam/pkg/model"
)

// TestEnumFieldsJSONCompatibility 具名枚举类型的 JSON 向下兼容（AGENTS.md 硬规则 4）：
// 底层是 string，序列化仍是普通 JSON 字符串；反序列化也不校验值域
// （值域校验归 service 白名单，而非类型系统）——前端与既有客户端无感知。
func TestEnumFieldsJSONCompatibility(t *testing.T) {
	resp := ApplicationDetailResp{
		AppID:  "a1",
		Source: model.AppSourceThirdParty,
		Status: model.AppStatusEnable,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"status":"enable"`) {
		t.Fatalf("status 应为普通字符串, got %s", b)
	}
	if !strings.Contains(string(b), `"source":"third_party"`) {
		t.Fatalf("source 应为普通字符串, got %s", b)
	}

	// 非法值必须能绑定成功（否则会变成 400 而不是 service 的功能级错误码）
	var got ApplicationDetailResp
	if err := json.Unmarshal([]byte(`{"appID":"a1","status":"totally-illegal","source":"x"}`), &got); err != nil {
		t.Fatalf("反序列化不应因值域失败: %v", err)
	}
	if got.Status != model.AppStatus("totally-illegal") {
		t.Fatalf("非法值应原样保留待 service 校验, got %q", got.Status)
	}
}
