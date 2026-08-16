package oidcop

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/crypto"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/morehao/ark-iam/pkg/iam/sso"
	"github.com/morehao/golib/glog"
)

const (
	logoutTokenLifetime = 15 * time.Minute
	// http 超时与重试配置
	logoutSendTimeout = 10 * time.Second
	logoutMaxAttempts = 3
	logoutBackoffStep = time.Second
	logoutJobTTL      = 10 * time.Minute
)

// logoutWorker 消费登出队列，异步生成并发送 back-channel logout_token。
// 它在 auth（OP）进程内运行，是 SLO 的通知执行端。
type logoutWorker struct {
	signer     jose.Signer
	issuer     string
	httpClient *http.Client
}

// NewLogoutWorker 构造背信道登出 worker。privKey 为 OP 的令牌签名私钥（同 ID token 签名）。
func NewLogoutWorker(privKey *rsa.PrivateKey, keyID string, issuer string) *logoutWorker {
	signer := newRSASigner(privKey, keyID)
	return &logoutWorker{
		signer:     signer,
		issuer:     issuer,
		httpClient: &http.Client{Timeout: logoutSendTimeout},
	}
}

// Run 起一个常驻消费者，阻塞地从队列取任务并发送，直至 ctx 取消。
// Redis 故障时退避重试，避免忙循环打爆 Redis。
func (w *logoutWorker) Run(ctx context.Context) {
	for {
		job, ok, err := sso.DequeueLogout(ctx, 2*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			glog.Warnf(ctx, "[logoutWorker.Run] dequeue logout fail, err:%v", err)
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				return
			}
			continue
		}
		if !ok {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		w.handleJob(ctx, job)
	}
}

func (w *logoutWorker) handleJob(ctx context.Context, job sso.LogoutJob) {
	if w.expired(job) {
		return
	}
	for attempt := 1; attempt <= logoutMaxAttempts; attempt++ {
		err := w.send(ctx, job)
		if err == nil {
			// 通知成功，删除登记以幂等
			if job.SessionID != "" {
				_ = sso.NewSLOStore().Delete(context.Background(), job.SessionID, job.OIDCSessionID)
			}
			return
		}
		glog.Warnf(ctx, "[logoutWorker.handleJob] send logout fail, attempt:%d/%d, clientID:%s, err:%v", attempt, logoutMaxAttempts, job.ClientID, err)
		if attempt < logoutMaxAttempts {
			time.Sleep(logoutBackoffStep * time.Duration(attempt))
		}
	}
}

// expired 丢弃超过 TTL 的过期任务（对齐 Zitadel MaxTtl，避免积压陈旧通知）。
// 依据 job 内嵌的创建时间戳判断，与 worker 进程生命周期解耦，多副本/长驻运行判定一致。
func (w *logoutWorker) expired(job sso.LogoutJob) bool {
	if job.CreatedAt.IsZero() {
		return false
	}
	return time.Since(job.CreatedAt) > logoutJobTTL
}

func (w *logoutWorker) send(ctx context.Context, job sso.LogoutJob) error {
	logoutToken, err := w.buildLogoutToken(job)
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Set("logout_token", logoutToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, job.BackChannelLogoutURI, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() {
		if cErr := resp.Body.Close(); cErr != nil {
			glog.Warnf(ctx, "[logoutWorker.send] close response body fail, err:%v", cErr)
		}
	}()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

// buildLogoutToken 构造标准 logout_token（含 events.backchannel-logout 与 sid）。
func (w *logoutWorker) buildLogoutToken(job sso.LogoutJob) (string, error) {
	jti, err := newJTI()
	if err != nil {
		return "", err
	}
	sub := job.UserID
	if sub == "" {
		sub = fmt.Sprintf("person:%s", job.PersonID)
	}
	claims := oidc.NewLogoutTokenClaims(
		w.issuer,
		sub,
		oidc.Audience{job.ClientID},
		time.Now().Add(logoutTokenLifetime),
		jti,
		job.SessionID, // sid：中心会话 ID；M3 阶段可由发起方 id_token_hint 携带
		time.Second,
	)
	return crypto.Sign(claims, w.signer)
}

func newRSASigner(privKey *rsa.PrivateKey, keyID string) jose.Signer {
	opts := &jose.SignerOptions{}
	if keyID != "" {
		opts.WithHeader("kid", keyID)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privKey}, opts)
	if err != nil {
		panic(fmt.Sprintf("new rsa signer fail: %v", err))
	}
	return signer
}

func newJTI() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate jti: %w", err)
	}
	return hex.EncodeToString(b), nil
}
