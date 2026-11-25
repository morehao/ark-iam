package appassembly

// 本文件提供 pkg 内测试用的轻量验签实现（contract.KeySource + contract.Verifier），
// 使 pkg/appassembly 的测试**不 import sdk**（保持「pkg 不依赖 sdk」的依赖方向，与
// pkg/oidckit/keys_test.go 的 fakeKeys 同一做法）。行为对齐 sdk/rp 的必要子集：
// RS256-only、kid→keySource、exp 必填、iss/aud 收紧、person/machine 必需 claim。

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/ark-iam/contract"
)

// fakeKeySource 是 contract.KeySource 的内存实现（按 kid 返回公钥，零网络）。
type fakeKeySource struct{ m map[string]*rsa.PublicKey }

func newFakeKeySource(m map[string]*rsa.PublicKey) fakeKeySource { return fakeKeySource{m: m} }

func (f fakeKeySource) PublicKey(_ context.Context, kid string) (*rsa.PublicKey, error) {
	if k, ok := f.m[kid]; ok && k != nil {
		return k, nil
	}
	return nil, contract.ErrUnknownKid
}

// testVerifier 是 contract.Verifier 的测试实现（jwt/v5 + contract.TokenClaims）。
type testVerifier struct {
	ks        contract.KeySource
	issuer    string
	audiences []string
}

// buildTestVerifier 构造与 pkg/middleware.VerifierBuilder 同签名的测试验签器构造器。
func buildTestVerifier(ks contract.KeySource, issuer string, audiences []string) contract.Verifier {
	return &testVerifier{ks: ks, issuer: issuer, audiences: audiences}
}

func (v *testVerifier) Verify(_ context.Context, raw string) (*contract.Identity, error) {
	if v.ks == nil {
		return nil, contract.ErrKeySourceUnavailable
	}
	if strings.TrimSpace(raw) == "" {
		return nil, contract.ErrMalformedToken
	}
	claims := &contract.TokenClaims{}
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30 * time.Second),
	}
	if v.issuer != "" {
		opts = append(opts, jwt.WithIssuer(v.issuer))
	}
	if len(v.audiences) > 0 {
		opts = append(opts, jwt.WithAudience(v.audiences...))
	}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, contract.ErrMissingKid
		}
		return v.ks.PublicKey(context.Background(), kid)
	}, opts...)
	if err != nil {
		return nil, classifyTestJWTError(err)
	}
	if !token.Valid {
		return nil, contract.ErrBadSignature
	}
	if !claims.HasPerson() && !claims.IsMachine() {
		return nil, fmt.Errorf("%w: neither person nor machine token", contract.ErrMissingClaim)
	}
	scopes := strings.Fields(strings.TrimSpace(claims.Scope))
	var actorID string
	if claims.Actor != nil {
		actorID, _ = claims.Actor["sub"].(string)
	}
	return &contract.Identity{
		PersonID:  claims.PersonID(),
		TenantID:  claims.TenantID,
		UserID:    claims.UserID,
		ClientID:  claims.ClientID,
		IsMachine: claims.IsMachine(),
		SessionID: claims.SessionID,
		Scopes:    scopes,
		ActorID:   actorID,
	}, nil
}

// classifyTestJWTError 把 jwt/v5 错误归一到 contract sentinel（与 rp 对齐的测试子集）。
func classifyTestJWTError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return fmt.Errorf("%w: %v", contract.ErrExpired, err)
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return fmt.Errorf("%w: %v", contract.ErrNotYetValid, err)
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return fmt.Errorf("%w: %v", contract.ErrBadSignature, err)
	case errors.Is(err, jwt.ErrTokenInvalidIssuer):
		return fmt.Errorf("%w: %v", contract.ErrBadIssuer, err)
	case errors.Is(err, jwt.ErrTokenInvalidAudience):
		return fmt.Errorf("%w: %v", contract.ErrBadAudience, err)
	case errors.Is(err, jwt.ErrTokenUnverifiable), errors.Is(err, contract.ErrUnknownKid):
		return contract.ErrUnknownKid
	default:
		return fmt.Errorf("%w: %v", contract.ErrMalformedToken, err)
	}
}
