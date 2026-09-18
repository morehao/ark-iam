package model

import (
	"reflect"
	"strings"
	"testing"
)

// hardenedSingleColumnTypes 汇总 model 层「每字段一个具名类型」的字典枚举与 JSON 载具类型：
// 17 个枚举（语义型 6 个 + 字段级开关型 11 个）+ 10 个载具类型。
//
// 类型按所属实体分散声明在各自的 model 文件（person.go/user.go/domain.go/application.go/
// application_client.go/connector.go/menu.go/user_identity.go/jsontypes.go），**刻意互不共用**：
// 跨字段误赋值必须编译不过，取值词汇统一为 active/suspended、normal/must_change、owner/normal、
// unverified/verified 与 enable/disable。本文件是这些不变式的唯一集中校验点。
func hardenedSingleColumnTypes() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(PersonStatus("")),
		reflect.TypeOf(UserStatus("")),
		reflect.TypeOf(PasswordStatus("")),
		reflect.TypeOf(OwnerType("")),
		reflect.TypeOf(DomainVerificationStatus("")),
		reflect.TypeOf(EmailVerificationState("")),
		reflect.TypeOf(AppPersonCreateTenantPolicy("")),
		reflect.TypeOf(AppJoinByInvitePolicy("")),
		reflect.TypeOf(ClientPKCEPolicy("")),
		reflect.TypeOf(ClientAuthTimeClaimPolicy("")),
		reflect.TypeOf(ConnectorAutoCreateUserFlag("")),
		reflect.TypeOf(ConnectorAccountLinkFlag("")),
		reflect.TypeOf(ConnectorSyncProfileFlag("")),
		reflect.TypeOf(ConnectorTokenStorageFlag("")),
		reflect.TypeOf(MenuHiddenFlag("")),
		reflect.TypeOf(MenuExternalLinkFlag("")),
		reflect.TypeOf(MenuKeepAliveFlag("")),
		reflect.TypeOf(RedirectURIList(nil)),
		reflect.TypeOf(PostLogoutRedirectURIList(nil)),
		reflect.TypeOf(AllowedOriginList(nil)),
		reflect.TypeOf(DefaultScopeList(nil)),
		reflect.TypeOf(ScopeList(nil)),
		reflect.TypeOf(AuthMethodList(nil)),
		reflect.TypeOf(ConnectorScopeList(nil)),
		reflect.TypeOf(ResponseTypeList(nil)),
		reflect.TypeOf(GrantTypeList(nil)),
		reflect.TypeOf(RoleTemplateItemList(nil)),
	}
}

// 枚举/载具类型必须与 DB 列一一对应：同一具名类型被两个不同列引用即失败（AC-10）。
func TestHardenedTypesAreColumnExclusive(t *testing.T) {
	registered := make(map[reflect.Type]struct{})
	for _, typ := range hardenedSingleColumnTypes() {
		registered[typ] = struct{}{}
	}

	usedBy := make(map[reflect.Type][]string)
	for _, entity := range AllEntities() {
		entityType := reflect.TypeOf(entity)
		if entityType.Kind() == reflect.Ptr {
			entityType = entityType.Elem()
		}
		tableName := tableNameOf(entity)
		for i := 0; i < entityType.NumField(); i++ {
			field := entityType.Field(i)
			column := gormTagValue(field.Tag.Get("gorm"), "column")
			if column == "" {
				continue
			}
			fieldType := field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			if _, ok := registered[fieldType]; !ok {
				continue
			}
			usedBy[fieldType] = append(usedBy[fieldType], tableName+"."+column)
		}
	}

	for typ, columns := range usedBy {
		if len(columns) > 1 {
			t.Fatalf("%s 被多个列引用：%s", typ, strings.Join(columns, ", "))
		}
	}
}

func TestHardenedTypeKindsAreStringOrSlice(t *testing.T) {
	for _, typ := range hardenedSingleColumnTypes() {
		switch typ.Kind() {
		case reflect.String, reflect.Slice:
		default:
			t.Fatalf("%s 期望 string 或 slice，实际 %s", typ, typ.Kind())
		}
	}
}

func TestFieldFlagConstantsUseEnableDisable(t *testing.T) {
	pairs := map[string][2]string{
		"AppPersonCreateTenantPolicy": {string(AppPersonCreateTenantPolicyEnable), string(AppPersonCreateTenantPolicyDisable)},
		"AppJoinByInvitePolicy":       {string(AppJoinByInvitePolicyEnable), string(AppJoinByInvitePolicyDisable)},
		"ClientPKCEPolicy":            {string(ClientPKCEPolicyEnable), string(ClientPKCEPolicyDisable)},
		"ClientAuthTimeClaimPolicy":   {string(ClientAuthTimeClaimPolicyEnable), string(ClientAuthTimeClaimPolicyDisable)},
		"ConnectorAutoCreateUserFlag": {string(ConnectorAutoCreateUserFlagEnable), string(ConnectorAutoCreateUserFlagDisable)},
		"ConnectorAccountLinkFlag":    {string(ConnectorAccountLinkFlagEnable), string(ConnectorAccountLinkFlagDisable)},
		"ConnectorSyncProfileFlag":    {string(ConnectorSyncProfileFlagEnable), string(ConnectorSyncProfileFlagDisable)},
		"ConnectorTokenStorageFlag":   {string(ConnectorTokenStorageFlagEnable), string(ConnectorTokenStorageFlagDisable)},
		"MenuHiddenFlag":              {string(MenuHiddenFlagEnable), string(MenuHiddenFlagDisable)},
		"MenuExternalLinkFlag":        {string(MenuExternalLinkFlagEnable), string(MenuExternalLinkFlagDisable)},
		"MenuKeepAliveFlag":           {string(MenuKeepAliveFlagEnable), string(MenuKeepAliveFlagDisable)},
	}
	for name, pair := range pairs {
		if pair[0] != "enable" || pair[1] != "disable" {
			t.Fatalf("%s 取值应为 enable/disable，实际 %q/%q", name, pair[0], pair[1])
		}
	}
}

func TestSemanticEnumConstantsAreDistinctAndNonEmpty(t *testing.T) {
	pairs := map[string][2]string{
		"PersonStatus":             {string(PersonStatusActive), string(PersonStatusSuspended)},
		"UserStatus":               {string(UserStatusActive), string(UserStatusSuspended)},
		"PasswordStatus":           {string(PasswordStatusNormal), string(PasswordStatusMustChange)},
		"OwnerType":                {string(OwnerTypeOwner), string(OwnerTypeNormal)},
		"DomainVerificationStatus": {string(DomainVerificationUnverified), string(DomainVerificationVerified)},
		"EmailVerificationState":   {string(EmailVerificationVerified), string(EmailVerificationUnverified)},
	}
	for name, pair := range pairs {
		if pair[0] == "" || pair[1] == "" || pair[0] == pair[1] {
			t.Fatalf("%s 常量必须非空且互异，实际 %q/%q", name, pair[0], pair[1])
		}
	}
}

func tableNameOf(entity any) string {
	if namer, ok := entity.(interface{ TableName() string }); ok {
		return namer.TableName()
	}
	return reflect.TypeOf(entity).String()
}

func gormTagValue(tag, key string) string {
	for _, part := range strings.Split(tag, ";") {
		if strings.HasPrefix(part, key+":") {
			return strings.TrimPrefix(part, key+":")
		}
	}
	return ""
}

// TestHardenedColumnsUseNamedEnums 固化 16 个 bool 列改造后的「表.列 → 具名枚举」绑定：
// 列名（如 is_suspended/is_owner/is_verified 已改名）或字段类型回退成 bool 都必须在这里失败。
func TestHardenedColumnsUseNamedEnums(t *testing.T) {
	want := map[string]reflect.Type{
		TableNamePerson + ".password_status":                 reflect.TypeOf(PasswordStatus("")),
		TableNamePerson + ".status":                          reflect.TypeOf(PersonStatus("")),
		TableNameUser + ".status":                            reflect.TypeOf(UserStatus("")),
		TableNameUser + ".owner_type":                        reflect.TypeOf(OwnerType("")),
		TableNameDomain + ".verification_status":             reflect.TypeOf(DomainVerificationStatus("")),
		TableNameApplication + ".allow_person_create_tenant": reflect.TypeOf(AppPersonCreateTenantPolicy("")),
		TableNameApplication + ".allow_join_by_invite":       reflect.TypeOf(AppJoinByInvitePolicy("")),
		TableNameApplicationClient + ".require_pkce":         reflect.TypeOf(ClientPKCEPolicy("")),
		TableNameApplicationClient + ".require_auth_time":    reflect.TypeOf(ClientAuthTimeClaimPolicy("")),
		TableNameConnector + ".allow_auto_create_user":       reflect.TypeOf(ConnectorAutoCreateUserFlag("")),
		TableNameConnector + ".allow_account_link":           reflect.TypeOf(ConnectorAccountLinkFlag("")),
		TableNameConnector + ".sync_profile":                 reflect.TypeOf(ConnectorSyncProfileFlag("")),
		TableNameConnector + ".enable_token_storage":         reflect.TypeOf(ConnectorTokenStorageFlag("")),
		TableNameMenu + ".hidden":                            reflect.TypeOf(MenuHiddenFlag("")),
		TableNameMenu + ".external_link":                     reflect.TypeOf(MenuExternalLinkFlag("")),
		TableNameMenu + ".keep_alive":                        reflect.TypeOf(MenuKeepAliveFlag("")),
	}

	got := make(map[string]reflect.Type, len(want))
	for _, entity := range AllEntities() {
		entityType := reflect.TypeOf(entity)
		if entityType.Kind() == reflect.Ptr {
			entityType = entityType.Elem()
		}
		tableName := tableNameOf(entity)
		for i := 0; i < entityType.NumField(); i++ {
			field := entityType.Field(i)
			column := gormTagValue(field.Tag.Get("gorm"), "column")
			key := tableName + "." + column
			if _, ok := want[key]; !ok {
				continue
			}
			got[key] = field.Type
			if tag := field.Tag.Get("gorm"); !strings.Contains(tag, "type:varchar(16)") {
				t.Errorf("%s 应为 varchar(16) 枚举列，实际 tag: %s", key, tag)
			}
		}
	}
	for key, typ := range want {
		if got[key] != typ {
			t.Errorf("%s 字段类型 = %v, want %v", key, got[key], typ)
		}
	}
}
