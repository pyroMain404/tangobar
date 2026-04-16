package templates

import "context"

// ctxKey is unexported so only this package (and callers via the helpers
// below) can touch the role value on a context.
type ctxKey int

const roleKey ctxKey = 1

// WithRole returns a new context carrying the given user role.
// Middleware on the handlers side calls this after a successful auth check.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

// RoleFromContext returns the role stored on ctx, or "" if none.
func RoleFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(roleKey).(string); ok {
		return v
	}
	return ""
}

// IsAdminCtx reports whether the context carries an admin role.
// Templates call this directly (same package) to conditionally render UI.
func IsAdminCtx(ctx context.Context) bool {
	return RoleFromContext(ctx) == "admin"
}
