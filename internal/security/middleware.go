package security

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Middleware struct {
	validator *Validator
}

func NewMiddleware(validator *Validator) *Middleware {
	return &Middleware{validator: validator}
}

func (m *Middleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w)
		principal, err := m.validator.ValidateRequest(r)
		if err != nil {
			writeSecurityError(w, http.StatusUnauthorized, "authentication_required", err.Error())
			return
		}
		if err := requireTenant(principal); err != nil {
			writeSecurityError(w, http.StatusForbidden, "tenant_required", err.Error())
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
	})
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
}

func requireTenant(principal Principal) error {
	if strings.TrimSpace(principal.Tenant) == "" {
		return errTenantMissing{}
	}
	return nil
}

type errTenantMissing struct{}

func (errTenantMissing) Error() string { return "authenticated principal must include tenant" }

func writeSecurityError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}

func RequireRole(next http.Handler, roles ...Role) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok || !principal.HasAnyRole(roles...) {
			writeSecurityError(w, http.StatusForbidden, "insufficient_role", "principal does not have the required role")
			return
		}
		next.ServeHTTP(w, r)
	})
}
