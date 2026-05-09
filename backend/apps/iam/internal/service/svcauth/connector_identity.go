package svcauth

import (
	"context"
	"encoding/json"

	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/pkg/code"
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

type identityMapper struct {
	userRepo         connectorUserRepository
	userIdentityRepo connectorUserIdentityRepository
}

func newIdentityMapper(userRepo connectorUserRepository, userIdentityRepo connectorUserIdentityRepository) *identityMapper {
	if userRepo == nil {
		userRepo = dao.NewUserDao()
	}
	if userIdentityRepo == nil {
		userIdentityRepo = dao.NewUserIdentityDao()
	}
	return &identityMapper{userRepo: userRepo, userIdentityRepo: userIdentityRepo}
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
		CreatedBy:    0,
	}
	if err := m.userRepo.Insert(ctx, user); err != nil {
		return nil, err
	}
	if err := m.bindIdentity(ctx, input.Connector, user, input.Identity, existingIdentity); err != nil {
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
