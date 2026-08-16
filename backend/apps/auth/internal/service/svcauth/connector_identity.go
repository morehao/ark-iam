package svcauth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/dbclient"
	"github.com/morehao/ark-iam/pkg/iam/model"
	"gorm.io/gorm"
)

type ConnectorRuntime struct {
	ID                  string
	TenantID            string
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
	GetByID(ctx context.Context, id string) (*model.PersonEntity, error)
	Insert(ctx context.Context, person *model.PersonEntity) error
}

type connectorUserIdentityRepository interface {
	GetByIssuerAndExternalSubject(ctx context.Context, issuer, externalSubject string) (*model.UserIdentityEntity, error)
	Insert(ctx context.Context, entity *model.UserIdentityEntity) error
	UpdateBinding(ctx context.Context, identityID, personID string, issuer string, detail []byte) error
}

// connectorUserRepository 负责写入租户成员关系（UserEntity）。
// 自动创建用户时若不同时建立成员关系，登录后续流程（listPersonTenants）将查不到租户而失败（S4）。
type connectorUserRepository interface {
	Insert(ctx context.Context, user *model.UserEntity) error
}

type connectorTxContextKey struct{}

type connectorDBProvider func(ctx context.Context) *gorm.DB

type connectorTxRunner func(ctx context.Context, fn func(txCtx context.Context) error) error

type identityMapperOption func(*identityMapper)

type identityMapper struct {
	personRepo       connectorPersonRepository
	userIdentityRepo connectorUserIdentityRepository
	userRepo         connectorUserRepository
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
	if mapper.userRepo == nil {
		mapper.userRepo = &connectorUserRepoAdapter{mapper: mapper}
	}
	mapper.personRepo = personRepo
	mapper.userIdentityRepo = userIdentityRepo
	if mapper.txRunner == nil {
		mapper.txRunner = mapper.runInTransaction
	}
	return mapper
}

// WithIdentityMapperUserRepository 注入租户成员仓库（测试注入桩用）。
func WithIdentityMapperUserRepository(repo connectorUserRepository) identityMapperOption {
	return func(m *identityMapper) {
		if repo != nil {
			m.userRepo = repo
		}
	}
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

type connectorUserRepoAdapter struct {
	mapper *identityMapper
}

func (r *connectorPersonRepoAdapter) GetByID(ctx context.Context, id string) (*model.PersonEntity, error) {
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

func (r *connectorUserIdentityRepoAdapter) UpdateBinding(ctx context.Context, identityID, personID string, issuer string, detail []byte) error {
	updateMap := map[string]any{
		"person_id": personID,
		"issuer":    issuer,
		"detail":    json.RawMessage(detail),
	}
	return r.mapper.repoDB(ctx).Model(&model.UserIdentityEntity{}).Where("id = ?", identityID).Updates(updateMap).Error
}

// connectorUserRepoAdapter 写入租户成员（UserEntity）。
func (r *connectorUserRepoAdapter) Insert(ctx context.Context, user *model.UserEntity) error {
	return r.mapper.repoDB(ctx).Create(user).Error
}

func (m *identityMapper) Resolve(ctx context.Context, input identityResolveInput) (*resolvedConnectorPerson, error) {
	existingIdentity, err := m.userIdentityRepo.GetByIssuerAndExternalSubject(ctx, input.Identity.Issuer, input.Identity.Subject)
	if err != nil {
		return nil, err
	}
	if existingIdentity != nil && existingIdentity.PersonID != "" {
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
		Username:     model.StrPtr(resolveIdentityUsername(input.Identity)),
		PrimaryEmail: model.StrPtr(input.Identity.Email),
		Name:         input.Identity.DisplayName,
		Avatar:       input.Identity.AvatarURL,
		Profile:      json.RawMessage("{}"),
		CustomData:   json.RawMessage("{}"),
		CreatedBy:    "",
	}
	if person.Name == "" {
		person.Name = resolveIdentityUsername(input.Identity)
	}
	if err := m.txRunner(ctx, func(txCtx context.Context) error {
		if err := m.personRepo.Insert(txCtx, person); err != nil {
			return err
		}
		// S4：自动创建的用户必须同事务建立租户成员关系（UserEntity），
		// 否则后续登录流程（Callback → listPersonTenants）查不到成员而失败。
		now := time.Now()
		user := &model.UserEntity{
			TenantID:   input.Connector.TenantID,
			PersonID:   person.ID,
			Name:       person.Name,
			Profile:    json.RawMessage("{}"),
			CustomData: json.RawMessage("{}"),
			JoinedAt:   &now,
			CreatedBy:  "",
		}
		if err := m.userRepo.Insert(txCtx, user); err != nil {
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
		CreatedBy:       "",
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
