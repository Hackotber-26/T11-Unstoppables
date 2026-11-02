package auth

import (
	"context"
	"crypto/rand"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"t11-unstoppables/internal/models"
	"t11-unstoppables/internal/storage"

	"golang.org/x/crypto/bcrypt"
)

const sessionLifetime = time.Hour

var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrInvalidRegistration = errors.New("invalid registration details")
var ErrInvalidInvitation = errors.New("invitation is invalid or expired")

type UserRepository interface {
	CreateAccount(context.Context, string, string, string, string, string, string) (models.User, error)
	CreateInvitation(context.Context, string, string, string, time.Time) error
	UserByEmail(context.Context, string) (models.User, error)
}

type Session struct {
	AccessToken string      `json:"accessToken"`
	ExpiresAt   time.Time   `json:"expiresAt"`
	User        models.User `json:"user"`
}

type Claims struct {
	Subject string `json:"sub"`
	Role    string `json:"role"`
	Issuer  string `json:"iss"`
	Issued  int64  `json:"iat"`
	Expires int64  `json:"exp"`
}

type Service struct {
	users  UserRepository
	secret []byte
	now    func() time.Time
}

func NewService(users UserRepository, secret string) (*Service, error) {
	if users == nil {
		return nil, errors.New("user repository is required")
	}
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 characters")
	}
	return &Service{users: users, secret: []byte(secret), now: time.Now}, nil
}

func (s *Service) RegisterRider(ctx context.Context, name, email, phone, password string) (Session, error) {
	return s.Register(ctx, name, email, phone, password, "rider", "")
}

func (s *Service) Register(ctx context.Context, name, email, phone, password, role, invitationCode string) (Session, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email || name == "" || len(name) > 100 || len(password) < 12 || len(password) > 72 || len(phone) > 40 {
		return Session{}, ErrInvalidRegistration
	}
	if role != "rider" && role != "driver" && role != "admin" {
		return Session{}, ErrInvalidRegistration
	}
	invitationHash := ""
	if role == "rider" {
		if invitationCode != "" {
			return Session{}, ErrInvalidRegistration
		}
	} else {
		if len(invitationCode) < 32 || len(invitationCode) > 128 {
			return Session{}, ErrInvalidInvitation
		}
		hash := sha256.Sum256([]byte(strings.TrimSpace(invitationCode)))
		invitationHash = hex.EncodeToString(hash[:])
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Session{}, fmt.Errorf("hash password: %w", err)
	}
	user, err := s.users.CreateAccount(ctx, name, email, strings.TrimSpace(phone), string(passwordHash), role, invitationHash)
	if errors.Is(err, storage.ErrInvalidInvitation) {
		return Session{}, ErrInvalidInvitation
	}
	if errors.Is(err, storage.ErrConflict) {
		return Session{}, storage.ErrConflict
	}
	if err != nil {
		return Session{}, err
	}
	if user.Role != role {
		return Session{}, errors.New("created account role does not match the requested role")
	}
	return s.newSession(user)
}

func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	return s.LoginAs(ctx, email, password, "")
}

func (s *Service) LoginAs(ctx context.Context, email, password, expectedRole string) (Session, error) {
	if len(password) == 0 || len(password) > 72 {
		return Session{}, ErrInvalidCredentials
	}
	user, err := s.users.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || expectedRole != "" && user.Role != expectedRole || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return Session{}, ErrInvalidCredentials
	}
	return s.newSession(user)
}

func (s *Service) CreateInvitation(ctx context.Context, role, createdBy string) (string, time.Time, error) {
	if role != "driver" && role != "admin" {
		return "", time.Time{}, ErrInvalidRegistration
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", time.Time{}, fmt.Errorf("generate invitation: %w", err)
	}
	code := base64.RawURLEncoding.EncodeToString(random)
	hash := sha256.Sum256([]byte(code))
	expiresAt := s.now().UTC().Add(7 * 24 * time.Hour)
	if err := s.users.CreateInvitation(ctx, role, createdBy, hex.EncodeToString(hash[:]), expiresAt); err != nil {
		return "", time.Time{}, fmt.Errorf("store invitation: %w", err)
	}
	return code, expiresAt, nil
}

func (s *Service) ParseToken(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("invalid token")
	}
	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "HS256" || header.Type != "JWT" {
		return Claims{}, errors.New("invalid token header")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, errors.New("invalid token signature")
	}
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return Claims{}, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, errors.New("invalid token payload")
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, errors.New("invalid token payload")
	}
	now := s.now().Unix()
	validRole := claims.Role == "rider" || claims.Role == "driver" || claims.Role == "admin"
	if claims.Subject == "" || !validRole || claims.Issuer != "trainsync" || claims.Issued > now+30 || claims.Expires <= now || claims.Expires <= claims.Issued || claims.Expires-claims.Issued > int64(sessionLifetime.Seconds()) {
		return Claims{}, errors.New("expired or invalid token")
	}
	return claims, nil
}

func (s *Service) newSession(user models.User) (Session, error) {
	now := s.now().UTC()
	claims := Claims{
		Subject: user.ID,
		Role:    user.Role,
		Issuer:  "trainsync",
		Issued:  now.Unix(),
		Expires: now.Add(sessionLifetime).Unix(),
	}
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		return Session{}, fmt.Errorf("encode session claims: %w", err)
	}
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(unsigned))
	token := unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	user.PasswordHash = ""
	return Session{AccessToken: token, ExpiresAt: now.Add(sessionLifetime), User: user}, nil
}