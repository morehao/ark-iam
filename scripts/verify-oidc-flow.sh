#!/bin/bash
set -euo pipefail

# ============================================================
# OIDC SSO 登录流程验证脚本
# 覆盖: authorize → login → callback → token → userinfo → logout
#
# 使用方式:
#   chmod +x scripts/verify-oidc-flow.sh
#   ./scripts/verify-oidc-flow.sh
#
# 前置条件:
#   1. 后端 IAM 服务运行在 localhost:8099
#   2. 数据库已插入测试 OAuth client (test-rp-client)
#   3. 测试用户已存在 (默认 admin/admin123)
#
# 可修改下方配置项适配不同环境
# ============================================================

# ==================== 配置项（按需修改） ====================
IAM_BASE="http://localhost:8099/v1/iam/oidc"   # OIDC Issuer 地址
CLIENT_ID="test-rp-client"                      # OAuth 客户端 ID
CLIENT_SECRET="my-test-client-secret"           # 客户端密钥
REDIRECT_URI="http://localhost:3001/"           # 回调地址（需在 client 中注册）
IDENTIFIER="admin"                              # 登录用户名/邮箱/手机号
PASSWORD="admin123"                             # 登录密码
STATE="test-state-123"                          # CSRF 防护 state 参数
NONCE="test-nonce-456"                          # 防重放 nonce 参数

# 绕过系统代理（所有请求均为 localhost，避免代理拦截）
export no_proxy="localhost,127.0.0.1"

# ==================== 颜色定义 ====================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# ==================== 临时文件 ====================
TMPDIR=$(mktemp -d)
HEADER_FILE="$TMPDIR/headers"
COOKIE_JAR="$TMPDIR/cookies"

cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

# ==================== 工具函数 ====================

# 输出 PASS 信息（绿色）
pass() { echo -e "  ${GREEN}PASS${NC}  $1"; }

# 输出 FAIL 信息（红色）并退出
fail() { echo -e "  ${RED}FAIL${NC}  $1"; exit 1; }

# 输出 INFO 信息（黄色）
info() { echo -e "  ${YELLOW}INFO${NC}  $1"; }

# 从响应头文件中提取指定 header 的值
get_header() {
    grep -i "^$1:" "$HEADER_FILE" | tail -1 | sed 's/^[^:]*:[[:space:]]*//' | tr -d '\r'
}

# 检查 HTTP 响应状态是否为 200 或 302
check_http_ok() {
    local code
    code=$(grep -i '^HTTP/[0-9.]* ' "$HEADER_FILE" | tail -1 | awk '{print $2}')
    [ "$code" = "200" ] || [ "$code" = "302" ]
}

# 检查命令是否存在，不存在则报错退出
require_cmd() {
    command -v "$1" >/dev/null 2>&1 || {
        echo "ERROR: $1 is required but not installed."
        exit 1
    }
}

# ==================== 前置检查 ====================
require_cmd curl
require_cmd jq

echo ""
echo "=============================================="
echo "     OIDC SSO 登录流程验证"
echo "     Issuer: $IAM_BASE"
echo "=============================================="
echo ""

# ============================================================
# Step 1: Discovery — 获取 OIDC Provider 元数据
# 验证内容: issuer、authorization_endpoint、token_endpoint 等
# ============================================================
echo "[1/8] 获取 Discovery 元数据..."

curl -s -D "$HEADER_FILE" -o "$TMPDIR/discovery.json" \
    "$IAM_BASE/.well-known/openid-configuration"

if ! check_http_ok; then
    fail "Discovery 请求失败"
fi

issuer=$(jq -r '.issuer' "$TMPDIR/discovery.json" 2>/dev/null || echo "")
authz_ep=$(jq -r '.authorization_endpoint' "$TMPDIR/discovery.json" 2>/dev/null || echo "")
token_ep=$(jq -r '.token_endpoint' "$TMPDIR/discovery.json" 2>/dev/null || echo "")
userinfo_ep=$(jq -r '.userinfo_endpoint' "$TMPDIR/discovery.json" 2>/dev/null || echo "")
jwks_uri=$(jq -r '.jwks_uri' "$TMPDIR/discovery.json" 2>/dev/null || echo "")

[ "$issuer" = "$IAM_BASE" ]        || fail "issuer 不匹配: $issuer"
[ -n "$authz_ep" ]                 || fail "缺少 authorization_endpoint"
[ -n "$token_ep" ]                 || fail "缺少 token_endpoint"
[ -n "$userinfo_ep" ]              || fail "缺少 userinfo_endpoint"
[ -n "$jwks_uri" ]                 || fail "缺少 jwks_uri"

pass "issuer=$issuer"

# ============================================================
# Step 2: JWKS — 获取公钥列表
# 验证内容: 密钥数量、算法类型（必须是 RSA + RS256）
# ============================================================
echo "[2/8] 获取 JWKS 公钥..."

curl -s -D "$HEADER_FILE" -o "$TMPDIR/jwks.json" \
    "$jwks_uri"

if ! check_http_ok; then
    fail "JWKS 请求失败"
fi

key_count=$(jq '.keys | length' "$TMPDIR/jwks.json" 2>/dev/null || echo "0")
kty=$(jq -r '.keys[0].kty' "$TMPDIR/jwks.json" 2>/dev/null || echo "")
alg=$(jq -r '.keys[0].alg' "$TMPDIR/jwks.json" 2>/dev/null || echo "")

[ "$key_count" -gt 0 ] || fail "JWKS 无密钥"
[ "$kty" = "RSA" ]      || fail "非 RSA 密钥: kty=$kty"
[ "$alg" = "RS256" ]    || fail "非 RS256 算法: alg=$alg"

pass "keys=$key_count, kty=$kty, alg=$alg"

# ============================================================
# Step 3: Authorize — 发起授权请求
# 流程: 模拟 RP 跳转 → 后端创建 AuthRequest → 返回 302
# 捕获: Location 中的 authRequestID
# ============================================================
echo "[3/8] 发起 Authorize 授权请求（提取 authRequestID）..."

authorize_url="$IAM_BASE/authorize?client_id=$CLIENT_ID&redirect_uri=$REDIRECT_URI&response_type=code&scope=openid%20profile%20email&state=$STATE&nonce=$NONCE"

# 不跟随重定向，捕获 302 的 Location 头
curl -s -D "$HEADER_FILE" -o /dev/null "$authorize_url"

location=$(get_header "location")
if [ -z "$location" ]; then
    fail "Authorize 未返回 Location 头"
fi

auth_request_id=$(echo "$location" | sed -n 's/.*authRequestID=\([^& ]*\).*/\1/p')
if [ -z "$auth_request_id" ]; then
    fail "无法从 Location 提取 authRequestID: $location"
fi

pass "authRequestID=$auth_request_id"

# ============================================================
# Step 4: Login — 提交用户凭证完成认证
# 请求: POST /login 携带 authRequestID + 用户名密码
# 响应: continueURL（用于后续回调）+ Set-Cookie（SSO Session）
# ============================================================
echo "[4/8] 提交登录凭证..."

login_resp=$(curl -s -X POST "$IAM_BASE/login" \
    -H 'Content-Type: application/json' \
    -c "$COOKIE_JAR" \
    -d "{
        \"authRequestID\": \"$auth_request_id\",
        \"identifier\": \"$IDENTIFIER\",
        \"password\": \"$PASSWORD\"
    }")

login_code=$(echo "$login_resp" | jq -r '.code' 2>/dev/null || echo "null")
continue_url=$(echo "$login_resp" | jq -r '.data.continueURL' 2>/dev/null || echo "")

if [ "$login_code" != "0" ]; then
    err_msg=$(echo "$login_resp" | jq -r '.msg' 2>/dev/null || echo "$login_resp")
    fail "登录失败 code=$login_code msg=$err_msg"
fi
if [ -z "$continue_url" ]; then
    fail "缺少 continueURL"
fi

pass "continueURL=$continue_url"

# ============================================================
# Step 5: Callback — 跟随 continueURL 完成授权码回调
# 流程: 后端生成授权码 → 302 重定向到 client 的 redirect_uri
# 捕获: Location 中的 code（授权码）和 state（CSRF 校验）
# ============================================================
echo "[5/8] 跟随回调获取授权码..."

curl -s -D "$HEADER_FILE" -o /dev/null \
    -b "$COOKIE_JAR" \
    "$continue_url"

callback_location=$(get_header "location")
if [ -z "$callback_location" ]; then
    fail "Callback 未返回 Location"
fi

code=$(echo "$callback_location" | sed -n 's/.*[?&]code=\([^& ]*\).*/\1/p')
returned_state=$(echo "$callback_location" | sed -n 's/.*[?&]state=\([^& ]*\).*/\1/p')

if [ -z "$code" ]; then
    fail "无法从 Location 提取 code: $callback_location"
fi
if [ "$returned_state" != "$STATE" ]; then
    fail "state 不匹配: expected=$STATE got=$returned_state"
fi

pass "code=${code:0:16}..., state=$returned_state"

# ============================================================
# Step 6: Token — 授权码换令牌
# 请求: POST /oauth/token 携带 code + client_id + client_secret
# 响应: access_token、id_token、refresh_token
# ============================================================
echo "[6/8] 授权码换 Token..."

token_resp=$(curl -s -X POST "$IAM_BASE/oauth/token" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    -d "grant_type=authorization_code" \
    -d "code=$code" \
    -d "redirect_uri=$REDIRECT_URI" \
    -d "client_id=$CLIENT_ID" \
    -d "client_secret=$CLIENT_SECRET")

access_token=$(echo "$token_resp" | jq -r '.access_token' 2>/dev/null || echo "")
token_type=$(echo "$token_resp" | jq -r '.token_type' 2>/dev/null || echo "")
expires_in=$(echo "$token_resp" | jq -r '.expires_in' 2>/dev/null || echo "0")
id_token=$(echo "$token_resp" | jq -r '.id_token' 2>/dev/null || echo "")
refresh_token=$(echo "$token_resp" | jq -r '.refresh_token' 2>/dev/null || echo "")

if [ -z "$access_token" ] || [ "$access_token" = "null" ]; then
    err=$(echo "$token_resp" | jq -r '.error_description // .error // empty' 2>/dev/null)
    fail "Token 交换失败: ${err:-$token_resp}"
fi
[ "$token_type" = "Bearer" ] || fail "token_type 不是 Bearer: $token_type"
[ "$expires_in" -gt 0 ]       || fail "expires_in 异常: $expires_in"
[ -n "$id_token" ]            || fail "缺少 id_token"
[ -n "$refresh_token" ]       || fail "缺少 refresh_token"

pass "access_token=${access_token:0:16}..., expires_in=${expires_in}s"

# ============================================================
# Step 7: UserInfo — 获取用户信息
# 携带 access_token 请求用户信息，验证 OIDC 兼容认证
# ============================================================
echo "[7/8] 获取用户信息..."

userinfo_resp=$(curl -s "$IAM_BASE/userinfo" \
    -H "Authorization: Bearer $access_token" \
    -w "\n%{http_code}" \
    -D "$HEADER_FILE")

userinfo_http_code=$(echo "$userinfo_resp" | tail -1)
userinfo_body=$(echo "$userinfo_resp" | sed '$d')

if [ "$userinfo_http_code" != "200" ]; then
    fail "UserInfo 请求失败 HTTP $userinfo_http_code: $userinfo_body"
fi

sub=$(echo "$userinfo_body" | jq -r '.sub' 2>/dev/null || echo "")
name=$(echo "$userinfo_body" | jq -r '.name' 2>/dev/null || echo "")
email=$(echo "$userinfo_body" | jq -r '.email' 2>/dev/null || echo "")

[ -n "$sub" ]   || fail "缺少 sub"
[ -n "$name" ]  || fail "缺少 name"
[ -n "$email" ] || fail "缺少 email"

pass "sub=$sub, name=$name, email=$email"

# ============================================================
# Step 8: Logout — 注销 SSO Session
# 使用 id_token_hint 发起登出，清除 SSO Session Cookie
# ============================================================
echo "[8/8] 注销登出..."

curl -s -D "$HEADER_FILE" -o /dev/null \
    -b "$COOKIE_JAR" \
    "$IAM_BASE/end_session?id_token_hint=$id_token"

end_session_code=$(grep -i '^HTTP/[0-9.]* ' "$HEADER_FILE" | tail -1 | awk '{print $2}')

if [ "$end_session_code" = "302" ] || [ "$end_session_code" = "200" ]; then
    pass "登出完成 HTTP $end_session_code"
else
    fail "登出失败 HTTP $end_session_code"
fi

# ============================================================
# 验证结果总结
# ============================================================
echo ""
echo "=============================================="
echo -e "  ${GREEN}全部 8 步验证通过${NC}"
echo "=============================================="
echo "  authRequestID : $auth_request_id"
echo "  code          : ${code:0:16}..."
echo "  access_token  : ${access_token:0:20}..."
echo "  sub           : $sub"
echo "=============================================="
echo ""
