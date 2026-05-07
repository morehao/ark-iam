package svcauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	"github.com/morehao/golib/gcrypto"
	"github.com/morehao/golib/glog"
)

const (
	PasswordMinLength          = 6
	TokenExpireDuration        = 24 * time.Hour
	RefreshTokenExpireDuration = 7 * 24 * time.Hour
)

type AuthSvc interface {
	Login(ctx *gin.Context, req *dtoauth.LoginReq) (*dtoauth.LoginResp, error)
	Register(ctx *gin.Context, req *dtoauth.RegisterReq) (*dtoauth.RegisterResp, error)
	RefreshToken(ctx *gin.Context, req *dtoauth.RefreshTokenReq) (*dtoauth.RefreshTokenResp, error)
	Logout(ctx *gin.Context, req *dtoauth.LogoutReq) error
	Userinfo(ctx *gin.Context, req *dtoauth.UserinfoReq) (*dtoauth.UserinfoResp, error)
	GetSsoAuthorizationUrl(ctx *gin.Context, req *dtoauth.SsoAuthorizationUrlReq) (*dtoauth.SsoAuthorizationUrlResp, error)
	SsoCallback(ctx *gin.Context, req *dtoauth.SsoCallbackReq) (*dtoauth.SsoCallbackResp, error)
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
	userDao := dao.NewUserDao()
	var userEntity *model.UserEntity
	var err error

	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		return nil, code.GetError(code.AuthIdentifierRequiredError)
	}

	if strings.Contains(identifier, "@") {
		userEntity, err = userDao.GetByCond(ctx, &dao.UserCond{
			TenantID:     req.TenantID,
			PrimaryEmail: identifier,
		})
	} else if len(identifier) >= 11 && strings.HasPrefix(identifier, "1") {
		userEntity, err = userDao.GetByCond(ctx, &dao.UserCond{
			TenantID:     req.TenantID,
			PrimaryPhone: identifier,
		})
	} else {
		userEntity, err = userDao.GetByCond(ctx, &dao.UserCond{
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

	userDao := dao.NewUserDao()

	if req.Username != "" {
		existingUser, _ := userDao.GetByCond(ctx, &dao.UserCond{
			TenantID: req.TenantID,
			Username: req.Username,
		})
		if existingUser != nil && existingUser.ID != 0 {
			return nil, code.GetError(code.UsernameAlreadyExistsError)
		}
	}

	if req.PrimaryEmail != "" {
		existingUser, _ := userDao.GetByCond(ctx, &dao.UserCond{
			TenantID:     req.TenantID,
			PrimaryEmail: req.PrimaryEmail,
		})
		if existingUser != nil && existingUser.ID != 0 {
			return nil, code.GetError(code.EmailAlreadyExistsError)
		}
	}

	if req.PrimaryPhone != "" {
		existingUser, _ := userDao.GetByCond(ctx, &dao.UserCond{
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

	if err := userDao.Insert(ctx, insertEntity); err != nil {
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

	userID, ok := tokenClaims["user_id"].(float64)
	if !ok {
		return nil, code.GetError(gconstant.TokenInvalidErr)
	}

	refreshTokenDao := dao.NewRefreshTokenDao()
	storedToken, err := refreshTokenDao.GetByCond(ctx, &dao.RefreshTokenCond{
		UserID: uint(userID),
		Token:  token.HashToken(req.RefreshToken),
	})
	if err != nil || storedToken == nil {
		return nil, code.GetError(code.RefreshTokenInvalidError)
	}

	userDao := dao.NewUserDao()
	userEntity, err := userDao.GetByID(ctx, uint(userID))
	if err != nil || userEntity == nil || userEntity.ID == 0 {
		return nil, code.GetError(code.UserNotExistError)
	}

	tokenInfo, err := svc.generateToken(ctx, userEntity)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.RefreshToken] generateToken fail, err:%v", err)
		return nil, code.GetError(code.TokenGenerateError)
	}

	if err := refreshTokenDao.Delete(ctx, storedToken.ID, gincontext.GetUserID(ctx)); err != nil {
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

	userDao := dao.NewUserDao()
	userEntity, err := userDao.GetByID(ctx, userID)
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

func (svc *authSvc) GetSsoAuthorizationUrl(ctx *gin.Context, req *dtoauth.SsoAuthorizationUrlReq) (*dtoauth.SsoAuthorizationUrlResp, error) {
	connectorID := req.ConnectorID
	if connectorID == "" {
		return nil, code.GetError(code.SsoConnectorNotExistError)
	}

	var ssoConnectorID uint
	if _, err := fmt.Sscan(connectorID, &ssoConnectorID); err != nil {
		return nil, code.GetError(code.SsoConnectorNotExistError)
	}

	ssoConnectorEntity, err := dao.NewSsoConnectorDao().GetByID(ctx, ssoConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.GetSsoAuthorizationUrl] dao GetByID fail, err:%v", err)
		return nil, code.GetError(code.SsoConnectorGetDetailError)
	}
	if ssoConnectorEntity == nil || ssoConnectorEntity.ID == 0 {
		return nil, code.GetError(code.SsoConnectorNotExistError)
	}

	authorizationUrl, err := svc.buildAuthorizationUrl(ssoConnectorEntity)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.GetSsoAuthorizationUrl] buildAuthorizationUrl fail, err:%v", err)
		return nil, code.GetError(code.SsoConnectorGetDetailError)
	}

	return &dtoauth.SsoAuthorizationUrlResp{
		AuthorizationUrl: authorizationUrl,
	}, nil
}

func (svc *authSvc) SsoCallback(ctx *gin.Context, req *dtoauth.SsoCallbackReq) (*dtoauth.SsoCallbackResp, error) {
	connectorID := req.ConnectorID
	if connectorID == "" {
		return nil, code.GetError(code.SsoConnectorNotExistError)
	}

	var ssoConnectorID uint
	if _, err := fmt.Sscan(connectorID, &ssoConnectorID); err != nil {
		return nil, code.GetError(code.SsoConnectorNotExistError)
	}

	ssoConnectorEntity, err := dao.NewSsoConnectorDao().GetByID(ctx, ssoConnectorID)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.SsoCallback] dao GetByID fail, err:%v", err)
		return nil, code.GetError(code.SsoConnectorGetDetailError)
	}
	if ssoConnectorEntity == nil || ssoConnectorEntity.ID == 0 {
		return nil, code.GetError(code.SsoConnectorNotExistError)
	}

	userInfo, err := svc.exchangeCodeForUserInfo(ctx, ssoConnectorEntity, req.Code, req.State)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.SsoCallback] exchangeCodeForUserInfo fail, err:%v", err)
		return nil, code.GetError(code.SsoAuthFailedError)
	}

	userEntity, err := svc.findOrCreateSsoUser(ctx, ssoConnectorEntity, userInfo)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.SsoCallback] findOrCreateSsoUser fail, err:%v", err)
		return nil, code.GetError(code.UserCreateError)
	}

	tokenInfo, err := svc.generateToken(ctx, userEntity)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.SsoCallback] generateToken fail, err:%v", err)
		return nil, code.GetError(code.TokenGenerateError)
	}

	svc.recordLoginLog(ctx, userEntity.TenantID, userEntity.ID, true)

	return &dtoauth.SsoCallbackResp{
		TokenInfo: *tokenInfo,
	}, nil
}

func (svc *authSvc) buildAuthorizationUrl(ssoConnector *model.SsoConnectorEntity) (string, error) {
	var config SsoConnectorConfig
	if err := json.Unmarshal(ssoConnector.Config, &config); err != nil {
		return "", err
	}

	state := generateState()

	params := url.Values{}
	params.Set("client_id", config.ClientID)
	params.Set("redirect_uri", config.RedirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(config.Scopes, " "))
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", config.AuthURL, params.Encode()), nil
}

type SsoConnectorConfig struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	AuthURL      string   `json:"authUrl"`
	TokenURL     string   `json:"tokenUrl"`
	UserInfoURL  string   `json:"userInfoUrl"`
	RedirectURI  string   `json:"redirectUri"`
	Scopes       []string `json:"scopes"`
}

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (svc *authSvc) exchangeCodeForUserInfo(ctx *gin.Context, ssoConnector *model.SsoConnectorEntity, code, state string) (*ssoUserInfo, error) {
	var config SsoConnectorConfig
	if err := json.Unmarshal(ssoConnector.Config, &config); err != nil {
		return nil, err
	}

	tokenResp, err := svc.exchangeCode(config.TokenURL, config.ClientID, config.ClientSecret, code, config.RedirectURI)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.exchangeCodeForUserInfo] exchange code fail, err:%v", err)
		return nil, err
	}

	userInfo, err := svc.fetchUserInfo(config.UserInfoURL, tokenResp.AccessToken)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.exchangeCodeForUserInfo] fetch user info fail, err:%v", err)
		return nil, err
	}

	return &ssoUserInfo{
		Issuer:    ssoConnector.ProviderName,
		Subject:   userInfo.Subject,
		Email:     userInfo.Email,
		Name:      userInfo.Name,
		AvatarUrl: userInfo.Picture,
	}, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type userInfoResponse struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (svc *authSvc) exchangeCode(tokenURL, clientID, clientSecret, code, redirectURI string) (*tokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

func (svc *authSvc) fetchUserInfo(userInfoURL, accessToken string) (*userInfoResponse, error) {
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo userInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

type ssoUserInfo struct {
	Issuer    string
	Subject   string
	Email     string
	Name      string
	AvatarUrl string
}

func (svc *authSvc) findOrCreateSsoUser(ctx *gin.Context, ssoConnector *model.SsoConnectorEntity, userInfo *ssoUserInfo) (*model.UserEntity, error) {
	userIdentityDao := dao.NewUserIdentityDao()
	existingIdentity, err := userIdentityDao.GetByIssuerAndIdentityID(ctx, ssoConnector.TenantID, ssoConnector.ProviderName, userInfo.Subject)
	if err != nil {
		glog.Errorf(ctx, "[svcauth.findOrCreateSsoUser] GetByIssuerAndIdentityID fail, err:%v", err)
	}

	if existingIdentity != nil && existingIdentity.UserID != 0 {
		userDao := dao.NewUserDao()
		return userDao.GetByID(ctx, existingIdentity.UserID)
	}

	username := userInfo.Email
	if username == "" {
		username = userInfo.Subject
	}

	userDao := dao.NewUserDao()
	newUser := &model.UserEntity{
		TenantID:  ssoConnector.TenantID,
		Username:  username,
		Name:      userInfo.Name,
		Avatar:    userInfo.AvatarUrl,
		CreatedBy: 0,
	}

	if err := userDao.Insert(ctx, newUser); err != nil {
		return nil, err
	}

	identityEntity := &model.UserIdentityEntity{
		TenantID:   ssoConnector.TenantID,
		UserID:     newUser.ID,
		IdentityID: userInfo.Subject,
		Issuer:     ssoConnector.ProviderName,
		CreatedBy:  0,
	}

	if err := userIdentityDao.Insert(ctx, identityEntity); err != nil {
		glog.Errorf(ctx, "[svcauth.findOrCreateSsoUser] Insert identity fail, err:%v", err)
	}

	return newUser, nil
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

	refreshTokenDao := dao.NewRefreshTokenDao()
	refreshTokenEntity := &model.RefreshTokenEntity{
		TenantID:      userEntity.TenantID,
		UserID:        userEntity.ID,
		ApplicationID: 0,
		Token:         token.HashToken(refreshTokenString),
		CreatedBy:     userEntity.ID,
	}
	if err := refreshTokenDao.Insert(ctx, refreshTokenEntity); err != nil {
		glog.Errorf(ctx, "[svcauth.generateToken] save refreshToken fail, err:%v", err)
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
		userDao := dao.NewUserDao()
		if err := userDao.UpdateMap(ctx, userID, map[string]interface{}{
			"last_sign_in_at": time.Now(),
		}); err != nil {
			glog.Errorf(ctx, "[svcauth.recordLoginLog] update last_sign_in_at fail, err:%v", err)
		}
	}
}