package svcauth

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/morehao/ark-iam/iam/dao"
	"github.com/morehao/ark-iam/iam/internal/dto/dtoauth"
	"github.com/morehao/ark-iam/iam/model"
	"github.com/morehao/ark-iam/iam/object/objauth"
	"github.com/morehao/ark-iam/pkg/code"
	"github.com/morehao/ark-iam/pkg/token"
	"github.com/morehao/golib/biz/gconstant"
	"github.com/morehao/golib/biz/gcontext/gincontext"
	"github.com/morehao/golib/biz/genericdao"
	"github.com/morehao/golib/biz/gobject"
	"github.com/morehao/golib/gauth/jwtauth"
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
)

const (
	PasswordMinLength          = 6
	TokenExpireDuration        = 24 * time.Hour
	RefreshTokenExpireDuration = 7 * 24 * time.Hour
)

type authUserStore interface {
	GetByID(ctx context.Context, id uint) (*model.UserEntity, error)
	GetByCond(ctx context.Context, cond genericdao.Cond) (*model.UserEntity, error)
	GetListByCond(ctx context.Context, cond genericdao.Cond) (model.UserEntityList, error)
	Insert(ctx context.Context, entity *model.UserEntity) error
	UpdateMap(ctx context.Context, id uint, updateMap map[string]interface{}) error
}

type authPersonStore interface {
	GetByID(ctx context.Context, id uint) (*model.PersonEntity, error)
	GetByCond(ctx context.Context, cond genericdao.Cond) (*model.PersonEntity, error)
	Insert(ctx context.Context, entity *model.PersonEntity) error
}

type authTenantStore interface {
	GetByID(ctx context.Context, id uint) (*model.TenantEntity, error)
	GetPageListByCond(ctx context.Context, cond genericdao.Cond) (model.TenantEntityList, int64, error)
}

type authRefreshTokenStore interface {
	GetByCond(ctx context.Context, cond genericdao.Cond) (*model.RefreshTokenEntity, error)
	Insert(ctx context.Context, entity *model.RefreshTokenEntity) error
	Delete(ctx context.Context, id, userID uint) error
}

var newAuthUserStore = func() authUserStore {
	return dao.NewUserDao()
}

var newAuthPersonStore = func() authPersonStore {
	return dao.NewPersonDao()
}

var newAuthTenantStore = func() authTenantStore {
	return dao.NewTenantDao()
}

var newAuthRefreshTokenStore = func() authRefreshTokenStore {
	return dao.NewRefreshTokenDao()
}

var authLoginRecorder = func(ctx *gin.Context, tenantID, userID uint, success bool) {
	defaultRecordLoginLog(ctx, tenantID, userID, success)
}

type AuthSvc interface {
	Login(ctx *gin.Context, req *dtoauth.LoginReq) (*dtoauth.LoginResp, error)
	AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, error)
	SelectTenant(ctx *gin.Context, req *dtoauth.SelectTenantReq) (*dtoauth.SelectTenantResp, error)
	SwitchTenant(ctx *gin.Context, req *dtoauth.SwitchTenantReq) (*dtoauth.SwitchTenantResp, error)
	MyTenants(ctx *gin.Context, req *dtoauth.MyTenantsReq) (*dtoauth.MyTenantsResp, error)
	Register(ctx *gin.Context, req *dtoauth.RegisterReq) (*dtoauth.RegisterResp, error)
	JoinTenant(ctx *gin.Context, req *dtoauth.JoinTenantReq) (*dtoauth.JoinTenantResp, error)
	RefreshToken(ctx *gin.Context, req *dtoauth.RefreshTokenReq) (*dtoauth.RefreshTokenResp, error)
	Logout(ctx *gin.Context, req *dtoauth.LogoutReq) error
	LogoutAll(ctx *gin.Context, req *dtoauth.LogoutAllReq) error
	Userinfo(ctx *gin.Context, req *dtoauth.UserinfoReq) (*dtoauth.UserinfoResp, error)
}

type authSvc struct {
	jwtSecret string
}

var _ AuthSvc = (*authSvc)(nil)

func NewAuthSvc(jwtSecret string) AuthSvc {
	return &authSvc{
		jwtSecret: jwtSecret,
	}
}

func (svc *authSvc) Login(ctx *gin.Context, req *dtoauth.LoginReq) (*dtoauth.LoginResp, error) {
	personDao := newAuthPersonStore()
	userDao := newAuthUserStore()
	personEntity, userEntity, tenants, err := svc.resolvePersonLogin(ctx, personDao, userDao, req.Identifier)
	if err != nil {
		return nil, err
	}
	personEntity, userEntity, err = svc.authenticateResolvedPerson(ctx, personEntity, userEntity, req.Password)
	if err != nil {
		return nil, err
	}

	personToken, err := svc.generatePersonToken(personEntity)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Login] generatePersonToken fail, err:%v", err)
		return nil, code.GetError(code.TokenGenerateError)
	}

	authLoginRecorder(ctx, userEntity.TenantID, userEntity.ID, true)

	return &dtoauth.LoginResp{
		PersonToken: *personToken,
		Tenants:     tenants,
	}, nil
}

func (svc *authSvc) AuthenticatePassword(ctx *gin.Context, identifier, password string) (*model.PersonEntity, *model.UserEntity, error) {
	personDao := newAuthPersonStore()
	userDao := newAuthUserStore()
	personEntity, userEntity, _, err := svc.resolvePersonLogin(ctx, personDao, userDao, identifier)
	if err != nil {
		return nil, nil, err
	}
	return svc.authenticateResolvedPerson(ctx, personEntity, userEntity, password)
}

func (svc *authSvc) authenticateResolvedPerson(ctx *gin.Context, personEntity *model.PersonEntity, userEntity *model.UserEntity, password string) (*model.PersonEntity, *model.UserEntity, error) {
	if personEntity.IsSuspended == 1 {
		return nil, nil, code.GetError(code.UserSuspendedError)
	}

	if personEntity.PasswordEncrypted == "" {
		return nil, nil, code.GetError(code.PasswordNotSetError)
	}

	if err := gcrypto.ComparePasswordHash(personEntity.PasswordEncrypted, password); err != nil {
		authLoginRecorder(ctx, userEntity.TenantID, userEntity.ID, false)
		glog.Errorf(ctx, "[svcauth.authenticateResolvedPerson] password mismatch, personID:%d", personEntity.ID)
		return nil, nil, code.GetError(code.PasswordMismatchError)
	}

	authLoginRecorder(ctx, userEntity.TenantID, userEntity.ID, true)
	return personEntity, userEntity, nil
}

func (svc *authSvc) SelectTenant(ctx *gin.Context, req *dtoauth.SelectTenantReq) (*dtoauth.SelectTenantResp, error) {
	personID := gincontext.GetPersonID(ctx)
	if personID == 0 {
		personID = svc.personIDFromToken(req.PersonToken)
	}
	if personID == 0 {
		return nil, code.GetError(gconstant.UnauthorizedErr)
	}

	tokenInfo, err := svc.issueTenantToken(ctx, personID, req.TenantID)
	if err != nil {
		return nil, err
	}

	return &dtoauth.SelectTenantResp{TokenInfo: *tokenInfo}, nil
}

func (svc *authSvc) SwitchTenant(ctx *gin.Context, req *dtoauth.SwitchTenantReq) (*dtoauth.SwitchTenantResp, error) {
	personID := gincontext.GetPersonID(ctx)
	if personID == 0 {
		return nil, code.GetError(gconstant.UnauthorizedErr)
	}

	tokenInfo, err := svc.issueTenantToken(ctx, personID, req.TenantID)
	if err != nil {
		return nil, err
	}

	return &dtoauth.SwitchTenantResp{TokenInfo: *tokenInfo}, nil
}

func (svc *authSvc) MyTenants(ctx *gin.Context, req *dtoauth.MyTenantsReq) (*dtoauth.MyTenantsResp, error) {
	personID := gincontext.GetPersonID(ctx)
	if personID == 0 {
		personID = svc.personIDFromToken(req.PersonToken)
	}
	if personID == 0 {
		return nil, code.GetError(gconstant.UnauthorizedErr)
	}

	_, tenants, err := svc.listPersonTenants(ctx, personID)
	if err != nil {
		return nil, err
	}

	return &dtoauth.MyTenantsResp{List: tenants}, nil
}

func (svc *authSvc) Register(ctx *gin.Context, req *dtoauth.RegisterReq) (*dtoauth.RegisterResp, error) {
	if err := validatePasswordStrength(req.Password); err != nil {
		return nil, code.GetError(code.PasswordValidationError)
	}

	if req.Username == "" && req.PrimaryEmail == "" && req.PrimaryPhone == "" {
		return nil, code.GetError(code.AuthIdentifierRequiredError)
	}

	personDao := newAuthPersonStore()
	userDao := newAuthUserStore()
	tenantDao := newAuthTenantStore()

	tenantEntity, err := tenantDao.GetByID(ctx.Request.Context(), req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Register] tenant dao GetByID fail, err:%v, tenantID:%d", err, req.TenantID)
		return nil, code.GetError(code.TenantGetDetailError)
	}
	if tenantEntity == nil || tenantEntity.ID == 0 {
		return nil, code.GetError(code.TenantNotExistError)
	}

	if req.Username != "" {
		existingPerson, _ := personDao.GetByCond(ctx.Request.Context(), &dao.PersonCond{
			Username: req.Username,
		})
		if existingPerson != nil && existingPerson.ID != 0 {
			return nil, code.GetError(code.UsernameAlreadyExistsError)
		}
	}

	if req.PrimaryEmail != "" {
		existingPerson, _ := personDao.GetByCond(ctx.Request.Context(), &dao.PersonCond{
			PrimaryEmail: req.PrimaryEmail,
		})
		if existingPerson != nil && existingPerson.ID != 0 {
			return nil, code.GetError(code.EmailAlreadyExistsError)
		}
	}

	if req.PrimaryPhone != "" {
		existingPerson, _ := personDao.GetByCond(ctx.Request.Context(), &dao.PersonCond{
			PrimaryPhone: req.PrimaryPhone,
		})
		if existingPerson != nil && existingPerson.ID != 0 {
			return nil, code.GetError(code.PhoneAlreadyExistsError)
		}
	}

	passwordHash, err := gcrypto.GeneratePasswordHash(req.Password)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Register] GeneratePasswordHash fail, err:%v", err)
		return nil, code.GetError(code.PasswordHashError)
	}

	personEntity := &model.PersonEntity{
		Username:          req.Username,
		PrimaryEmail:      req.PrimaryEmail,
		PrimaryPhone:      req.PrimaryPhone,
		PasswordEncrypted: passwordHash,
		PasswordMethod:    "bcrypt",
		Name:              req.Name,
		CreatedBy:         0,
	}
	if err := personDao.Insert(ctx.Request.Context(), personEntity); err != nil {
		glog.Errorf(ctx, "[svcauth.Register] person dao Insert fail, err:%v", err)
		return nil, code.GetError(code.UserCreateError)
	}
	now := time.Now()
	insertEntity := &model.UserEntity{
		TenantID:  req.TenantID,
		PersonID:  personEntity.ID,
		Name:      req.Name,
		IsOwner:   1,
		JoinedAt:  &now,
		CreatedBy: 0,
	}

	if err := userDao.Insert(ctx.Request.Context(), insertEntity); err != nil {
		glog.Errorf(ctx, "[svcauth.Register] dao Insert fail, err:%v", err)
		return nil, code.GetError(code.UserCreateError)
	}

	return &dtoauth.RegisterResp{
		UserID: insertEntity.ID,
	}, nil
}

func (svc *authSvc) JoinTenant(ctx *gin.Context, req *dtoauth.JoinTenantReq) (*dtoauth.JoinTenantResp, error) {
	personID := gincontext.GetPersonID(ctx)
	if personID == 0 {
		return nil, code.GetError(gconstant.UnauthorizedErr)
	}

	tenantDao := newAuthTenantStore()
	userDao := newAuthUserStore()

	tenantEntity, err := tenantDao.GetByID(ctx.Request.Context(), req.TenantID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.JoinTenant] tenant dao GetByID fail, err:%v, tenantID:%d", err, req.TenantID)
		return nil, code.GetError(code.TenantGetDetailError)
	}
	if tenantEntity == nil || tenantEntity.ID == 0 {
		return nil, code.GetError(code.TenantNotExistError)
	}

	existingUser, err := userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{PersonID: personID, TenantID: req.TenantID})
	if err != nil {
		glog.Errorf(ctx, "[svcauth.JoinTenant] user dao GetByCond fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if existingUser != nil && existingUser.ID != 0 {
		return nil, code.GetError(code.AlreadyJoinedError)
	}

	now := time.Now()
	userEntity := &model.UserEntity{
		TenantID:  req.TenantID,
		PersonID:  personID,
		Name:      "",
		IsOwner:   0,
		JoinedAt:  &now,
		CreatedBy: 0,
	}
	if err := userDao.Insert(ctx.Request.Context(), userEntity); err != nil {
		glog.Errorf(ctx, "[svcauth.JoinTenant] user dao Insert fail, err:%v", err)
		return nil, code.GetError(code.UserCreateError)
	}

	return &dtoauth.JoinTenantResp{
		UserID: userEntity.ID,
	}, nil
}

func (svc *authSvc) RefreshToken(ctx *gin.Context, req *dtoauth.RefreshTokenReq) (*dtoauth.RefreshTokenResp, error) {
	if req.RefreshToken == "" {
		return nil, code.GetError(code.RefreshTokenRequiredError)
	}

	tokenClaims, err := svc.parseRefreshToken(req.RefreshToken)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.RefreshToken] parseRefreshToken fail, err:%v", err)
		return nil, code.GetError(gconstant.TokenInvalidErr)
	}

	userID, ok := parsePositiveIntegerClaim(tokenClaims, "user_id")
	if !ok {
		return nil, code.GetError(gconstant.TokenInvalidErr)
	}
	tenantID, ok := parsePositiveIntegerClaim(tokenClaims, "tenant_id")
	if !ok {
		return nil, code.GetError(gconstant.TokenInvalidErr)
	}

	refreshTokenDao := newAuthRefreshTokenStore()
	storedToken, err := refreshTokenDao.GetByCond(ctx.Request.Context(), &dao.RefreshTokenCond{
		UserID: uint(userID),
		Token:  token.HashToken(req.RefreshToken),
	})
	if err != nil || storedToken == nil {
		return nil, code.GetError(code.RefreshTokenInvalidError)
	}
	if storedToken.TenantID != uint(tenantID) {
		return nil, code.GetError(code.RefreshTokenInvalidError)
	}
	if storedToken.RevokedAt != nil {
		return nil, code.GetError(code.RefreshTokenInvalidError)
	}
	if storedToken.ExpiredAt == nil || !storedToken.ExpiredAt.After(time.Now()) {
		return nil, code.GetError(code.RefreshTokenInvalidError)
	}

	userDao := newAuthUserStore()
	userEntity, err := userDao.GetByID(ctx.Request.Context(), uint(userID))
	if err != nil || userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	tokenInfo, err := svc.generateToken(ctx, userEntity)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.RefreshToken] generateToken fail, err:%v", err)
		return nil, code.GetError(code.TokenGenerateError)
	}

	if err := refreshTokenDao.Delete(ctx.Request.Context(), storedToken.ID, storedToken.UserID); err != nil {
		glog.Errorf(ctx, "[svcauth.RefreshToken] delete old refreshToken fail, err:%v", err)
	}

	return &dtoauth.RefreshTokenResp{
		TokenInfo: *tokenInfo,
	}, nil
}

func (svc *authSvc) Logout(ctx *gin.Context, req *dtoauth.LogoutReq) error {
	if req.RefreshToken != "" {
		if err := token.AddRefreshTokenToBlacklist(ctx, req.RefreshToken); err != nil {
			glog.Errorf(ctx, "[svcauth.Logout] AddRefreshTokenToBlacklist fail, err:%v", err)
		}
	}

	accessToken := ctx.GetHeader("Authorization")
	if accessToken != "" {
		if err := token.AddTokenToBlacklist(ctx, accessToken, token.TokenExpireDuration); err != nil {
			glog.Errorf(ctx, "[svcauth.Logout] AddTokenToBlacklist fail, err:%v", err)
		}
	}

	return nil
}

func (svc *authSvc) LogoutAll(ctx *gin.Context, req *dtoauth.LogoutAllReq) error {
	return svc.Logout(ctx, &dtoauth.LogoutReq{RefreshToken: req.RefreshToken})
}

func (svc *authSvc) Userinfo(ctx *gin.Context, req *dtoauth.UserinfoReq) (*dtoauth.UserinfoResp, error) {
	userDao := newAuthUserStore()

	userID := gincontext.GetUserID(ctx)
	personID := gincontext.GetPersonID(ctx)
	tenantID := gincontext.GetTenantID(ctx)

	var userEntity *model.UserEntity
	var err error

	if userID != 0 {
		userEntity, err = userDao.GetByID(ctx.Request.Context(), userID)
	} else if personID != 0 && tenantID != 0 {
		userEntity, err = userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{PersonID: personID, TenantID: tenantID})
	} else {
		return nil, code.GetError(gconstant.UnauthorizedErr)
	}

	if err != nil {
		glog.Errorf(ctx, "[svcauth.Userinfo] dao query fail, err:%v, userID:%d, personID:%d, tenantID:%d", err, userID, personID, tenantID)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	if personID == 0 {
		personID = userEntity.PersonID
	}

	personInfo := objauth.PersonInfo{}
	if personID != 0 {
		personDao := newAuthPersonStore()
		personEntity, personErr := personDao.GetByID(ctx.Request.Context(), personID)
		if personErr == nil && personEntity != nil && personEntity.ID != 0 {
			personInfo = objauth.PersonInfo{
				PersonID: personEntity.ID,
				Name:     personEntity.Name,
				Avatar:   personEntity.Avatar,
			}
		} else {
			personInfo = objauth.PersonInfo{
				PersonID: personID,
			}
		}
	}

	return &dtoauth.UserinfoResp{
		PersonInfo: personInfo,
		UserInfo: objauth.TenantUserInfo{
			UserID:   userEntity.ID,
			TenantID: userEntity.TenantID,
			Name:     userEntity.Name,
			IsOwner:  userEntity.IsOwner,
		},
	}, nil
}

func (svc *authSvc) generateToken(ctx *gin.Context, userEntity *model.UserEntity) (*objauth.TokenInfo, error) {
	now := time.Now()
	accessTokenExp := now.Add(TokenExpireDuration)
	refreshTokenExp := now.Add(RefreshTokenExpireDuration)

	accessTokenClaims := &jwtauth.Claims[gobject.UserClaims]{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessTokenExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		CustomData: gobject.UserClaims{
			UserID:    userEntity.ID,
			TenantID:  userEntity.TenantID,
			PersonID:  userEntity.PersonID,
			TokenType: gobject.TokenTypeAuth,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessTokenString, err := accessToken.SignedString([]byte(svc.jwtSecret))
	if err != nil {
		return nil, err
	}

	refreshTokenClaims := jwt.MapClaims{
		"user_id":   userEntity.ID,
		"tenant_id": userEntity.TenantID,
		"exp":       refreshTokenExp.Unix(),
		"iat":       now.Unix(),
		"type":      "refresh",
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(svc.jwtSecret))
	if err != nil {
		return nil, err
	}

	refreshTokenDao := newAuthRefreshTokenStore()
	refreshTokenEntity := &model.RefreshTokenEntity{
		TenantID:      userEntity.TenantID,
		PersonID:      userEntity.PersonID,
		UserID:        userEntity.ID,
		OAuthClientID: 0,
		Token:         token.HashToken(refreshTokenString),
		ExpiredAt:     timePointer(refreshTokenExp),
		CreatedBy:     userEntity.ID,
	}
	if err := refreshTokenDao.Insert(ctx.Request.Context(), refreshTokenEntity); err != nil {
		glog.Errorf(ctx, "[svcauth.generateToken] save refreshToken fail, err:%v", err)
		return nil, err
	}

	return &objauth.TokenInfo{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(TokenExpireDuration.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (svc *authSvc) generatePersonToken(personEntity *model.PersonEntity) (*objauth.PersonTokenInfo, error) {
	now := time.Now()
	accessTokenExp := now.Add(TokenExpireDuration)
	claims := jwt.MapClaims{
		"person_id": personEntity.ID,
		"exp":       accessTokenExp.Unix(),
		"iat":       now.Unix(),
		"type":      "person",
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessTokenString, err := accessToken.SignedString([]byte(svc.jwtSecret))
	if err != nil {
		return nil, err
	}
	return &objauth.PersonTokenInfo{TokenInfo: objauth.TokenInfo{AccessToken: accessTokenString, ExpiresIn: int64(TokenExpireDuration.Seconds()), TokenType: "Bearer"}}, nil
}

func (svc *authSvc) resolvePersonLogin(ctx *gin.Context, personDao authPersonStore, userDao authUserStore, identifier string) (*model.PersonEntity, *model.UserEntity, []objauth.TenantOption, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, nil, nil, code.GetError(code.AuthIdentifierRequiredError)
	}

	personCond := &dao.PersonCond{}
	if strings.Contains(identifier, "@") {
		personCond.PrimaryEmail = identifier
	} else if len(identifier) >= 11 && strings.HasPrefix(identifier, "1") {
		personCond.PrimaryPhone = identifier
	} else {
		personCond.Username = identifier
	}

	personEntity, err := personDao.GetByCond(ctx.Request.Context(), personCond)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Login] person dao GetByCond fail, err:%v", err)
		return nil, nil, nil, code.GetError(code.UserGetDetailError)
	}
	if personEntity == nil || personEntity.ID == 0 {
		return nil, nil, nil, code.GetError(code.UserNotExistError)
	}

	userEntity, tenants, err := svc.listPersonTenants(ctx, personEntity.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	if userEntity == nil || userEntity.ID == 0 {
		userEntity, err = userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{PersonID: personEntity.ID})
		if err != nil {
			glog.Errorf(ctx, "[svcauth.Login] user dao GetByCond fail, err:%v", err)
			return nil, nil, nil, code.GetError(code.UserGetDetailError)
		}
		if userEntity == nil || userEntity.ID == 0 {
			return nil, nil, nil, code.GetError(code.UserNotExistError)
		}
	}
	if userEntity.IsSuspended == 1 {
		return nil, nil, nil, code.GetError(code.UserSuspendedError)
	}
	return personEntity, userEntity, tenants, nil
}

func (svc *authSvc) listPersonTenants(ctx *gin.Context, personID uint) (*model.UserEntity, []objauth.TenantOption, error) {
	userDao := newAuthUserStore()
	tenantDao := newAuthTenantStore()
	userEntity, err := userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{PersonID: personID})
	if err != nil {
		glog.Errorf(ctx, "[svcauth.listPersonTenants] user dao GetByCond fail, err:%v", err)
		return nil, nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, nil, code.GetError(code.UserNotExistError)
	}
	joinedUsers, err := userDao.GetListByCond(ctx.Request.Context(), &dao.UserCond{PersonID: personID})
	if err != nil {
		glog.Errorf(ctx, "[svcauth.listPersonTenants] user dao GetListByCond fail, err:%v", err)
		return nil, nil, code.GetError(code.UserGetDetailError)
	}
	options := make([]objauth.TenantOption, 0, len(joinedUsers))
	for _, joinedUser := range joinedUsers {
		if joinedUser.TenantID == 0 {
			continue
		}
		tenantEntity, getErr := tenantDao.GetByID(ctx.Request.Context(), joinedUser.TenantID)
		if getErr != nil {
			glog.Errorf(ctx, "[svcauth.listPersonTenants] tenant dao GetByID fail, err:%v, tenantID:%d", getErr, joinedUser.TenantID)
			return nil, nil, code.GetError(code.UserGetDetailError)
		}
		if tenantEntity == nil || tenantEntity.ID == 0 {
			continue
		}
		options = append(options, objauth.TenantOption{TenantID: tenantEntity.ID, Name: tenantEntity.Name, Tag: tenantEntity.Tag, UserID: joinedUser.ID, IsOwner: joinedUser.IsOwner})
	}
	return userEntity, options, nil
}

func (svc *authSvc) issueTenantToken(ctx *gin.Context, personID, tenantID uint) (*objauth.TokenInfo, error) {
	userDao := newAuthUserStore()
	tenantDao := newAuthTenantStore()
	userEntity, err := userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{PersonID: personID, TenantID: tenantID})
	if err != nil {
		glog.Errorf(ctx, "[svcauth.issueTenantToken] user dao GetByCond fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}
	tenantEntity, err := tenantDao.GetByID(ctx.Request.Context(), tenantID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.issueTenantToken] tenant dao GetByID fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if tenantEntity == nil || tenantEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}
	tokenInfo, err := svc.generateToken(ctx, userEntity)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.issueTenantToken] generateToken fail, err:%v", err)
		return nil, code.GetError(code.TokenGenerateError)
	}
	return tokenInfo, nil
}

func (svc *authSvc) personIDFromToken(personToken string) uint {
	if strings.TrimSpace(personToken) == "" {
		return 0
	}
	claims, err := jwt.Parse(personToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(svc.jwtSecret), nil
	})
	if err != nil {
		return 0
	}
	mapClaims, ok := claims.Claims.(jwt.MapClaims)
	if !ok {
		return 0
	}
	personID, ok := parsePositiveIntegerClaim(mapClaims, "person_id")
	if !ok {
		return 0
	}
	return personID
}

func (svc *authSvc) parseRefreshToken(refreshToken string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(svc.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if tokenType, ok := claims["type"].(string); ok && tokenType == "refresh" {
			return claims, nil
		}
		return nil, errors.New("invalid token type")
	}

	return nil, errors.New("invalid token")
}

func validatePasswordStrength(password string) error {
	if len(password) < PasswordMinLength {
		return errors.New("password too short")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("password must contain uppercase, lowercase and digit")
	}

	return nil
}

func parsePositiveIntegerClaim(claims jwt.MapClaims, key string) (uint, bool) {
	value, ok := claims[key].(float64)
	if !ok || value <= 0 || math.Trunc(value) != value {
		return 0, false
	}
	return uint(value), true
}

func defaultRecordLoginLog(ctx *gin.Context, tenantID, userID uint, success bool) {
	loginIP := gincontext.GetClientIP(ctx)
	userAgent := ctx.GetHeader("User-Agent")

	loginLogEntity := &model.UserLoginLogEntity{
		TenantID:  tenantID,
		UserID:    userID,
		LoginType: "password",
		LoginIP:   loginIP,
		UserAgent: userAgent,
		LoginTime: time.Now(),
		CreatedBy: 0,
	}

	if err := dao.NewUserLoginLogDao().Insert(ctx, loginLogEntity); err != nil {
		glog.Errorf(ctx, "[svcauth.recordLoginLog] insert login log fail, err:%v", err)
	}

	if success {
		userDao := newAuthUserStore()
		if err := userDao.UpdateMap(ctx.Request.Context(), userID, map[string]interface{}{
			"last_sign_in_at": time.Now(),
		}); err != nil {
			glog.Errorf(ctx, "[svcauth.recordLoginLog] update last_sign_in_at fail, err:%v", err)
		}
	}
}

func timePointer(t time.Time) *time.Time {
	return &t
}
