package svcoidc

import (
	"context"
	"errors"
	"time"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

var (
	ErrStoreUnavailable    = errors.New("oidc protocol state store unavailable")
	ErrSessionNotFound     = errors.New("auth request not found")
	ErrCodeInvalid         = errors.New("authorization code invalid")
	ErrCodeAlreadyUsed     = errors.New("authorization code already used")
	ErrCodeCollision       = errors.New("authorization code collision")
	ErrSessionNotCompleted = errors.New("auth request not completed")
)

type ProtocolStateStore interface {
	CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, userID string) (op.AuthRequest, error)
	AuthRequestByID(ctx context.Context, id string) (op.AuthRequest, error)
	AuthRequestByCode(ctx context.Context, code string) (op.AuthRequest, error)
	SaveAuthCode(ctx context.Context, id, code string) error
	CompleteAuthRequest(id string, subject string, authTime time.Time, amr []string, acr string) error
	DeleteAuthRequest(ctx context.Context, id string) error
	Health(ctx context.Context) error
}
