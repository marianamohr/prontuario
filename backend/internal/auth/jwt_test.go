package auth

import (
	"testing"
	"time"
)

func TestBuildAndParseJWT(t *testing.T) {
	secret := []byte("test-secret-min-32-chars!!")
	userID := "user-123"
	role := RoleProfessional
	clinicID := "clinic-456"
	tok, err := BuildJWT(secret, userID, role, &clinicID, false, nil, time.Hour)
	if err != nil {
		t.Fatalf("BuildJWT: %v", err)
	}
	claims, err := ParseJWT(secret, tok)
	if err != nil {
		t.Fatalf("ParseJWT: %v", err)
	}
	if claims.UserID != userID || claims.Role != role || claims.ClinicID == nil || *claims.ClinicID != clinicID {
		t.Fatalf("claims mismatch: %+v", claims)
	}
	if claims.IsImpersonated {
		t.Fatal("should not be impersonated")
	}
}

func TestJWTImpersonated(t *testing.T) {
	secret := []byte("test-secret-min-32-chars!!")
	sessionID := "sess-789"
	tok, err := BuildJWT(secret, "target-id", RoleProfessional, ptr("c1"), true, &sessionID, 15*time.Minute)
	if err != nil {
		t.Fatalf("BuildJWT: %v", err)
	}
	claims, err := ParseJWT(secret, tok)
	if err != nil {
		t.Fatalf("ParseJWT: %v", err)
	}
	if !claims.IsImpersonated || claims.ImpersonationSessionID == nil || *claims.ImpersonationSessionID != sessionID {
		t.Fatalf("impersonation claims: %+v", claims)
	}
}

func ptr(s string) *string { return &s }
