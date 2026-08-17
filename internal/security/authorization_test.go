package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizeFinOps_ShouldAllowApproverToApprove(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/action-plans/plan-1/approve", nil)
	request = request.WithContext(WithPrincipal(context.Background(), Principal{
		Subject: "user-1",
		Tenant:  "tenant-a",
		Roles:   map[Role]struct{}{RoleApprover: {}},
	}))
	called := false
	handler := AuthorizeFinOps(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !called {
		t.Fatalf("expected request to be authorized, status=%d called=%v", response.Code, called)
	}
}

func TestAuthorizeFinOps_ShouldRejectViewerFromApprove(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/action-plans/plan-1/approve", nil)
	request = request.WithContext(WithPrincipal(context.Background(), Principal{
		Subject: "user-1",
		Tenant:  "tenant-a",
		Roles:   map[Role]struct{}{RoleViewer: {}},
	}))
	handler := AuthorizeFinOps(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler should not be called") }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.Code)
	}
}

func TestAuthorizeFinOps_ShouldAllowAdminForExecution(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/action-plans/plan-1/execute", nil)
	request = request.WithContext(WithPrincipal(context.Background(), Principal{
		Subject: "admin-1",
		Tenant:  "tenant-a",
		Roles:   map[Role]struct{}{RoleAdmin: {}},
	}))
	handler := AuthorizeFinOps(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected admin execution authorization, got %d", response.Code)
	}
}
