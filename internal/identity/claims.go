package identity

import (
	"github.com/golang-jwt/jwt/v5"
)

// User roles matching motocabz identity service
const (
	RoleAdmin  = "ADMIN"
	RoleDriver = "DRIVER"
	RoleRider  = "RIDER"
)

// Claims represents JWT claims compatible with motocabz tokens
type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// IsDriver returns true if the user has driver role
func (c *Claims) IsDriver() bool {
	return c.Role == RoleDriver
}

// IsRider returns true if the user has rider role
func (c *Claims) IsRider() bool {
	return c.Role == RoleRider
}

// IsAdmin returns true if the user has admin role
func (c *Claims) IsAdmin() bool {
	return c.Role == RoleAdmin
}
