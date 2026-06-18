package svcauth

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/morehao/ark-iam/pkg/iam/model"
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

type resolvedConnectorPerson struct {
	Person *model.PersonEntity
}

type connectorPersonRepository interface {
	GetByID(ctx context.Context, id uint) (*model.PersonEntity, error)
	Insert(ctx context.Context, person *model.PersonEntity) error
}

type connectorUserIdentityRepository interface {
	GetByIssuerAndExternalSubject(ctx context.Context, issuer, externalSubject string) (*model.UserIdentityEntity, error)
	Insert(ctx context.Context, entity *model.UserIdentityEntity) error
	UpdateBinding(ctx context.Context, identityID, personID uint, issuer string, detail []byte) error
}

type connectorTxContextKey struct{}

type connectorDBProvider func(ctx context.Context) *gorm.DB

type connectorTxRunner func(ctx context.Context, fn func(txCtx context.Context) error) error

type identityMapperOption func(*identityMapper)

type identityMapper struct {
	personRepo       connectorPersonRepository
	userIdentityRepo connectorUserIdentityRepository
	dbProvider       connectorDBProvider
	txRunner         connectorTxRunner
}

func newIdentityMapper(personRepo connectorPersonRepository, userIdentityRepo connectorUserIdentityRepository, opts ...identityMapperOption) *identityMapper {
	mapper := &identityMapper{
		dbProvider: defaultConnectorDBProvider,
	}
	for _, opt := range opts {
		opt(mapper)
	}
	if personRepo == nil {
		personRepo = &connectorPersonRepoAdapter{mapper: mapper}
	}
	if userIdentityRepo == nil {
		userIdentityRepo = &connectorUserIdentityRepoAdapter{mapper: mapper}
	}
	mapper.personRepo = personRepo
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

type connectorPersonRepoAdapter struct {
	mapper *identityMapper
}

type connectorUserIdentityRepoAdapter struct {
	mapper *identityMapper
}

func (r *connectorPersonRepoAdapter) GetByID(ctx context.Context, id uint) (*model.PersonEntity, error) {
	var entity model.PersonEntity
	err := r.mapper.repoDB(ctx).First(&entity, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

func (r *connectorPersonRepoAdapter) Insert(ctx context.Context, person *model.PersonEntity) error {
	return r.mapper.repoDB(ctx).Create(person).Error
}

func (r *connectorUserIdentityRepoAdapter) GetByIssuerAndExternalSubject(ctx context.Context, issuer, externalSubject string) (*model.UserIdentityEntity, error) {
	var entity model.UserIdentityEntity
	err := r.mapper.repoDB(ctx).
		Where("issuer = ? AND external_subject = ? AND deleted_at IS NULL", issuer, externalSubject).
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

func (r *connectorUserIdentityRepoAdapter) UpdateBinding(ctx context.Context, identityID, personID uint, issuer string, detail []byte) error {
	updateMap := map[string]any{
		"person_id":  personID,
		"issuer":     issuer,
		"detail":     json.RawMessage(detail),
	}
	return r.mapper.repoDB(ctx).Model(&model.UserIdentityEntity{}).Where("id = ?", identityID).Updates(updateMap).Error
}

func (m *identityMapper) Resolve(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error) {
	existingIdentity, err := m.userIdentityRepo.GetByIssuerAndExternalSubject(ctx, input.Identity.Issuer, input.Identity.Subject)
	if err != nil {
		return nil, err
	}
	if existingIdentity != nil && existingIdentity.PersonID != 0 {
		person, getErr := m.personRepo.GetByID(ctx, existingIdentity.PersonID)
		if getErr != nil {
			return nil, getErr
		}
		if person != nil {
			return &resolvedConnectorPerson{Person: person}, nil
		}
	}
	if !input.Connector.AllowAutoCreateUser {
		return nil, code.GetError(code.UserNotExistError)
	}

	person := &model.PersonEntity{
		Username:     resolveIdentityUsername(input.Identity),
		PrimaryEmail: input.Identity.Email,
		Name:         input.Identity.DisplayName,
		Avatar:       input.Identity.AvatarURL,
		Profile:      json.RawMessage("{}"),
		CustomData:   json.RawMessage("{}"),
		CreatedBy:    0,
	}
	if person.Name == "" {
		person.Name = resolveIdentityUsername(input.Identity)
	}
	if err := m.txRunner(ctx, func(txCtx context.Context) error {
		if err := m.personRepo.Insert(txCtx, person); err != nil {
			return err
		}
		if err := m.bindIdentity(txCtx, input.Connector, person, input.Identity, existingIdentity); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &resolvedConnectorPerson{Person: person}, nil
}

func (m *identityMapper) bindIdentity(ctx context.Context, connector ConnectorRuntime, person *model.PersonEntity, identity StandardIdentity, existingIdentity *model.UserIdentityEntity) error {
	_ = connector
	detail, err := json.Marshal(identity)
	if err != nil {
		return err
	}
	if existingIdentity != nil {
		return m.userIdentityRepo.UpdateBinding(ctx, existingIdentity.ID, person.ID, identity.Issuer, detail)
	}
	return m.userIdentityRepo.Insert(ctx, &model.UserIdentityEntity{
		PersonID:        person.ID,
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
