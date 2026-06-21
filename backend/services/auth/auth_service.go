package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	googleclient "github.com/subbu/family_tree/client/google"
	postgresclient "github.com/subbu/family_tree/client/postgres"
	"github.com/subbu/family_tree/models"
	"github.com/subbu/family_tree/services/user"
)

type service struct {
	google     googleclient.Client
	users      user.Service
	jwtSecret  []byte
	siteAdmin  string
}

func NewService(
	google googleclient.Client,
	db postgresclient.Client,
	users user.Service,
	jwtSecret string,
	siteAdminEmail string,
) Service {
	_ = db
	return &service{
		google:    google,
		users:     users,
		jwtSecret: []byte(jwtSecret),
		siteAdmin: siteAdminEmail,
	}
}

func (s *service) LoginURL(state string) string {
	return s.google.AuthURL(state)
}

func (s *service) HandleCallback(ctx context.Context, code string) (string, models.User, error) {
	accessToken, err := s.google.Exchange(ctx, code)
	if err != nil {
		return "", models.User{}, err
	}

	info, err := s.google.FetchUserInfo(ctx, accessToken)
	if err != nil {
		return "", models.User{}, err
	}

	siteRole := models.SiteRoleUser
	if s.siteAdmin != "" && strings.EqualFold(info.Email, s.siteAdmin) {
		siteRole = models.SiteRoleAdmin
	}

	u, err := s.users.UpsertGoogleUser(ctx, user.UpsertGoogleUserInput{
		GoogleSub: info.Sub,
		Email:     info.Email,
		Name:      info.Name,
		AvatarURL: info.Picture,
		SiteRole:  siteRole,
	})
	if err != nil {
		return "", models.User{}, err
	}

	token, err := s.issueToken(u)
	if err != nil {
		return "", models.User{}, err
	}

	return token, u, nil
}

func (s *service) ValidateToken(ctx context.Context, token string) (models.User, error) {
	_ = ctx
	userID, err := s.parseToken(token)
	if err != nil {
		return models.User{}, err
	}
	return s.users.GetByID(context.Background(), userID)
}

func (s *service) UserIDFromToken(ctx context.Context, token string) (uuid.UUID, error) {
	_ = ctx
	return s.parseToken(token)
}

type tokenClaims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
}

func (s *service) issueToken(user models.User) (string, error) {
	claims := tokenClaims{
		Sub: user.ID.String(),
		Exp: time.Now().Add(24 * time.Hour).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.sign(encodedPayload)
	return encodedPayload + "." + signature, nil
}

func (s *service) parseToken(token string) (uuid.UUID, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return uuid.Nil, errors.New("invalid token format")
	}

	if !hmac.Equal([]byte(s.sign(parts[0])), []byte(parts[1])) {
		return uuid.Nil, errors.New("invalid token signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return uuid.Nil, err
	}

	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return uuid.Nil, err
	}
	if time.Now().Unix() > claims.Exp {
		return uuid.Nil, errors.New("token expired")
	}

	return uuid.Parse(claims.Sub)
}

func (s *service) sign(payload string) string {
	mac := hmac.New(sha256.New, s.jwtSecret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}