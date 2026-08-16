package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"analytics-dashboard-be/internal/domain"
)

// AuthService authenticates users and issues/validates JWTs.
type AuthService struct {
	users  domain.UserRepository
	secret []byte
	ttl    time.Duration
}

func NewAuthService(users domain.UserRepository, secret string, ttlHours int) *AuthService {
	if ttlHours <= 0 {
		ttlHours = 24
	}
	return &AuthService{
		users:  users,
		secret: []byte(secret),
		ttl:    time.Duration(ttlHours) * time.Hour,
	}
}

// Claims is the JWT payload.
type Claims struct {
	Username string      `json:"username"`
	Role     domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// AuthenticatedUser is the identity extracted from a valid token.
type AuthenticatedUser struct {
	Username string         `json:"username"`
	Role     domain.Role    `json:"role"`
	Claims   []domain.Claim `json:"claims"`
}

// Login verifies credentials and returns a freshly-issued token. A new token is
// minted on every successful login, so logging in always renews the session.
func (s *AuthService) Login(ctx context.Context, username, password string) (string, AuthenticatedUser, error) {
	user, err := s.users.ByUsername(ctx, username)
	if err != nil {
		// Collapse "not found" and any lookup error into invalid-credentials so
		// we never reveal whether a username exists.
		return "", AuthenticatedUser{}, domain.ErrInvalidCredentials
	}
	// Constant-time bcrypt comparison; the plaintext password is never stored.
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", AuthenticatedUser{}, domain.ErrInvalidCredentials
	}

	now := time.Now()
	claims := Claims{
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", AuthenticatedUser{}, domain.ErrInternal
	}
	return token, s.identity(user.Username, user.Role), nil
}

// Users lists all accounts (admin feature). Never includes password hashes.
func (s *AuthService) Users(ctx context.Context) ([]domain.User, error) {
	return s.users.List(ctx)
}

// Validate parses and verifies a token, returning the authenticated identity.
func (s *AuthService) Validate(tokenStr string) (AuthenticatedUser, error) {
	var claims Claims
	token, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrTokenInvalid
		}
		return s.secret, nil
	})
	if err != nil {
		if isExpiry(err) {
			return AuthenticatedUser{}, domain.ErrTokenExpired
		}
		return AuthenticatedUser{}, domain.ErrTokenInvalid
	}
	if !token.Valid {
		return AuthenticatedUser{}, domain.ErrTokenInvalid
	}
	return s.identity(claims.Username, claims.Role), nil
}

func (s *AuthService) identity(username string, role domain.Role) AuthenticatedUser {
	return AuthenticatedUser{Username: username, Role: role, Claims: domain.ClaimsFor(role)}
}

// bcryptCost is deliberately above the library default (10). Cost 12 makes each
// hash ~4× slower to compute, which meaningfully raises the bar for offline
// brute-force if the DB is ever leaked, while staying fast enough for login.
const bcryptCost = 12

// HashPassword produces a salted bcrypt hash for storage. bcrypt embeds a random
// per-password salt, so identical passwords hash differently and rainbow tables
// are useless.
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(h), err
}

func isExpiry(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired)
}
