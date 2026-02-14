package auth

import "context"

type contextKey string

const claimsKey contextKey = "claims"

func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

func ClaimsFrom(ctx context.Context) *Claims {
	if c, _ := ctx.Value(claimsKey).(*Claims); c != nil {
		return c
	}
	return nil
}

func ClinicIDFrom(ctx context.Context) *string {
	c := ClaimsFrom(ctx)
	if c == nil {
		return nil
	}
	return c.ClinicID
}

func UserIDFrom(ctx context.Context) string {
	c := ClaimsFrom(ctx)
	if c == nil {
		return ""
	}
	return c.UserID
}

func RoleFrom(ctx context.Context) string {
	c := ClaimsFrom(ctx)
	if c == nil {
		return ""
	}
	return c.Role
}

func IsSuperAdmin(ctx context.Context) bool {
	return RoleFrom(ctx) == RoleSuperAdmin
}

func IsImpersonated(ctx context.Context) bool {
	c := ClaimsFrom(ctx)
	return c != nil && c.IsImpersonated
}
