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
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
	"gorm.io/gorm"
)

const (
	PasswordMinLength          = 6
	TokenExpireDuration        = 24 * time.Hour
	RefreshTokenExpireDuration = 7 * 24 * time.Hour
)

type authUserStore interface {
	GetByID(ctx context.Context, id uint) (*model.UserEntity, error)
	GetByCond(ctx context.Context, cond genericdao.Cond) (*model.UserEntity, error)
	Insert(ctx context.Context, entity *model.UserEntity) error
	UpdateMap(ctx context.Context, id uint, updateMap map[string]interface{}) error
}

type authRefreshTokenStore interface {
	GetByCond(ctx context.Context, cond genericdao.Cond) (*model.RefreshTokenEntity, error)
	Insert(ctx context.Context, entity *model.RefreshTokenEntity) error
	Delete(ctx context.Context, id, userID uint) error
}

var newAuthUserStore = func() authUserStore {
	return dao.NewUserDao()
}

var newAuthRefreshTokenStore = func() authRefreshTokenStore {
	return dao.NewRefreshTokenDao()
}

type AuthSvc interface {
	Login(ctx *gin.Context, req *dtoauth.LoginReq) (*dtoauth.LoginResp, error)
	Register(ctx *gin.Context, req *dtoauth.RegisterReq) (*dtoauth.RegisterResp, error)
	RefreshToken(ctx *gin.Context, req *dtoauth.RefreshTokenReq) (*dtoauth.RefreshTokenResp, error)
	Logout(ctx *gin.Context, req *dtoauth.LogoutReq) error
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
	userDao := newAuthUserStore()
	var userEntity *model.UserEntity
	var err error

	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		return nil, code.GetError(code.AuthIdentifierRequiredError)
	}

	if strings.Contains(identifier, "@") {
		userEntity, err = userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{
			TenantID:     req.TenantID,
			PrimaryEmail: identifier,
		})
	} else if len(identifier) >= 11 && strings.HasPrefix(identifier, "1") {
		userEntity, err = userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{
			TenantID:     req.TenantID,
			PrimaryPhone: identifier,
		})
	} else {
		userEntity, err = userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{
			TenantID: req.TenantID,
			Username: identifier,
		})
	}

	if err != nil {
		glog.Errorf(ctx, "[svcauth.Login] dao GetByCond fail, err:%v", err)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	if userEntity.IsSuspended == 1 {
		return nil, code.GetError(code.UserSuspendedError)
	}

	if userEntity.PasswordEncrypted == "" || userEntity.PasswordMethod == "" {
		return nil, code.GetError(code.PasswordNotSetError)
	}

	if err := gcrypto.ComparePasswordHash(userEntity.PasswordEncrypted, req.Password); err != nil {
		svc.recordLoginLog(ctx, req.TenantID, userEntity.ID, false)
		glog.Errorf(ctx, "[svcauth.Login] password mismatch, userID:%d", userEntity.ID)
		return nil, code.GetError(code.PasswordMismatchError)
	}

	tokenInfo, err := svc.generateToken(ctx, userEntity)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Login] generateToken fail, err:%v", err)
		return nil, code.GetError(code.TokenGenerateError)
	}

	svc.recordLoginLog(ctx, req.TenantID, userEntity.ID, true)

	return &dtoauth.LoginResp{
		TokenInfo: *tokenInfo,
	}, nil
}

func (svc *authSvc) Register(ctx *gin.Context, req *dtoauth.RegisterReq) (*dtoauth.RegisterResp, error) {
	if err := validatePasswordStrength(req.Password); err != nil {
		return nil, code.GetError(code.PasswordValidationError)
	}

	if req.Username == "" && req.PrimaryEmail == "" && req.PrimaryPhone == "" {
		return nil, code.GetError(code.AuthIdentifierRequiredError)
	}

	userDao := newAuthUserStore()

	if req.Username != "" {
		existingUser, _ := userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{
			TenantID: req.TenantID,
			Username: req.Username,
		})
		if existingUser != nil && existingUser.ID != 0 {
			return nil, code.GetError(code.UsernameAlreadyExistsError)
		}
	}

	if req.PrimaryEmail != "" {
		existingUser, _ := userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{
			TenantID:     req.TenantID,
			PrimaryEmail: req.PrimaryEmail,
		})
		if existingUser != nil && existingUser.ID != 0 {
			return nil, code.GetError(code.EmailAlreadyExistsError)
		}
	}

	if req.PrimaryPhone != "" {
		existingUser, _ := userDao.GetByCond(ctx.Request.Context(), &dao.UserCond{
			TenantID:     req.TenantID,
			PrimaryPhone: req.PrimaryPhone,
		})
		if existingUser != nil && existingUser.ID != 0 {
			return nil, code.GetError(code.PhoneAlreadyExistsError)
		}
	}

	passwordHash, err := gcrypto.GeneratePasswordHash(req.Password)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Register] GeneratePasswordHash fail, err:%v", err)
		return nil, code.GetError(code.PasswordHashError)
	}

	insertEntity := &model.UserEntity{
		TenantID:          req.TenantID,
		Username:          req.Username,
		PrimaryEmail:      req.PrimaryEmail,
		PrimaryPhone:      req.PrimaryPhone,
		PasswordEncrypted: passwordHash,
		PasswordMethod:    "Argon2id",
		Name:              req.Name,
		CreatedBy:         0,
	}

	if err := userDao.Insert(ctx.Request.Context(), insertEntity); err != nil {
		glog.Errorf(ctx, "[svcauth.Register] dao Insert fail, err:%v", err)
		return nil, code.GetError(code.UserCreateError)
	}

	return &dtoauth.RegisterResp{
		UserID: insertEntity.ID,
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
	if storedToken.RevokedAt != nil && storedToken.RevokedAt.Valid {
		return nil, code.GetError(code.RefreshTokenInvalidError)
	}
	if storedToken.ExpiresAt == nil || !storedToken.ExpiresAt.Valid || !storedToken.ExpiresAt.Time.After(time.Now()) {
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

func (svc *authSvc) Userinfo(ctx *gin.Context, req *dtoauth.UserinfoReq) (*dtoauth.UserinfoResp, error) {
	userID := gincontext.GetUserID(ctx)
	if userID == 0 {
		return nil, code.GetError(gconstant.UnauthorizedErr)
	}

	userDao := newAuthUserStore()
	userEntity, err := userDao.GetByID(ctx.Request.Context(), userID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.Userinfo] dao GetByID fail, err:%v, userID:%d", err, userID)
		return nil, code.GetError(code.UserGetDetailError)
	}
	if userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	return &dtoauth.UserinfoResp{
		UserInfo: objauth.UserInfo{
			UserID:       userEntity.ID,
			TenantID:     userEntity.TenantID,
			Username:     userEntity.Username,
			PrimaryEmail: userEntity.PrimaryEmail,
			PrimaryPhone: userEntity.PrimaryPhone,
			Name:         userEntity.Name,
			Avatar:       userEntity.Avatar,
		},
	}, nil
}

func (svc *authSvc) generateToken(ctx *gin.Context, userEntity *model.UserEntity) (*objauth.TokenInfo, error) {
	now := time.Now()
	accessTokenExp := now.Add(TokenExpireDuration)
	refreshTokenExp := now.Add(RefreshTokenExpireDuration)

	accessTokenClaims := jwt.MapClaims{
		"user_id":   userEntity.ID,
		"tenant_id": userEntity.TenantID,
		"username":  userEntity.Username,
		"exp":       accessTokenExp.Unix(),
		"iat":       now.Unix(),
		"type":      "access",
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
		UserID:        userEntity.ID,
		ApplicationID: 0,
		Token:         token.HashToken(refreshTokenString),
		ExpiresAt: &gorm.DeletedAt{
			Time:  refreshTokenExp,
			Valid: true,
		},
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

func (svc *authSvc) recordLoginLog(ctx *gin.Context, tenantID, userID uint, success bool) {
	loginIP := gincontext.GetClientIP(ctx)
	userAgent := ctx.GetHeader("User-Agent")

	loginLogEntity := &model.UserLoginLogEntity{
		TenantID:  tenantID,
		UserID:   userID,
		LoginIP:  loginIP,
		UserAgent: userAgent,
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
