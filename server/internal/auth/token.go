package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Manager struct {
	secret []byte
}

func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

type AccessClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type ReconnectClaims struct {
	jwt.RegisteredClaims
}

func (m *Manager) GenerateAccessToken(playerID, username string, ttl time.Duration) (string, error) {
	claims := AccessClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   playerID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return m.sign(claims)
}

func (m *Manager) GenerateReconnectToken(playerID string, ttl time.Duration) (string, error) {
	claims := ReconnectClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   playerID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return m.sign(claims)
}

func (m *Manager) ParseReconnectToken(token string) (string, error) {
	parsed, err := jwt.ParseWithClaims(token, &ReconnectClaims{}, func(t *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(*ReconnectClaims)
	if !ok || !parsed.Valid {
		return "", jwt.ErrTokenInvalidClaims
	}
	return claims.Subject, nil
}

func (m *Manager) sign(claims jwt.Claims) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(m.secret)
}
