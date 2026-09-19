package oidcop

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"

	"github.com/morehao/ark-iam/pkg/credential"
	"github.com/morehao/ark-iam/pkg/dao"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/model"
	"github.com/morehao/ark-iam/pkg/object/objauth"
	"github.com/morehao/ark-iam/pkg/sso"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

// errRefreshTokenReused 标记 refresh token 复用（已被并发轮换或撤销）。
// 按 RFC 9706 §4.1 检测到复用时应撤销整个 token 家族。
var errRefreshTokenReused = errors.New("refresh token reused")

// refreshRotationGraceWindow 是刷新令牌轮换的并发宽限窗口。
//
// 背景：轮换成功后旧 refresh token 立刻被条件撤销，此后任何携带旧 token 的请求都会被
// 判为「复用」，进而按 RFC 9706 §4.1 撤销整个 token 家族、把用户踢下线。但真实世界里
// 「同一秒内两个请求」绝大多数不是攻击，而是客户端/网络重试（响应丢失后重发）、
// 或用户同时开了两个标签页——此时家族撤销是明显的误伤，用户被迫重新登录。
//
// 语义：轮换成功后的 refreshRotationGraceWindow 内，携带**同一把**旧 token 的请求不再
// 判为复用，而是**原样返回刚签发的那把新 refresh token**（幂等重放）。
// 窗口外的复用、以及窗口内携带**另一把**已撤销 token 的请求，一律按复用处理
// （家族撤销，fail-closed）。窗口取 10s：覆盖一次 TCP 重传 + 服务端超时重试，
// 远小于任何人工重放的价值。
const refreshRotationGraceWindow = 10 * time.Second

// maxRefreshGraceEntries 是宽限缓存的条目上界（超过即先清理过期项，再淘汰最旧一条）。
// 条目只在窗口内存活，正常并发量下远达不到该上界；设上界是为了防止
// 「大量唯一旧 token 在窗口内并发」把进程内存打满。
const maxRefreshGraceEntries = 4096

// refreshGraceEntry 是一次轮换的可重放结果。
type refreshGraceEntry struct {
	newRefreshToken string
	expiration      time.Time
	rotatedAt       time.Time
}

// rotationResult 是宽限窗口内可重放的轮换产物。
// 只含 refresh token：access token 每次都必须新签发（jti 唯一）。
type rotationResult struct {
	newRefreshToken string
	expiration      time.Time
}

// rotationKey 把一把 refresh token 映射为宽限缓存键。
//
// 刻意**只**用 token 本身、不带 client_id：refresh token 是 256 位随机不透明串，
// 全局唯一且行内已绑定 person/tenant/client，不存在跨客户端撞键；
// 而 client_id 在「旧 token 已被撤销」这条路径上并不总能解析出来
// （撤销行的 ApplicationClientID 可能为空或指向已删行），带进键里会让同一把 token
// 在登记与查询两侧算出不同的键，宽限窗口直接失效。
//
// 用 sha256 派生而不是直接存明文 token：缓存只为单进程短期存在，
// 但仍不应在内存里留明文长期凭证。
func rotationKey(refreshToken string) string {
	sum := sha256.Sum256([]byte(refreshToken))
	return base64.RawURLEncoding.EncodeToString(sum[:16])
}

// withinRotationGrace 判断一次撤销是否仍处在并发宽限窗口内。
func withinRotationGrace(revokedAt, now time.Time) bool {
	return !revokedAt.IsZero() && now.Sub(revokedAt) <= refreshRotationGraceWindow
}

// lookupRotationGrace 命中宽限窗口时返回可重放的轮换结果。
func (s *PersistentStore) lookupRotationGrace(key string) (refreshGraceEntry, bool) {
	now := time.Now()
	s.rotationMu.Lock()
	defer s.rotationMu.Unlock()
	entry, ok := s.rotationGrace[key]
	if !ok {
		return refreshGraceEntry{}, false
	}
	if now.Sub(entry.rotatedAt) > refreshRotationGraceWindow {
		delete(s.rotationGrace, key)
		return refreshGraceEntry{}, false
	}
	return entry, true
}

// rememberRotationGrace 登记一次轮换结果，并顺带清理过期项与淘汰最旧条目。
func (s *PersistentStore) rememberRotationGrace(key string, entry refreshGraceEntry) {
	now := time.Now()
	s.rotationMu.Lock()
	defer s.rotationMu.Unlock()
	for k, v := range s.rotationGrace {
		if now.Sub(v.rotatedAt) > refreshRotationGraceWindow {
			delete(s.rotationGrace, k)
		}
	}
	for len(s.rotationGrace) >= maxRefreshGraceEntries {
		// 达到上界（窗口内出现远超预期的唯一旧 token 并发）：淘汰最旧的一条腾位置。
		// 整体清空会连带丢掉刚写入的窗口，让同一把 token 的重试退化成复用误判；
		// 淘汰最旧则最坏只影响最老的那次轮换。
		oldestKey, oldestAt := "", now
		for k, v := range s.rotationGrace {
			if oldestKey == "" || v.rotatedAt.Before(oldestAt) {
				oldestKey, oldestAt = k, v.rotatedAt
			}
		}
		if oldestKey == "" {
			break
		}
		delete(s.rotationGrace, oldestKey)
	}
	s.rotationGrace[key] = entry
}

// beginRotation 把当前请求登记为该旧 token 的首个轮换者。
// 返回 (leader=true, nil) 表示由本请求执行真正的轮换；
// 返回 (false, done) 表示已有在途轮换，done 会在其结束时关闭。
func (s *PersistentStore) beginRotation(key string) (bool, chan struct{}) {
	s.rotationMu.Lock()
	defer s.rotationMu.Unlock()
	if done, ok := s.rotationInflight[key]; ok {
		return false, done
	}
	done := make(chan struct{})
	s.rotationInflight[key] = done
	return true, done
}

// endRotation 结束在途登记并唤醒等待者。
func (s *PersistentStore) endRotation(key string, done chan struct{}) {
	s.rotationMu.Lock()
	delete(s.rotationInflight, key)
	s.rotationMu.Unlock()
	close(done)
}

// refreshWithGrace 在「轮换执行的起点」处理宽限窗口，返回 ok=true 表示本次请求已被
// 宽限路径满足（调用方直接返回该结果）。
//
// 两条命中路径：
//  1. 旧 token 已经轮换过且在窗口内 → 直接重放刚签发的结果（幂等）；
//  2. 另一个请求正在用同一把旧 token 轮换 → 等它结束再读缓存（避免并发双轮换）。
//
// 返回 ok=false 时调用方照常执行轮换；此时若旧 token 已被撤销、且不在窗口内，
// 既有的条件撤销判定会把它落到 errRefreshTokenReused（家族撤销）——严格语义不变。
func (s *PersistentStore) refreshWithGrace(currentRefreshToken string) (rotationResult, bool) {
	key := rotationKey(currentRefreshToken)

	if entry, ok := s.lookupRotationGrace(key); ok {
		// access token 的明文只在签发函数内组装，这里只能回放 refresh token；
		// 调用方会据此重新签发 access token（jti 唯一），语义等价且更安全。
		return rotationResult{newRefreshToken: entry.newRefreshToken, expiration: entry.expiration}, true
	}

	leader, done := s.beginRotation(key)
	if leader {
		// defer 保证无论成功/失败都唤醒等待者并清掉在途登记。
		defer s.endRotation(key, done)
		return rotationResult{}, false
	}

	// 只等「在途轮换结束」或「窗口到期」，不看请求 ctx：刷新是写操作，
	// 客户端断开不代表轮换没发生，此时若提前返回会让调用方以为"没轮换过"而重试。
	select {
	case <-done:
	case <-time.After(refreshRotationGraceWindow):
		// 在途轮换迟迟不结束：不阻塞请求，退回常规路径自行处理（旧 token 已被撤销时
		// 条件撤销会命中 0 行 → 仍按复用处理，fail-closed）。
		return rotationResult{}, false
	}

	if entry, ok := s.lookupRotationGrace(key); ok {
		return rotationResult{newRefreshToken: entry.newRefreshToken, expiration: entry.expiration}, true
	}
	return rotationResult{}, false
}

// persistentStoreOption 承载持久化存储的可注入配置。
type persistentStoreOption struct {
	// issuer 为 OP 的 issuer，构造 OIDCClient 时注入，用于生成 LoginURL。
	issuer string
}

// PersistentStoreOption 允许调用方为持久化存储注入配置。
type PersistentStoreOption func(*persistentStoreOption)

// WithIssuer 注入 OP issuer，使签发的 OIDCClient 能构造正确的 LoginURL。
func WithIssuer(issuer string) PersistentStoreOption {
	return func(o *persistentStoreOption) { o.issuer = issuer }
}

type PersistentStore struct {
	applicationClientDao       func(opts ...dao.DaoOption) *dao.ApplicationClientDao
	applicationClientSecretDao func(opts ...dao.DaoOption) *dao.ApplicationClientSecretDao
	personDao                  func(opts ...dao.DaoOption) *dao.PersonDao
	tenantDao                  func(opts ...dao.DaoOption) *dao.TenantDao
	userDao                    func(opts ...dao.DaoOption) *dao.UserDao
	refreshTokenDao            func(opts ...dao.DaoOption) *dao.RefreshTokenDao
	apiKeyDao                  func(opts ...dao.DaoOption) *dao.ApiKeyDao
	userRoleDao                func(opts ...dao.DaoOption) *dao.UserRoleDao
	roleDao                    func(opts ...dao.DaoOption) *dao.RoleDao
	issuer                     string
	// db 返回用于事务的 DB 句柄（轮换原子性等）。默认全局 iam 库，
	// 测试可注入独立 SQLite 连接。
	db func(ctx context.Context) *gorm.DB

	// rotationMu 保护下面的宽限缓存与在途登记表（见 refreshRotationGraceWindow）。
	rotationMu       sync.Mutex
	rotationGrace    map[string]refreshGraceEntry
	rotationInflight map[string]chan struct{}
}

func NewPersistentStore(opts ...PersistentStoreOption) *PersistentStore {
	cfg := &persistentStoreOption{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &PersistentStore{
		applicationClientDao:       func(opts ...dao.DaoOption) *dao.ApplicationClientDao { return dao.NewApplicationClientDao() },
		applicationClientSecretDao: dao.NewApplicationClientSecretDao,
		personDao:                  dao.NewPersonDao,
		tenantDao:                  dao.NewTenantDao,
		userDao:                    dao.NewUserDao,
		refreshTokenDao:            dao.NewRefreshTokenDao,
		apiKeyDao:                  func(opts ...dao.DaoOption) *dao.ApiKeyDao { return dao.NewApiKeyDao() },
		userRoleDao:                dao.NewUserRoleDao,
		roleDao:                    dao.NewRoleDao,
		issuer:                     cfg.issuer,
		db:                         dbclient.IamDB,
		rotationGrace:              make(map[string]refreshGraceEntry),
		rotationInflight:           make(map[string]chan struct{}),
	}
}

// 协议层（op.Storage 适配）的租户作用域策略
//
// 协议端点（/oidc/*）没有「当前租户」：调用 ctx 要么来自 zitadel 透传的请求上下文，
// 要么来自后台 worker。因此本文件每一次数据访问都必须自证可见范围：
//
//   - 按全局唯一键（client_id、API Key 摘要、refresh token 摘要、行 ID）或自然人级范围
//     读取 → dbclient.CrossTenantContext(ctx)（跨全部租户）；
//   - 已解析出目标租户的行级读写 → dbclient.ExplicitTenantContext(ctx, tenantID)。
//
// 禁止用「ctx 恰好没有作用域」表达跨租户可见：那样上游一旦带上租户，查询会变成静默误过滤
// （少读不报错、UPDATE 命中 0 行不报错），而 fail-closed 会把真正的缺声明当场暴露。

// txDB 返回事务用 DB；未注入时回退全局 iam 库。
func (s *PersistentStore) txDB(ctx context.Context) *gorm.DB {
	if s.db != nil {
		if db := s.db(ctx); db != nil {
			return db
		}
	}
	return dbclient.IamDB(ctx)
}

func (s *PersistentStore) LookupApiKeyByRawKey(ctx context.Context, rawKey string) (*model.ApiKeyEntity, error) {
	// API Key 摘要全局唯一，且本查询发生在「租户确定之前」：显式声明跨全部租户。
	ctx = dbclient.CrossTenantContext(ctx)
	hash := credential.HashSecret(rawKey)
	entity, err := s.apiKeyDao().GetByCond(ctx, &dao.ApiKeyCond{KeyHash: hash})
	if err != nil || entity == nil || entity.ID == "" {
		return nil, nil
	}
	if entity.RevokedAt != nil && !entity.RevokedAt.IsZero() {
		return nil, nil
	}
	if entity.ExpiredAt != nil && entity.ExpiredAt.Before(time.Now()) {
		return nil, nil
	}
	return entity, nil
}

func (s *PersistentStore) GetApiKeyClientByRawKey(ctx context.Context, rawKey string) (op.Client, error) {
	entity, err := s.LookupApiKeyByRawKey(ctx, rawKey)
	if err != nil || entity == nil {
		return nil, oidc.ErrInvalidClient()
	}
	return NewApiKeyOpClient(entity), nil
}

func (s *PersistentStore) GetClientByClientID(ctx context.Context, clientID string) (op.Client, error) {
	// client_id 全局唯一（租户确定之前即需校验）：显式声明跨全部租户。
	ctx = dbclient.CrossTenantContext(ctx)
	// H4：仅返回启用状态的 client，管理员停用后 authorize/token/client_credentials 立即失效
	clientEntity, err := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID, Status: model.ApplicationClientStatusEnable})
	if err != nil || clientEntity == nil || clientEntity.ID == "" {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}
	return NewOIDCClient(clientEntity, s.issuer), nil
}

func (s *PersistentStore) AuthorizeClientIDSecret(ctx context.Context, clientID, clientSecret string) error {
	// client_id + 密钥摘要校验同样发生在租户确定之前：显式声明跨全部租户。
	ctx = dbclient.CrossTenantContext(ctx)
	clientHash := credential.HashSecret(clientSecret)

	// H4：停用 client 的 secret 一律拒绝
	clientEntity, err := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID, Status: model.ApplicationClientStatusEnable})
	if err != nil || clientEntity == nil || clientEntity.ID == "" {
		return oidc.ErrInvalidClient()
	}
	secrets, err := s.applicationClientSecretDao().GetListByCond(ctx, &dao.ApplicationClientSecretCond{ApplicationClientID: clientEntity.ID})
	if err != nil {
		return oidc.ErrInvalidClient()
	}
	for _, sec := range secrets {
		// H14：哈希比较使用恒定时间算法，避免时序侧信道
		if subtle.ConstantTimeCompare([]byte(sec.ValueHash), []byte(clientHash)) == 1 && sec.RevokedAt == nil {
			if sec.ExpiredAt == nil || sec.ExpiredAt.After(time.Now()) {
				return nil
			}
		}
	}
	return oidc.ErrInvalidClient()
}

// fillUserInfoByScopes 按 scope 填充 userinfo 标准声明。
// email_verified 恒为 false：当前无邮箱验证流程，不得宣称已验证（H5）。
func fillUserInfoByScopes(userinfo *oidc.UserInfo, person *model.PersonEntity, scopes []string) {
	for _, scope := range scopes {
		switch scope {
		case oidc.ScopeProfile:
			userinfo.Name = person.Name
			userinfo.PreferredUsername = model.DerefStr(person.Username)
		case oidc.ScopeEmail:
			userinfo.Email = model.DerefStr(person.PrimaryEmail)
			if userinfo.Email != "" {
				userinfo.EmailVerified = false
			}
		case oidc.ScopePhone:
			userinfo.PhoneNumber = model.DerefStr(person.PrimaryPhone)
			userinfo.PhoneNumberVerified = false
		}
	}
}

// ClaimGroups 是 ID token / userinfo 的角色组声明名（非标准 OIDC 声明，跨系统授权契约）：
// 值为该租户内、且属于本次请求客户端所属应用的角色编码（model.RoleCode）。下游系统按编码认
// 自己的权限策略（如对象存储的 claim_prefix + 策略名），故角色编码即契约——它由应用角色模板
// （application.role_template）统一定义后物化到各租户，租户自建角色的编码为空串、不参与本声明。
//
// 声明的作用域是「应用」而非「租户」：只承载请求方所属应用的编码。若按租户把所有应用的编码
// 一起发出去，任何应用里的角色都会流进每个下游的策略命名空间——在无关应用里造一个同码角色
// 即可命中下游策略（跨应用越权），而下游侧无从分辨。
const ClaimGroups = "groups"

// hasScope 判断 scope 列表是否包含指定 scope。
func hasScope(scopes []string, want string) bool {
	for _, scope := range scopes {
		if scope == want {
			return true
		}
	}
	return false
}

// appendRoleGroupClaims 在 profile scope 下把「该租户内、且属于本次请求客户端所属应用的
// 角色编码」写入 groups 声明。
//
// 租户必须由调用方显式给出（授权码流取授权票据的租户、刷新流取 refresh token 的租户、
// userinfo 端点取 access token 元数据的租户）：角色与角色绑定都是租户维度实体，多租户自然人的
// 角色集合只能按本次请求的租户裁剪——不跨租户兜底、也不猜租户。
//
// 应用作用域（clientID → application_client.app_id）是声明的命名空间边界：groups 是给下游系统
// 认策略名用的，而下游只可能认自己那个应用的角色。缺这层裁剪，任何应用的角色编码都会进入每个
// 下游的策略命名空间（在无关应用里造同码角色即可命中下游策略），且下游无从分辨来源。
//
// 两种 fail-closed：客户端/应用解析失败、以及角色读取失败，都返回 error 让 op 层拒绝本次签发：
// 宁可登录失败，也不签发作用域不明或缺 groups 的 token——那会让下游把用户当作「无任何策略」
// 而静默降权。
func (s *PersistentStore) appendRoleGroupClaims(ctx context.Context, userinfo *oidc.UserInfo, tenantID, clientID, subject string, scopes []string) error {
	if tenantID == "" || !hasScope(scopes, oidc.ScopeProfile) {
		return nil
	}
	pid, err := ParseSubject(subject)
	if err != nil {
		return nil
	}
	// 角色/角色绑定都带 tenant_id：协议层没有「当前租户」，显式声明指定租户作用域。
	tctx := dbclient.ExplicitTenantContext(ctx, tenantID)
	// 先定应用作用域再读角色：应用未定（客户端为空或未绑定应用）就不产出声明，不猜命名空间。
	appID, err := s.resolveClientAppID(ctx, clientID)
	if err != nil {
		glog.Errorf(ctx, "[PersistentStore.appendRoleGroupClaims] resolve client app fail, tenantID:%s, clientID:%s, err:%v", tenantID, clientID, err)
		return err
	}
	if appID == "" {
		return nil
	}
	// user_role.user_id 是**租户成员**（tenant_user）主键，不是自然人 ID：协议层只拿得到
	// subject(person:<id>)，必须先按 (person_id, tenant_id) 解析出成员行。少这一步会恒查不到角色，
	// 并「静默产出空 groups」——下游按 groups 认策略名且是 fail-closed，会直接拒绝登录（端到端已实测）。
	members, err := s.userDao().GetListByCond(tctx, &dao.UserCond{PersonID: pid, TenantID: tenantID})
	if err != nil {
		glog.Errorf(ctx, "[PersistentStore.appendRoleGroupClaims] query tenant_user fail, tenantID:%s, err:%v", tenantID, err)
		return err
	}
	if len(members) == 0 {
		return nil
	}
	roleIDs := make([]string, 0, len(members))
	seenRoleID := make(map[string]struct{}, len(members))
	for _, member := range members {
		userRoles, uerr := s.userRoleDao().GetListByCond(tctx, &dao.UserRoleCond{TenantID: tenantID, UserID: member.ID})
		if uerr != nil {
			glog.Errorf(ctx, "[PersistentStore.appendRoleGroupClaims] query user_role fail, tenantID:%s, userID:%s, err:%v", tenantID, member.ID, uerr)
			return uerr
		}
		for _, userRole := range userRoles {
			if _, ok := seenRoleID[userRole.RoleID]; ok {
				continue
			}
			seenRoleID[userRole.RoleID] = struct{}{}
			roleIDs = append(roleIDs, userRole.RoleID)
		}
	}
	if len(roleIDs) == 0 {
		return nil
	}
	// AppID 过滤即「作用域裁剪」本体：只保留请求方所属应用的角色。未归属应用（app_id 为空串）
	// 的系统角色天然被排除——它们没有下游命名空间，发出去只会污染下游的策略匹配。
	roles, err := s.roleDao().GetListByCond(tctx, &dao.RoleCond{TenantID: tenantID, IDs: roleIDs, AppID: appID})
	if err != nil {
		glog.Errorf(ctx, "[PersistentStore.appendRoleGroupClaims] query role fail, tenantID:%s, appID:%s, err:%v", tenantID, appID, err)
		return err
	}
	codes := make([]string, 0, len(roles))
	seen := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		code := string(role.Code)
		// 空编码不产出声明：空串在下游会被当成一个"策略名"去匹配，属静默错配。
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	if len(codes) == 0 {
		return nil
	}
	// 稳定排序：同一用户的 groups 每次签发顺序一致，便于下游比对与排障。
	sort.Strings(codes)
	userinfo.AppendClaims(ClaimGroups, codes)
	return nil
}

// resolveClientAppID 解析本次签发客户端所属的应用——即 groups 声明的命名空间。
//
// client_id 全局唯一（`application_client.code` 带全局唯一索引），且解析发生在「租户确定之前」，
// 与 GetClientByClientID 同口径：显式声明跨全部租户读取。
//
// 返回空串表示「没有可用的命名空间」，此时不产出声明而不报错——与「租户未定不猜租户」同构：
// clientID 为空（调用方拿不到客户端）或客户端未绑定应用，都属合法的「无作用域」。
// 客户端记录不存在则是状态异常，返回 error 由调用方 fail-closed 拒绝签发。
func (s *PersistentStore) resolveClientAppID(ctx context.Context, clientID string) (string, error) {
	if clientID == "" {
		return "", nil
	}
	cctx := dbclient.CrossTenantContext(ctx)
	client, err := s.applicationClientDao().GetByCond(cctx, &dao.ApplicationClientCond{Code: clientID})
	if err != nil {
		return "", err
	}
	if client == nil || client.ID == "" {
		return "", fmt.Errorf("application client not found: %s", clientID)
	}
	return client.AppID, nil
}

func (s *PersistentStore) SetUserinfoFromScopes(ctx context.Context, userinfo *oidc.UserInfo, userID, clientID string, scopes []string) error {
	userinfo.Subject = userID
	pid, err := ParseSubject(userID)
	if err != nil {
		return nil
	}
	person, err := s.personDao().GetByID(ctx, pid)
	if err != nil || person == nil || person.ID == "" {
		return nil
	}
	fillUserInfoByScopes(userinfo, person, scopes)
	return nil
}

// SetUserinfoFromToken 按 access token 签发时记录的 scope 裁剪 userinfo 声明（M2）。
// 已被撤销（黑名单）或元数据不可得（Redis 不可用 / 未知 token）时返回 error，
// 由 op 层拒绝请求（403），避免绕过 scope 授权泄露 email/name 等声明。
func (s *PersistentStore) SetUserinfoFromToken(ctx context.Context, userinfo *oidc.UserInfo, tokenID, subject, origin string) error {
	userinfo.Subject = subject
	if isAccessTokenRevoked(ctx, tokenID) {
		return errors.New("access token revoked")
	}
	meta := loadAccessTokenMeta(ctx, tokenID)
	if meta == nil {
		return errors.New("access token meta not found")
	}
	pid, err := ParseSubject(subject)
	if err != nil {
		return nil
	}
	person, err := s.personDao().GetByID(ctx, pid)
	if err != nil || person == nil || person.ID == "" {
		return nil
	}
	fillUserInfoByScopes(userinfo, person, meta.Scopes)
	// userinfo 端点回查场景：groups 同样按该 access token 所属租户 + 客户端所属应用裁剪
	// （与 ID token 口径一致）；客户端取元数据里签发时记下的 client_id。
	if err := s.appendRoleGroupClaims(ctx, userinfo, meta.TenantID, meta.ClientID, subject, meta.Scopes); err != nil {
		return err
	}
	return nil
}

// SetIntrospectionFromToken 返回 RFC 7662 规定的完整 introspection 响应（M1）：
// scope/client_id/sub/exp/iat/token_type/username 及私有声明。
// 元数据不可得或 token 已被撤销时返回 error，op 层将保持 active=false。
func (s *PersistentStore) SetIntrospectionFromToken(ctx context.Context, introspection *oidc.IntrospectionResponse, tokenID, subject, clientID string) error {
	if isAccessTokenRevoked(ctx, tokenID) {
		return errors.New("access token revoked")
	}
	meta := loadAccessTokenMeta(ctx, tokenID)
	if meta == nil {
		return errors.New("access token meta not found")
	}
	introspection.Scope = oidc.SpaceDelimitedArray(meta.Scopes)
	introspection.ClientID = meta.ClientID
	introspection.TokenType = oidc.BearerToken
	introspection.Expiration = oidc.FromTime(meta.ExpiresAt)
	introspection.IssuedAt = oidc.FromTime(meta.IssuedAt)
	introspection.Subject = meta.Subject
	introspection.Audience = oidc.Audience{meta.ClientID}
	introspection.Username = meta.Username
	// 人 token 补充 username（person 的用户名）；机器 token 直接用 client 标识。
	if introspection.Username == "" {
		if pid, perr := ParseSubject(meta.Subject); perr == nil {
			if person, perr2 := s.personDao().GetByID(ctx, pid); perr2 == nil && person != nil && person.ID != "" {
				introspection.Username = model.DerefStr(person.Username)
			}
		} else {
			introspection.Username = meta.ClientID
		}
	}
	claims := make(map[string]any, 3)
	if meta.TenantID != "" {
		claims["tenant_id"] = meta.TenantID
	}
	if meta.TokenUsage != "" {
		claims["token_usage"] = meta.TokenUsage
	}
	if meta.SessionID != "" {
		claims["sid"] = meta.SessionID
	}
	if len(claims) > 0 {
		introspection.Claims = claims
	}
	return nil
}

func (s *PersistentStore) GetPrivateClaimsFromScopes(ctx context.Context, userID, clientID string, scopes []string) (map[string]any, error) {
	// 需要看到该自然人在**全部租户**的成员关系（多租户时宁可不产出 claim）：显式声明跨全部租户。
	ctx = dbclient.CrossTenantContext(ctx)
	pid, err := ParseSubject(userID)
	if err != nil {
		return nil, nil
	}
	users, err := s.userDao().GetListByCond(ctx, &dao.UserCond{PersonID: pid})
	if err != nil || len(users) == 0 {
		return nil, nil
	}
	// L4：请求未携带明确租户上下文时，只有单租户才可确定 tenant_id；
	// 多租户（users>1）存在歧义，宁可不产出 claim 也不静默取 users[0]。
	if len(users) != 1 {
		return nil, nil
	}
	return objauth.TokenClaims{TenantID: users[0].TenantID}.OIDCPrivateClaims(), nil
}

func (s *PersistentStore) GetKeyByIDAndClientID(ctx context.Context, keyID, clientID string) (*jose.JSONWebKey, error) {
	return nil, errors.New("key not found")
}

func (s *PersistentStore) ValidateJWTProfileScopes(ctx context.Context, userID string, scopes []string) ([]string, error) {
	return scopes, nil
}

func (s *PersistentStore) CreateAccessToken(ctx context.Context, request op.TokenRequest) (accessTokenID string, expiration time.Time, err error) {
	// 本方法只按 client_id 取配置，不做租户内行级读写：显式声明跨全部租户。
	ctx = dbclient.CrossTenantContext(ctx)
	accessTokenID, err = randomTokenID("at")
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate access token id: %w", err)
	}

	// sessionID：code 流且 client 未启用 refresh_token grant 时也走本方法，
	// 此时从 AuthRequest 提取会话标识，保证这类 client 也能收到 back-channel 登出通知。
	sessionID := ""
	if authReq, ok := request.(*AuthRequest); ok {
		sessionID = authReq.SessionID
	}

	ttl := 15 * time.Minute
	var backChannelLogoutURI string
	var clientID string
	if ccReq, ok := request.(*clientCredentialsTokenRequest); ok {
		clientID = ccReq.ClientID()
		if entity, e := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID}); e == nil && entity != nil && entity.AccessTokenTTL > 0 {
			ttl = time.Duration(entity.AccessTokenTTL) * time.Second
		} else if e != nil {
			glog.Warnf(ctx, "[PersistentStore.CreateAccessToken] load client ttl fail, clientID:%s, err:%v", clientID, e)
		}
	} else if authReq, ok := request.(*AuthRequest); ok {
		clientID = authReq.GetClientID()
		if clientID != "" {
			if entity, e := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID}); e == nil && entity != nil {
				backChannelLogoutURI = entity.BackChannelLogoutURI
				if entity.AccessTokenTTL > 0 {
					ttl = time.Duration(entity.AccessTokenTTL) * time.Second
				}
			}
		}
	}
	expiration = time.Now().Add(ttl)
	storeAccessTokenMeta(ctx, accessTokenID, accessTokenMeta{
		Subject:    request.GetSubject(),
		ClientID:   getClientIDFromRequest(request),
		Scopes:     request.GetScopes(),
		IssuedAt:   time.Now(),
		ExpiresAt:  expiration,
		TenantID:   tenantIDFromRequest(request),
		SessionID:  sessionID,
		TokenUsage: tokenUsageFromRequest(request),
	})
	// back-channel 登记：与 CreateAccessAndRefreshTokens 对齐，覆盖"仅 access token"的 code 流
	if sessionID != "" && backChannelLogoutURI != "" {
		pid, perr := ParseSubject(request.GetSubject())
		if perr == nil {
			_ = sso.NewSLOStore().Register(ctx, sessionID, sso.LogoutRegistration{
				OIDCSessionID:        accessTokenID,
				ClientID:             clientID,
				UserID:               BuildSubject(pid),
				SessionID:            sessionID,
				BackChannelLogoutURI: backChannelLogoutURI,
			})
		}
	}
	return accessTokenID, expiration, nil
}

func (s *PersistentStore) CreateAccessAndRefreshTokens(ctx context.Context, request op.TokenRequest, currentRefreshToken string) (accessTokenID string, newRefreshToken string, expiration time.Time, err error) {
	// 先按自然人列出其全部租户成员关系（再在内存中按 selectedTenantID 选定）：显式声明跨全部租户。
	ctx = dbclient.CrossTenantContext(ctx)

	// consumedRefreshToken 是**本次刷新要消费掉的那把旧 token**（授权码流为空）。
	// 单独留一份：函数后段还会用 currentRefreshToken 做条件撤销，
	// 而宽限窗口的登记与查询都必须以被消费的那把为键。
	consumedRefreshToken := currentRefreshToken

	// P9：刷新轮换的并发宽限窗口。刻意不做类型断言——refresh 流的 request 是
	// *refreshTokenRequest，不应假设它实现 op.AuthRequest；宽限窗口只依赖 token 本身。
	if consumedRefreshToken != "" {
		if result, ok := s.refreshWithGrace(consumedRefreshToken); ok {
			// access token 每次都新签发（jti 必须唯一，绝不复用），
			// 只有 refresh token 是幂等重放——这正是宽限窗口要解决的重复轮换问题。
			replayAccessTokenID, idErr := randomTokenID("at")
			if idErr != nil {
				return "", "", time.Time{}, fmt.Errorf("generate access token id: %w", idErr)
			}
			return replayAccessTokenID, result.newRefreshToken, result.expiration, nil
		}
	}
	accessTokenID, err = randomTokenID("at")
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate access token id: %w", err)
	}

	var personID string
	personID, err = ParseSubject(request.GetSubject())
	if err != nil {
		return "", "", time.Time{}, err
	}

	users, err := s.userDao().GetListByCond(ctx, &dao.UserCond{PersonID: personID})
	if err != nil || len(users) == 0 {
		return "", "", time.Time{}, fmt.Errorf("user not found for person %s", personID)
	}

	selectedTenantID := selectedTenantFromRequest(request)
	var userEntity *model.UserEntity
	if selectedTenantID != "" {
		for i := range users {
			if users[i].TenantID == selectedTenantID {
				userEntity = &users[i]
				break
			}
		}
	}
	if userEntity == nil {
		userEntity = &users[0]
	}
	// 选中的租户在此刻才确定：后续 refresh token 的插入/条件撤销都属于该租户，
	// 显式声明「指定租户」作用域（事务必须在声明之后创建，插件按事务自身的 ctx 注入）。
	tenantCtx := dbclient.ExplicitTenantContext(ctx, userEntity.TenantID)

	clientID := ""
	if authReq, ok := request.(op.AuthRequest); ok {
		clientID = authReq.GetClientID()
	}

	var applicationClientID string
	var clientAccessTokenTTL time.Duration
	var clientRefreshTokenTTL time.Duration
	backChannelLogoutURI := ""
	if clientID != "" {
		clientEntity, err := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID})
		if err == nil && clientEntity != nil {
			applicationClientID = clientEntity.ID
			backChannelLogoutURI = clientEntity.BackChannelLogoutURI
			if clientEntity.AccessTokenTTL > 0 {
				clientAccessTokenTTL = time.Duration(clientEntity.AccessTokenTTL) * time.Second
			}
			if clientEntity.RefreshTokenTTL > 0 {
				clientRefreshTokenTTL = time.Duration(clientEntity.RefreshTokenTTL) * time.Second
			}
		}
	}
	if clientAccessTokenTTL <= 0 {
		clientAccessTokenTTL = 15 * time.Minute
	}
	expiration = time.Now().Add(clientAccessTokenTTL)

	// H2：refresh token 持久化授权时授予的 scope / amr / auth_time，
	// 刷新时原样还原（RFC 6749 §6 要求刷新 token 的 scope 与原始授权一致）。
	scopes := append([]string(nil), request.GetScopes()...)
	// op.TokenRequest 不含 amr/auth_time（仅 IDTokenRequest 有），
	// 授权码流/刷新流的具体请求类型均实现之，此处用接口断言提取。
	var amr []string
	if r, ok := request.(interface{ GetAMR() []string }); ok {
		amr = append([]string(nil), r.GetAMR()...)
	}
	authTime := time.Time{}
	if r, ok := request.(interface{ GetAuthTime() time.Time }); ok {
		authTime = r.GetAuthTime()
	}
	if authTime.IsZero() {
		authTime = time.Now()
	}

	sessionID := ""
	if authReq, ok := request.(*AuthRequest); ok {
		sessionID = authReq.SessionID
	}
	if rr, ok := request.(*refreshTokenRequest); ok {
		// 刷新轮换必须还原授权时持久化的会话标识（M4），否则刷新后的 token 丢失 sid，
		// 无法再按会话粒度关联背信道登出与 id_token 的 sid 声明。
		sessionID = rr.GetSessionID()
	}

	now := time.Now()
	refreshTokenExp := now.Add(30 * 24 * time.Hour)
	if clientRefreshTokenTTL > 0 {
		refreshTokenExp = now.Add(clientRefreshTokenTTL)
	}
	// refresh token 是长期凭证，值必须为密码学随机（不可由时间戳+ID 推断）。
	refreshTokenValue, err := randomTokenID("rt")
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshTokenHash := credential.HashSecret(refreshTokenValue)
	refreshEntity := &model.RefreshTokenEntity{
		PersonID:            personID,
		TenantID:            userEntity.TenantID,
		UserID:              userEntity.ID,
		ApplicationClientID: applicationClientID,
		SessionID:           sessionID,
		Token:               refreshTokenHash,
		Scopes:              model.ScopeList(scopes),
		AMR:                 model.AuthMethodList(amr),
		AuthTime:            &authTime,
		ExpiredAt:           &refreshTokenExp,
		CreatedBy:           userEntity.ID,
	}
	if currentRefreshToken != "" {
		refreshEntity.LastRotatedAt = &now
	}

	// S7：新行插入与旧行撤销放同一事务，且旧行撤销采用条件 UPDATE
	// （WHERE token=? AND revoked_at IS NULL，行数=1 才算成功），
	// 杜绝并发刷新时两个请求都通过校验、各自产出一套新 token 的分裂。
	// 条件撤销命中 0 行说明旧 token 已被并发轮换/撤销 → 视为复用攻击
	// （RFC 9706 §4.1），撤销该 person 全部 refresh token（token 家族）。
	txErr := s.txDB(tenantCtx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(refreshEntity).Error; err != nil {
			return err
		}
		if currentRefreshToken != "" {
			oldTokenHash := credential.HashSecret(currentRefreshToken)
			res := tx.Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
				Where("token = ? AND revoked_at IS NULL", oldTokenHash).
				Update("revoked_at", &now)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errRefreshTokenReused
			}
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, errRefreshTokenReused) {
			// 复用：旧 token 家族全部作废（独立于回滚的事务执行）
			glog.Warnf(ctx, "[PersistentStore.CreateAccessAndRefreshTokens] refresh token reused, revoke family, personID:%s", personID)
			famErr := s.txDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
				Where("person_id = ? AND revoked_at IS NULL", personID).
				Update("revoked_at", &now).Error
			if famErr != nil {
				glog.Warnf(ctx, "[PersistentStore.CreateAccessAndRefreshTokens] revoke token family fail, personID:%s, err:%v", personID, famErr)
			}
			return "", "", time.Time{}, op.ErrInvalidRefreshToken
		}
		return "", "", time.Time{}, txErr
	}

	// P9：轮换成功后登记宽限窗口，键是**刚被消费掉的那把旧 token**，
	// 使窗口内携带同一把旧 token 的并发/重试请求幂等拿到这把新 refresh token，
	// 而不是被判为复用并撤销整个 token 家族。
	if consumedRefreshToken != "" {
		s.rememberRotationGrace(rotationKey(consumedRefreshToken), refreshGraceEntry{
			newRefreshToken: refreshTokenValue,
			expiration:      expiration,
			rotatedAt:       now,
		})
	}

	// 登出登记：有 SSO 会话 且 client 配置了 back_channel_logout_uri 时，登记该会话对该 client 的通知关系。
	// 无 sid（服务账号/Client Credentials）或未配置背信道 URI 则跳过，对齐 OIDC Back-Channel 注册要求。
	if sessionID != "" && backChannelLogoutURI != "" {
		_ = sso.NewSLOStore().Register(ctx, sessionID, sso.LogoutRegistration{
			OIDCSessionID:        accessTokenID,
			ClientID:             clientID,
			UserID:               BuildSubject(personID),
			SessionID:            sessionID,
			BackChannelLogoutURI: backChannelLogoutURI,
		})
	}

	// access token 元数据（M1/M2：introspection 与 userinfo 按 scope 裁剪）。
	storeAccessTokenMeta(ctx, accessTokenID, accessTokenMeta{
		Subject:    request.GetSubject(),
		ClientID:   clientID,
		Scopes:     scopes,
		IssuedAt:   time.Now(),
		ExpiresAt:  expiration,
		TenantID:   userEntity.TenantID,
		SessionID:  sessionID,
		TokenUsage: "",
	})

	return accessTokenID, refreshTokenValue, expiration, nil
}

// tenantCarrier 由授权票据（*AuthRequest）与刷新令牌（*refreshTokenRequest）实现：
// 人 token 的租户是授权链路里显式解析出来的上下文，必须落到 access token 元数据上，
// userinfo（groups）与 introspection（tenant_id）才能在不跨租户兜底的前提下还原该 token 的租户。
type tenantCarrier interface {
	GetTenantID() string
}

// tenantIDFromRequest / tokenUsageFromRequest 供 access token 元数据使用。
func tenantIDFromRequest(request op.TokenRequest) string {
	if ccReq, ok := request.(*clientCredentialsTokenRequest); ok {
		return ccReq.ownerTenantID
	}
	if carrier, ok := request.(tenantCarrier); ok {
		return carrier.GetTenantID()
	}
	return ""
}

func tokenUsageFromRequest(request op.TokenRequest) objauth.TokenUsage {
	if ccReq, ok := request.(*clientCredentialsTokenRequest); ok && ccReq.isApiKey {
		return objauth.TokenUsageMachine
	}
	return ""
}

func (s *PersistentStore) TokenRequestByRefreshToken(ctx context.Context, refreshToken string) (op.RefreshTokenRequest, error) {
	// refresh token 摘要全局唯一（刷新时租户由该行还原）：显式声明跨全部租户。
	ctx = dbclient.CrossTenantContext(ctx)
	refreshTokenHash := credential.HashSecret(refreshToken)
	storedToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
	if err != nil || storedToken == nil || storedToken.ID == "" {
		return nil, op.ErrInvalidRefreshToken
	}
	// P9：刚被轮换撤销的 token 在宽限窗口内仍然"可读"——这里的放行不是授权，
	// 只是把最终判定交给 CreateAccessAndRefreshTokens（那里才有幂等重放/复用检测）。
	// 窗口外的撤销、以及已过期的 token 一律当场拒绝。
	if storedToken.RevokedAt != nil && !withinRotationGrace(*storedToken.RevokedAt, time.Now()) {
		return nil, op.ErrInvalidRefreshToken
	}
	if storedToken.ExpiredAt == nil || !storedToken.ExpiredAt.After(time.Now()) {
		return nil, op.ErrInvalidRefreshToken
	}

	clientID := ""
	if storedToken.ApplicationClientID != "" {
		clientEntity, err := s.applicationClientDao().GetByID(ctx, storedToken.ApplicationClientID)
		if err == nil && clientEntity != nil {
			clientID = clientEntity.Code
		}
	}

	// H2：还原授权时持久化的 scope / amr / auth_time / session_id，
	// 保证刷新后的 token 与原始授权一致（RFC 6749 §6），并让刷新后的 token 携带 sid。
	scopes := storedToken.Scopes.Strings()
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, oidc.ScopeProfile}
	}
	amr := storedToken.AMR.Strings()
	if len(amr) == 0 {
		amr = []string{"pwd"}
	}
	authTime := storedToken.CreatedAt
	if storedToken.AuthTime != nil && !storedToken.AuthTime.IsZero() {
		authTime = *storedToken.AuthTime
	}

	return &refreshTokenRequest{
		subject:   BuildSubject(storedToken.PersonID),
		audience:  []string{clientID},
		scopes:    scopes,
		clientID:  clientID,
		amr:       amr,
		authTime:  authTime,
		tenantID:  storedToken.TenantID,
		sessionID: storedToken.SessionID,
	}, nil
}

type refreshTokenRequest struct {
	subject   string
	audience  []string
	scopes    []string
	clientID  string
	amr       []string
	authTime  time.Time
	tenantID  string
	sessionID string
}

func (r *refreshTokenRequest) GetAMR() []string                 { return r.amr }
func (r *refreshTokenRequest) GetAudience() []string            { return r.audience }
func (r *refreshTokenRequest) GetAuthTime() time.Time           { return r.authTime }
func (r *refreshTokenRequest) GetClientID() string              { return r.clientID }
func (r *refreshTokenRequest) GetScopes() []string              { return r.scopes }
func (r *refreshTokenRequest) GetSubject() string               { return r.subject }
func (r *refreshTokenRequest) SetCurrentScopes(scopes []string) { r.scopes = scopes }
func (r *refreshTokenRequest) GetTenantID() string              { return r.tenantID }
func (r *refreshTokenRequest) GetSessionID() string             { return r.sessionID }

// selectedTenantFromRequest 返回请求携带的租户 ID（authorization code 或 refresh token 轮换时从其存储的 tenant 读取）。
// 未设置（TenantID == ""）时返回 0，由调用方决定回退逻辑。
func selectedTenantFromRequest(request op.TokenRequest) string {
	if ar, ok := request.(*AuthRequest); ok {
		return ar.GetTenantID()
	}
	if rr, ok := request.(*refreshTokenRequest); ok {
		return rr.GetTenantID()
	}
	return ""
}

func (s *PersistentStore) TerminateSession(ctx context.Context, userID string, clientID string) error {
	// 自然人级全局登出：显式声明跨全部租户。
	ctx = dbclient.CrossTenantContext(ctx)
	personID, err := ParseSubject(userID)
	if err != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] ParseSubject fail, userID:%s, err:%v", userID, err)
		return nil
	}
	glog.Infof(ctx, "[PersistentStore.TerminateSession] terminating session, userID:%s, personID:%s, clientID:%s", userID, personID, clientID)

	if ssoErr := sso.NewSSOSessionStore().RevokeSessionsByPersonID(ctx, personID); ssoErr != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] revoke SSO sessions fail, err:%v", ssoErr)
	}

	now := time.Now()
	dbErr := s.txDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
		Where("person_id = ?", personID).Where("revoked_at IS NULL").
		Update("revoked_at", &now).Error
	if dbErr != nil {
		glog.Warnf(ctx, "[PersistentStore.TerminateSession] revoke refresh tokens fail, personID:%s, err:%v", personID, dbErr)
	}
	return nil
}

func (s *PersistentStore) RevokeToken(ctx context.Context, tokenOrTokenID string, userID string, clientID string) *oidc.Error {
	// 按 token 行 ID / 自然人 / client_id 撤销：三者都不是租户内定位，显式声明跨全部租户。
	ctx = dbclient.CrossTenantContext(ctx)
	if tokenOrTokenID == "" {
		return nil
	}
	// access token jti（at- 前缀）→ 加入 Redis 黑名单，使 OP 侧 userinfo/introspection 拒绝
	if strings.HasPrefix(tokenOrTokenID, "at-") {
		revokeAccessToken(ctx, tokenOrTokenID)
		return nil
	}
	// refresh token 行 ID → 按主键撤销（RFC 7009：GetRefreshTokenInfo 返回行 ID 后由 op 传入）
	q := s.txDB(ctx).Model(&model.RefreshTokenEntity{}).Table(model.TableNameRefreshToken).
		Where("id = ?", tokenOrTokenID)
	if userID != "" {
		if pid, perr := ParseSubject(userID); perr == nil {
			q = q.Where("person_id = ?", pid)
		}
	}
	if clientID != "" {
		if entity, cErr := s.applicationClientDao().GetByCond(ctx, &dao.ApplicationClientCond{Code: clientID}); cErr == nil && entity != nil && entity.ID != "" {
			q = q.Where("application_client_id = ?", entity.ID)
		}
	}
	now := time.Now()
	if updateErr := q.Update("revoked_at", &now).Error; updateErr != nil {
		glog.Warnf(ctx, "[PersistentStore.RevokeToken] update revoked_at fail, tokenID:%s, err:%v", tokenOrTokenID, updateErr)
	}
	return nil
}

func (s *PersistentStore) GetRefreshTokenInfo(ctx context.Context, clientID string, tokenValue string) (userID string, tokenID string, err error) {
	// refresh token 摘要全局唯一（撤销前需先定位）：显式声明跨全部租户。
	ctx = dbclient.CrossTenantContext(ctx)
	refreshTokenHash := credential.HashSecret(tokenValue)
	storedToken, err := s.refreshTokenDao().GetByCond(ctx, &dao.RefreshTokenCond{Token: refreshTokenHash})
	if err != nil || storedToken == nil || storedToken.ID == "" {
		return "", "", op.ErrInvalidRefreshToken
	}
	if storedToken.RevokedAt != nil {
		return "", "", op.ErrInvalidRefreshToken
	}
	if storedToken.ExpiredAt == nil || !storedToken.ExpiredAt.After(time.Now()) {
		return "", "", op.ErrInvalidRefreshToken
	}
	// RFC 7009 §2.1：token 必须属于发起撤销请求的 client
	if clientID != "" {
		if entity, cErr := s.applicationClientDao().GetByID(ctx, storedToken.ApplicationClientID); cErr == nil && entity != nil && entity.Code != "" {
			if entity.Code != clientID {
				return "", "", op.ErrInvalidRefreshToken
			}
		}
	}
	// 返回行 ID（而非空串）：zitadel 撤销流程会用它作为待撤销 token 传给 RevokeToken
	return BuildSubject(storedToken.PersonID), storedToken.ID, nil
}
