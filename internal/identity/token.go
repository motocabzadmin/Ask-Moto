package identity

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenService handles JWT token operations
type TokenService struct {
	secret []byte
}

// NewTokenService creates a new token service with the given secret
func NewTokenService(secret string) *TokenService {
	return &TokenService{
		secret: []byte(secret),
	}
}

// GenerateToken creates a new JWT token for the given user
func (s *TokenService) GenerateToken(userID string, role string, ttl time.Duration) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ParseToken parses and validates a JWT token string
func (s *TokenService) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ValidateToken validates a token and checks expiration
func (s *TokenService) ValidateToken(tokenString string) (*Claims, error) {
	claims, err := s.ParseToken(tokenString)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(now) {
		return nil, errors.New("token expired")
	}

	// Validate token was issued within the last 30 days (matching motocabz logic)
	if claims.IssuedAt != nil {
		issuedAt := claims.IssuedAt.Time
		timeSinceIssued := now.Sub(issuedAt)
		// Allow 30 seconds buffer for clock skew
		if timeSinceIssued > 30*24*time.Hour+30*time.Second {
			return nil, errors.New("token expired")
		}
	}

	return claims, nil
}

// DefaultTokenTTL is the default token time-to-live (30 days, matching motocabz)
const DefaultTokenTTL = 30 * 24 * time.Hour

// RefreshTokenTTL is the refresh token time-to-live (30 days, matching motocabz)
const RefreshTokenTTL = 30 * 24 * time.Hour
