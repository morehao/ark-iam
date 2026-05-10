package svcauth

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"gorm.io/gorm"
)

type ConnectorRuntime struct {
	ID                  uint
	TenantID            uint
	AllowAutoCreateUser bool
}

type identityResolveInput struct {
	Connector ConnectorRuntime
	Identity  StandardIdentity
}

type connectorUserRepository interface {
	GetByID(ctx context.Context, id uint) (*model.UserEntity, error)
	Insert(ctx context.Context, user *model.UserEntity) error
}

type connectorUserIdentityRepository interface {
	GetByConnectorAndExternalSubject(ctx context.Context, tenantID, connectorID uint, externalSubject string) (*model.UserIdentityEntity, error)
	Insert(ctx context.Context, entity *model.UserIdentityEntity) error
	UpdateBinding(ctx context.Context, identityID, userID uint, issuer string, detail []byte) error
}

type connectorTxContextKey struct{}

type connectorDBProvider func(ctx context.Context) *gorm.DB

type connectorTxRunner func(ctx context.Context, fn func(txCtx context.Context) error) error

type identityMapperOption func(*identityMapper)

type identityMapper struct {
	userRepo         connectorUserRepository
	userIdentityRepo connectorUserIdentityRepository
	dbProvider       connectorDBProvider
	txRunner         connectorTxRunner
}

func newIdentityMapper(userRepo connectorUserRepository, userIdentityRepo connectorUserIdentityRepository, opts ...identityMapperOption) *identityMapper {
	mapper := &identityMapper{
		dbProvider: defaultConnectorDBProvider,
	}
	for _, opt := range opts {
		opt(mapper)
	}
	if userRepo == nil {
		userRepo = &connectorUserRepoAdapter{mapper: mapper}
	}
	if userIdentityRepo == nil {
		userIdentityRepo = &connectorUserIdentityRepoAdapter{mapper: mapper}
	}
	mapper.userRepo = userRepo
	mapper.userIdentityRepo = userIdentityRepo
	if mapper.txRunner == nil {
		mapper.txRunner = mapper.runInTransaction
	}
	return mapper
}

func WithIdentityMapperDBProvider(provider connectorDBProvider) identityMapperOption {
	return func(m *identityMapper) {
		if provider != nil {
			m.dbProvider = provider
		}
	}
}

func WithIdentityMapperTxRunner(runner connectorTxRunner) identityMapperOption {
	return func(m *identityMapper) {
		if runner != nil {
			m.txRunner = runner
		}
	}
}

func (m *identityMapper) runInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return m.db(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, connectorTxContextKey{}, tx))
	})
}

func defaultConnectorDBProvider(ctx context.Context) *gorm.DB {
	return dbclient.IamDB(ctx)
}

func connectorRunInTransactionNoop(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

func (m *identityMapper) db(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(connectorTxContextKey{}).(*gorm.DB); ok && tx != nil {
		return tx
	}
	return m.dbProvider(ctx)
}

func (m *identityMapper) repoDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(connectorTxContextKey{}).(*gorm.DB); ok && tx != nil {
		return tx.Session(&gorm.Session{SkipDefaultTransaction: true})
	}
	return m.db(ctx)
}

type connectorUserRepoAdapter struct {
	mapper *identityMapper
}

type connectorUserIdentityRepoAdapter struct {
	mapper *identityMapper
}

func (r *connectorUserRepoAdapter) GetByID(ctx context.Context, id uint) (*model.UserEntity, error) {
	var entity model.UserEntity
	err := r.mapper.repoDB(ctx).First(&entity, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (r *connectorUserRepoAdapter) Insert(ctx context.Context, user *model.UserEntity) error {
	return r.mapper.repoDB(ctx).Create(user).Error
}

func (r *connectorUserIdentityRepoAdapter) GetByConnectorAndExternalSubject(ctx context.Context, tenantID, connectorID uint, externalSubject string) (*model.UserIdentityEntity, error) {
	var entity model.UserIdentityEntity
	err := r.mapper.repoDB(ctx).
		Where("tenant_id = ? AND connector_id = ? AND external_subject = ? AND deleted_at IS NULL", tenantID, connectorID, externalSubject).
		First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (r *connectorUserIdentityRepoAdapter) Insert(ctx context.Context, entity *model.UserIdentityEntity) error {
	return r.mapper.repoDB(ctx).Create(entity).Error
}

func (r *connectorUserIdentityRepoAdapter) UpdateBinding(ctx context.Context, identityID, userID uint, issuer string, detail []byte) error {
	return r.mapper.repoDB(ctx).Model(&model.UserIdentityEntity{}).Where("id = ?", identityID).Updates(map[string]any{
		"user_id":    userID,
		"issuer":     issuer,
		"detail":     json.RawMessage(detail),
		"updated_by": userID,
	}).Error
}

func (m *identityMapper) Resolve(ctx context.Context, input identityResolveInput) (*model.UserEntity, error) {
	existingIdentity, err := m.userIdentityRepo.GetByConnectorAndExternalSubject(ctx, input.Connector.TenantID, input.Connector.ID, input.Identity.Subject)
	if err != nil {
		return nil, err
	}
	if existingIdentity != nil && existingIdentity.UserID != 0 {
		user, getErr := m.userRepo.GetByID(ctx, existingIdentity.UserID)
		if getErr != nil {
			return nil, getErr
		}
		if user != nil {
			return user, nil
		}
	}
	if !input.Connector.AllowAutoCreateUser {
		return nil, code.GetError(code.UserNotExistError)
	}

	user := &model.UserEntity{
		TenantID:     input.Connector.TenantID,
		Username:     resolveIdentityUsername(input.Identity),
		PrimaryEmail: input.Identity.Email,
		Name:         input.Identity.DisplayName,
		Avatar:       input.Identity.AvatarURL,
		Profile:      json.RawMessage("{}"),
		Identities:   json.RawMessage("{}"),
		CustomData:   json.RawMessage("{}"),
		CreatedBy:    0,
	}
	if err := m.txRunner(ctx, func(txCtx context.Context) error {
		if err := m.userRepo.Insert(txCtx, user); err != nil {
			return err
		}
		if err := m.bindIdentity(txCtx, input.Connector, user, input.Identity, existingIdentity); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return user, nil
}

func (m *identityMapper) bindIdentity(ctx context.Context, connector ConnectorRuntime, user *model.UserEntity, identity StandardIdentity, existingIdentity *model.UserIdentityEntity) error {
	detail, err := json.Marshal(identity)
	if err != nil {
		return err
	}
	if existingIdentity != nil {
		return m.userIdentityRepo.UpdateBinding(ctx, existingIdentity.ID, user.ID, identity.Issuer, detail)
	}
	return m.userIdentityRepo.Insert(ctx, &model.UserIdentityEntity{
		TenantID:        connector.TenantID,
		UserID:          user.ID,
		ConnectorID:     connector.ID,
		Issuer:          identity.Issuer,
		ExternalSubject: identity.Subject,
		Detail:          detail,
		CreatedBy:       0,
	})
}

func resolveIdentityUsername(identity StandardIdentity) string {
	if identity.Username != "" {
		return identity.Username
	}
	if identity.Email != "" {
		return identity.Email
	}
	return identity.Subject
}
