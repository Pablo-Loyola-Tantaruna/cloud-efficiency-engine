package security

import (
	"net/http"
	"strings"
)

func AuthorizeFinOps(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeSecurityError(w, http.StatusUnauthorized, "authentication_required", "authenticated principal is missing")
			return
		}
		roles := rolesForRequest(r.Method, r.URL.Path)
		if !principal.HasAnyRole(roles...) {
			writeSecurityError(w, http.StatusForbidden, "insufficient_role", "principal does not have the required role")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rolesForRequest(method, path string) []Role {
	switch {
	case path == "/api/v1/action-plans" && method == http.MethodPost:
		return []Role{RoleOperator}
	case strings.HasPrefix(path, "/api/v1/action-plans/") && method == http.MethodGet:
		return []Role{RoleViewer, RoleOperator, RoleApprover}
	case strings.HasSuffix(path, "/approve") && method == http.MethodPost:
		return []Role{RoleApprover}
	case strings.HasSuffix(path, "/execute") && method == http.MethodPost:
		return []Role{RoleOperator}
	case strings.HasSuffix(path, "/submit") && method == http.MethodPost:
		return []Role{RoleOperator}
	case strings.HasSuffix(path, "/dry-run") && method == http.MethodPost:
		return []Role{RoleViewer, RoleOperator, RoleApprover}
	case strings.HasPrefix(path, "/api/v1/executions/") && method == http.MethodGet:
		return []Role{RoleViewer, RoleOperator, RoleApprover}
	case strings.HasPrefix(path, "/api/v1/recovery/") && method == http.MethodGet:
		return []Role{RoleViewer, RoleOperator, RoleApprover}
	case path == "/api/v1/analyze" && method == http.MethodPost:
		return []Role{RoleViewer, RoleOperator, RoleApprover}
	default:
		return []Role{RoleAdmin}
	}
}
