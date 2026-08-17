package security

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "01234567890123456789012345678901"

func signedToken(t *testing.T, method jwt.SigningMethod, claims Claims) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func TestValidator_ShouldRejectMissingAuthorization(t *testing.T) {
	validator, err := NewValidator(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/api/v1/action-plans", nil)
	if _, err := validator.ValidateRequest(request); err == nil {
		t.Fatal("expected missing authorization error")
	}
}

func TestValidator_ShouldValidatePrincipalClaims(t *testing.T) {
	validator, err := NewValidator(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	raw := signedToken(t, jwt.SigningMethodHS256, Claims{
		Roles:  []string{"approver"},
		Tenant: "tenant-a",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	request := httptest.NewRequest("POST", "/api/v1/action-plans/plan-1/approve", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	principal, err := validator.ValidateRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Subject != "user-123" || principal.Tenant != "tenant-a" || !principal.HasRole(RoleApprover) {
		t.Fatalf("unexpected principal: %+v", principal)
	}
}

func TestValidator_ShouldRejectUnexpectedSigningMethod(t *testing.T) {
	validator, err := NewValidator(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	raw := signedToken(t, jwt.SigningMethodHS384, Claims{
		Roles:  []string{"viewer"},
		Tenant: "tenant-a",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	if _, err := validator.ValidateToken(raw); err == nil {
		t.Fatal("expected signing method rejection")
	}
}

func TestValidator_ShouldRejectExpiredToken(t *testing.T) {
	validator, err := NewValidator(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	raw := signedToken(t, jwt.SigningMethodHS256, Claims{
		Roles:  []string{"viewer"},
		Tenant: "tenant-a",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	if _, err := validator.ValidateToken(raw); err == nil {
		t.Fatal("expected expired token rejection")
	}
}
