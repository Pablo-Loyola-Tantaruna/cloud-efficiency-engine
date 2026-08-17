package security

import (
	"errors"
	"os"
	"strings"
)

var ErrAuthenticationDisabled = errors.New("authentication explicitly disabled")

func MiddlewareFromEnv() (*Middleware, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("FINOPS_AUTH_MODE")))
	if mode == "disabled" {
		return nil, ErrAuthenticationDisabled
	}
	secret := strings.TrimSpace(os.Getenv("FINOPS_JWT_SECRET"))
	validator, err := NewValidator(secret)
	if err != nil {
		return nil, err
	}
	return NewMiddleware(validator), nil
}
