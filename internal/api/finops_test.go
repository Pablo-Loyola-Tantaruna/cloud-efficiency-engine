package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud-efficiency-engine/internal/analysis/actions"
	"cloud-efficiency-engine/internal/domain"
)

type fakeActionPlanRepository struct{ plans map[string]domain.ActionPlan }

func newFakeActionPlanRepository() *fakeActionPlanRepository {
	return &fakeActionPlanRepository{plans: make(map[string]domain.ActionPlan)}
}
func (r *fakeActionPlanRepository) GetActionPlanByID(id string) (domain.ActionPlan, bool, error) {
	p, ok := r.plans[id]
	return p, ok, nil
}
func (r *fakeActionPlanRepository) CreateActionPlanIfAbsent(plan domain.ActionPlan) (bool, error) {
	if _, ok := r.plans[plan.ID]; ok {
		return false, nil
	}
	r.plans[plan.ID] = plan
	return true, nil
}
func (r *fakeActionPlanRepository) UpdateActionPlan(plan domain.ActionPlan) error {
	r.plans[plan.ID] = plan
	return nil
}

type fakeApprovalRepository struct {
	approvals map[string]domain.ActionApproval
}

func newFakeApprovalRepository() *fakeApprovalRepository {
	return &fakeApprovalRepository{approvals: make(map[string]domain.ActionApproval)}
}
func (r *fakeApprovalRepository) GetApprovalByPlanID(planID string) (domain.ActionApproval, bool, error) {
	a, ok := r.approvals[planID]
	return a, ok, nil
}
func (r *fakeApprovalRepository) SaveApproval(approval domain.ActionApproval) error {
	r.approvals[approval.PlanID] = approval
	return nil
}

func newTestAction() domain.Action {
	return domain.Action{ID: "action-1", Type: domain.ActionReduceNodeGroup, Provider: domain.CloudProviderAWS, Cluster: "cluster-1", NodeGroup: "workers", CurrentValue: 8, DesiredValue: 6, MonthlySavingsUSD: 100, AnnualizedSavingsUSD: 1200, Risk: domain.ActionRiskMedium, RequiresApproval: true}
}

func newTestPlan(t *testing.T) domain.ActionPlan {
	t.Helper()
	plan, err := actions.BuildActionPlan(domain.CloudProviderAWS, "cluster-1", []domain.Action{newTestAction()})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func newTestFinOpsHandler() (*FinOpsHandler, *fakeActionPlanRepository, *fakeApprovalRepository) {
	plans := newFakeActionPlanRepository()
	approvals := newFakeApprovalRepository()
	return NewFinOpsHandler(plans, approvals, nil, nil, nil, nil, nil, nil), plans, approvals
}

func TestFinOpsHandler_ShouldCreatePlan(t *testing.T) {
	t.Parallel()
	handler, plans, _ := newTestFinOpsHandler()
	mux := http.NewServeMux()
	handler.Register(mux)
	payload := map[string]any{"provider": "aws", "cluster": "cluster-1", "actions": []domain.Action{newTestAction()}}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/action-plans", bytes.NewReader(body)).WithContext(context.Background())
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(plans.plans) != 1 {
		t.Fatalf("expected one persisted plan, got %d", len(plans.plans))
	}
}

func TestFinOpsHandler_ShouldMovePlanThroughApprovalLifecycle(t *testing.T) {
	t.Parallel()
	handler, plans, approvals := newTestFinOpsHandler()
	mux := http.NewServeMux()
	handler.Register(mux)
	plan := newTestPlan(t)
	plans.plans[plan.ID] = plan

	submitRecorder := httptest.NewRecorder()
	mux.ServeHTTP(submitRecorder, httptest.NewRequest(http.MethodPost, "/api/v1/action-plans/"+plan.ID+"/submit", nil))
	if submitRecorder.Code != http.StatusOK {
		t.Fatalf("expected submit 200, got %d: %s", submitRecorder.Code, submitRecorder.Body.String())
	}
	if plans.plans[plan.ID].Status != domain.ActionPlanPendingApproval {
		t.Fatalf("expected PENDING_APPROVAL, got %s", plans.plans[plan.ID].Status)
	}

	approveRecorder := httptest.NewRecorder()
	mux.ServeHTTP(approveRecorder, httptest.NewRequest(http.MethodPost, "/api/v1/action-plans/"+plan.ID+"/approve", bytes.NewBufferString(`{"approvedBy":"tester","comment":"approved for test"}`)))
	if approveRecorder.Code != http.StatusOK {
		t.Fatalf("expected approve 200, got %d: %s", approveRecorder.Code, approveRecorder.Body.String())
	}
	if plans.plans[plan.ID].Status != domain.ActionPlanReadyToApply {
		t.Fatalf("expected READY_TO_APPLY, got %s", plans.plans[plan.ID].Status)
	}
	if len(approvals.approvals) != 1 {
		t.Fatalf("expected approval to persist")
	}
}

func TestFinOpsHandler_ShouldGenerateDryRunOnlyWhenReady(t *testing.T) {
	t.Parallel()
	handler, plans, _ := newTestFinOpsHandler()
	mux := http.NewServeMux()
	handler.Register(mux)
	plan := newTestPlan(t)
	plan.Status = domain.ActionPlanReadyToApply
	plans.plans[plan.ID] = plan

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/action-plans/"+plan.ID+"/dry-run", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected dry-run 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestFinOpsHandler_ShouldRejectExecuteWhenRuntimeNotConfigured(t *testing.T) {
	t.Parallel()
	handler, plans, _ := newTestFinOpsHandler()
	mux := http.NewServeMux()
	handler.Register(mux)
	plan := newTestPlan(t)
	plan.Status = domain.ActionPlanReadyToApply
	plans.plans[plan.ID] = plan

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/action-plans/"+plan.ID+"/execute", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
