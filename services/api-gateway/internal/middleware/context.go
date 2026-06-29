package middleware

import "context"

// ctxKey is an unexported type for context keys in this package.
// Using a typed int prevents collisions with keys from other packages.
type ctxKey int

const (
	keyTenantID ctxKey = iota
	keyStoreID
	keyUserID
	keyRole
	keyJTI
	keyRawToken
)

// Exported accessors — used by handlers.

// TenantIDFromCtx returns the authenticated tenant ID from the context.
func TenantIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(keyTenantID).(string)
	return v
}

// StoreIDFromCtx returns the resolved store ID from the context.
func StoreIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(keyStoreID).(string)
	return v
}

// UserIDFromCtx returns the authenticated user ID (JWT sub) from the context.
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(keyUserID).(string)
	return v
}

// RoleFromCtx returns the authenticated user role from the context.
func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(keyRole).(string)
	return v
}

// JTIFromCtx returns the JWT ID (jti claim) from the context.
func JTIFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(keyJTI).(string)
	return v
}

// RawTokenFromCtx returns the raw JWT string from the context.
func RawTokenFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(keyRawToken).(string)
	return v
}

// Unexported setters — used only within this package.

func withTenantID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyTenantID, id)
}

func withStoreID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyStoreID, id)
}

func withUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyUserID, id)
}

func withRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, keyRole, role)
}

func withJTI(ctx context.Context, jti string) context.Context {
	return context.WithValue(ctx, keyJTI, jti)
}

func withRawToken(ctx context.Context, tok string) context.Context {
	return context.WithValue(ctx, keyRawToken, tok)
}
