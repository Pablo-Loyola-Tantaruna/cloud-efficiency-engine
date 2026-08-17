package security

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Role string

const (
	RoleViewer   Role = "viewer"
	RoleOperator Role = "operator"
	RoleApprover Role = "approver"
	RoleAdmin    Role = "admin"
)

type Principal struct {
	Subject string
	Tenant  string
	Roles   map[Role]struct{}
}

func (p Principal) HasRole(role Role) bool {
	if _, ok := p.Roles[RoleAdmin]; ok {
		return true
	}
	_, ok := p.Roles[role]
	return ok
}

func (p Principal) HasAnyRole(roles ...Role) bool {
	for _, role := range roles {
		if p.HasRole(role) {
			return true
		}
	}
	return false
}

type Claims struct {
	Roles  []string `json:"roles,omitempty"`
	Tenant string   `json:"tenant,omitempty"`
	jwt.RegisteredClaims
}

type Validator struct {
	secret []byte
	now    func() time.Time
}

func NewValidator(secret string) (*Validator, error) {
	secret = strings.TrimSpace(secret)
	if len(secret) < 32 {
		return nil, errors.New("FINOPS_JWT_SECRET must contain at least 32 characters")
	}
	return &Validator{secret: []byte(secret), now: time.Now}, nil
}

func (v *Validator) ValidateRequest(r *http.Request) (Principal, error) {
	if v == nil {
		return Principal{}, errors.New("JWT validator is not configured")
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return Principal{}, errors.New("authorization header is required")
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return Principal{}, errors.New("authorization header must use Bearer token")
	}
	return v.ValidateToken(parts[1])
}

func (v *Validator) ValidateToken(raw string) (Principal, error) {
	if strings.TrimSpace(raw) == "" {
		return Principal{}, errors.New("token is empty")
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected JWT signing method %q", token.Method.Alg())
		}
		return v.secret, nil
	}, jwt.WithTimeFunc(v.now), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		if err == nil {
			err = errors.New("invalid JWT")
		}
		return Principal{}, err
	}
	if claims.Subject == "" {
		return Principal{}, errors.New("JWT subject is required")
	}
	roles := make(map[Role]struct{}, len(claims.Roles))
	for _, rawRole := range claims.Roles {
		switch Role(strings.ToLower(strings.TrimSpace(rawRole))) {
		case RoleViewer, RoleOperator, RoleApprover, RoleAdmin:
			roles[Role(strings.ToLower(strings.TrimSpace(rawRole)))] = struct{}{}
		default:
			return Principal{}, fmt.Errorf("unsupported JWT role %q", rawRole)
		}
	}
	if len(roles) == 0 {
		return Principal{}, errors.New("JWT must contain at least one supported role")
	}
	return Principal{Subject: claims.Subject, Tenant: strings.TrimSpace(claims.Tenant), Roles: roles}, nil
}

type contextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok
}
