package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	RoleProfessional = "PROFESSIONAL"
	RoleLegalGuardian = "LEGAL_GUARDIAN"
	RoleSuperAdmin   = "SUPER_ADMIN"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID                string    `json:"user_id"`
	Role                  string    `json:"role"`
	ClinicID              *string   `json:"clinic_id,omitempty"`
	IsImpersonated        bool      `json:"is_impersonated"`
	ImpersonationSessionID *string   `json:"impersonation_session_id,omitempty"`
}

func BuildJWT(secret []byte, userID, role string, clinicID *string, isImpersonated bool, impersonationSessionID *string, exp time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(exp)),
		},
		UserID:         userID,
		Role:           role,
		ClinicID:       clinicID,
		IsImpersonated: isImpersonated,
		ImpersonationSessionID: impersonationSessionID,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

func ParseJWT(secret []byte, tokenString string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if c, ok := t.Claims.(*Claims); ok && t.Valid {
		return c, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
