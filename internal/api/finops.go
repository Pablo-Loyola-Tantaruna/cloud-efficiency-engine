package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cloud-efficiency-engine/internal/analysis/actions"
	rediscache "cloud-efficiency-engine/internal/cache/redis"
	"cloud-efficiency-engine/internal/domain"
)

type FinOpsHandler struct {
	plans        actions.ActionPlanRepository
	approvals    actions.ActionApprovalRepository
	recoveries   actions.RecoveryActionRepository
	executions   actions.ExecutionRecordRepository
	audit        actions.AuditEventRepository
	verification actions.VerificationResultRepository
	executionEng *actions.ExecutionEngine
	cache        *rediscache.Client
	lockTTL      time.Duration
}

func NewFinOpsHandler(plans actions.ActionPlanRepository, approvals actions.ActionApprovalRepository, recoveries actions.RecoveryActionRepository, executions actions.ExecutionRecordRepository, audit actions.AuditEventRepository, verification actions.VerificationResultRepository, executionEngine *actions.ExecutionEngine, cache *rediscache.Client) *FinOpsHandler {
	return &FinOpsHandler{plans: plans, approvals: approvals, recoveries: recoveries, executions: executions, audit: audit, verification: verification, executionEng: executionEngine, cache: cache, lockTTL: 2 * time.Minute}
}

func (h *FinOpsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/action-plans", h.actionPlans)
	mux.HandleFunc("/api/v1/action-plans/", h.actionPlan)
	mux.HandleFunc("/api/v1/executions/", h.execution)
	mux.HandleFunc("/api/v1/recovery/", h.recovery)
}

func (h *FinOpsHandler) actionPlans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, ErrCodeInvalidRequest, "method not allowed", requestIDFromContext(r.Context()))
		return
	}
	if h == nil || h.plans == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeAnalysisFailed, "FinOps persistence is not configured", requestIDFromContext(r.Context()))
		return
	}
	var request struct {
		Provider domain.CloudProvider `json:"provider"`
		Cluster  string               `json:"cluster"`
		Actions  []domain.Action      `json:"actions"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	plan, err := actions.BuildActionPlan(request.Provider, request.Cluster, request.Actions)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	created, err := h.plans.CreateActionPlanIfAbsent(plan)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeAnalysisFailed, "could not persist action plan", requestIDFromContext(r.Context()))
		return
	}
	writeJSON(w, map[string]any{"created": created, "plan": plan}, http.StatusCreated, requestIDFromContext(r.Context()))
}

func (h *FinOpsHandler) actionPlan(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/action-plans/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "action plan id is required", requestIDFromContext(r.Context()))
		return
	}
	planID := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		h.getActionPlan(w, r, planID)
		return
	}
	if len(parts) != 2 || r.Method != http.MethodPost {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "unsupported action plan operation", requestIDFromContext(r.Context()))
		return
	}
	switch parts[1] {
	case "submit":
		h.submitActionPlan(w, r, planID)
	case "approve":
		h.approveActionPlan(w, r, planID)
	case "dry-run":
		h.dryRunActionPlan(w, r, planID)
	case "execute":
		h.executeActionPlan(w, r, planID)
	default:
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "unsupported action plan operation", requestIDFromContext(r.Context()))
	}
}

func (h *FinOpsHandler) getActionPlan(w http.ResponseWriter, r *http.Request, planID string) {
	cacheKey := "finops:plan:" + planID
	if h.cache != nil {
		var cached domain.ActionPlan
		if found, err := h.cache.GetJSON(r.Context(), cacheKey, &cached); err == nil && found {
			writeJSON(w, cached, http.StatusOK, requestIDFromContext(r.Context()))
			return
		}
	}
	plan, ok, err := h.plans.GetActionPlanByID(planID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeAnalysisFailed, "could not load action plan", requestIDFromContext(r.Context()))
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "action plan not found", requestIDFromContext(r.Context()))
		return
	}
	if h.cache != nil {
		_ = h.cache.SetJSON(r.Context(), cacheKey, plan, 30*time.Second)
	}
	writeJSON(w, plan, http.StatusOK, requestIDFromContext(r.Context()))
}

func (h *FinOpsHandler) invalidatePlanCache(ctx context.Context, planID string) {
	if h.cache != nil {
		_ = h.cache.Delete(ctx, "finops:plan:"+planID)
	}
}

func (h *FinOpsHandler) submitActionPlan(w http.ResponseWriter, r *http.Request, planID string) {
	plan, ok, err := h.plans.GetActionPlanByID(planID)
	if err != nil || !ok {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "action plan not found", requestIDFromContext(r.Context()))
		return
	}
	if plan.Status != domain.ActionPlanPreview {
		writeError(w, http.StatusConflict, ErrCodeInvalidRequest, fmt.Sprintf("action plan is %s, expected PREVIEW", plan.Status), requestIDFromContext(r.Context()))
		return
	}
	plan.Status = domain.ActionPlanPendingApproval
	if err := h.plans.UpdateActionPlan(plan); err != nil {
		writeError(w, http.StatusConflict, ErrCodeInvalidRequest, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	h.invalidatePlanCache(r.Context(), planID)
	writeJSON(w, plan, http.StatusOK, requestIDFromContext(r.Context()))
}

func (h *FinOpsHandler) approveActionPlan(w http.ResponseWriter, r *http.Request, planID string) {
	plan, ok, err := h.plans.GetActionPlanByID(planID)
	if err != nil || !ok {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "action plan not found", requestIDFromContext(r.Context()))
		return
	}
	var request struct {
		ApprovedBy string `json:"approvedBy"`
		Comment    string `json:"comment"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	approved, approval, err := actions.ApproveActionPlan(plan, request.ApprovedBy, request.Comment, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusConflict, ErrCodeInvalidRequest, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	if err := h.plans.UpdateActionPlan(approved); err != nil {
		writeError(w, http.StatusConflict, ErrCodeInvalidRequest, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	if err := h.approvals.SaveApproval(approval); err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeAnalysisFailed, "could not persist approval", requestIDFromContext(r.Context()))
		return
	}
	approved.Status = domain.ActionPlanReadyToApply
	if err := h.plans.UpdateActionPlan(approved); err != nil {
		writeError(w, http.StatusConflict, ErrCodeInvalidRequest, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	h.invalidatePlanCache(r.Context(), planID)
	writeJSON(w, map[string]any{"plan": approved, "approval": approval}, http.StatusOK, requestIDFromContext(r.Context()))
}

func (h *FinOpsHandler) dryRunActionPlan(w http.ResponseWriter, r *http.Request, planID string) {
	plan, ok, err := h.plans.GetActionPlanByID(planID)
	if err != nil || !ok {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "action plan not found", requestIDFromContext(r.Context()))
		return
	}
	preview, err := actions.BuildDryRunExecution(plan)
	if err != nil {
		writeError(w, http.StatusConflict, ErrCodeInvalidRequest, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeJSON(w, preview, http.StatusOK, requestIDFromContext(r.Context()))
}

func (h *FinOpsHandler) executeActionPlan(w http.ResponseWriter, r *http.Request, planID string) {
	if h.executionEng == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeAnalysisFailed, "execution runtime is not configured", requestIDFromContext(r.Context()))
		return
	}
	plan, ok, err := h.plans.GetActionPlanByID(planID)
	if err != nil || !ok {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "action plan not found", requestIDFromContext(r.Context()))
		return
	}
	var request struct {
		ActionID string `json:"actionId"`
	}
	if err := decodeOptionalJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	if request.ActionID == "" {
		if len(plan.Actions) != 1 {
			writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "actionId is required for plans with multiple actions", requestIDFromContext(r.Context()))
			return
		}
		request.ActionID = plan.Actions[0].ID
	}
	var action *domain.Action
	for i := range plan.Actions {
		if plan.Actions[i].ID == request.ActionID {
			action = &plan.Actions[i]
			break
		}
	}
	if action == nil {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "action not found in plan", requestIDFromContext(r.Context()))
		return
	}

	lockKey := "finops:execution-lock:" + plan.ID + ":" + action.ID
	if h.cache != nil {
		token, locked, lockErr := h.cache.TryLock(r.Context(), lockKey, h.lockTTL)
		if lockErr != nil {
			writeError(w, http.StatusServiceUnavailable, ErrCodeAnalysisFailed, "execution coordination unavailable", requestIDFromContext(r.Context()))
			return
		}
		if !locked {
			writeError(w, http.StatusConflict, ErrCodeInvalidRequest, "execution is already in progress", requestIDFromContext(r.Context()))
			return
		}
		defer func() { _ = h.cache.Unlock(context.Background(), lockKey, token) }()
	}

	record, verification, err := h.executionEng.Execute(r.Context(), plan, *action)
	if err != nil {
		status := http.StatusConflict
		if record.ID == "" {
			status = http.StatusInternalServerError
		}
		writeError(w, status, ErrCodeAnalysisFailed, err.Error(), requestIDFromContext(r.Context()))
		return
	}
	writeJSON(w, map[string]any{"execution": record, "verification": verification}, http.StatusOK, requestIDFromContext(r.Context()))
}

func (h *FinOpsHandler) execution(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/executions/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" || h.executions == nil {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "execution not found", requestIDFromContext(r.Context()))
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		record, ok := h.executions.GetByID(id)
		if !ok {
			writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "execution not found", requestIDFromContext(r.Context()))
			return
		}
		writeJSON(w, record, http.StatusOK, requestIDFromContext(r.Context()))
		return
	}
	if len(parts) != 2 || r.Method != http.MethodGet {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "unsupported execution operation", requestIDFromContext(r.Context()))
		return
	}
	switch parts[1] {
	case "history":
		record, ok := h.executions.GetByID(id)
		if !ok {
			writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "execution not found", requestIDFromContext(r.Context()))
			return
		}
		writeJSON(w, h.executions.ListByIdempotencyKey(record.IdempotencyKey), http.StatusOK, requestIDFromContext(r.Context()))
	case "audit":
		if h.audit == nil {
			writeError(w, http.StatusServiceUnavailable, ErrCodeAnalysisFailed, "audit persistence is not configured", requestIDFromContext(r.Context()))
			return
		}
		writeJSON(w, h.audit.ListByExecution(id), http.StatusOK, requestIDFromContext(r.Context()))
	case "verification":
		if h.verification == nil {
			writeError(w, http.StatusServiceUnavailable, ErrCodeAnalysisFailed, "verification persistence is not configured", requestIDFromContext(r.Context()))
			return
		}
		result, ok := h.verification.GetByExecutionID(id)
		if !ok {
			writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "verification result not found", requestIDFromContext(r.Context()))
			return
		}
		writeJSON(w, result, http.StatusOK, requestIDFromContext(r.Context()))
	default:
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "unsupported execution operation", requestIDFromContext(r.Context()))
	}
}

func (h *FinOpsHandler) recovery(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/recovery/"), "/"), "/")
	if len(parts) != 1 || parts[0] == "" || r.Method != http.MethodGet || h.recoveries == nil {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "recovery action not found", requestIDFromContext(r.Context()))
		return
	}
	action, ok, err := h.recoveries.GetRecoveryByID(parts[0])
	if err != nil || !ok {
		writeError(w, http.StatusNotFound, ErrCodeInvalidRequest, "recovery action not found", requestIDFromContext(r.Context()))
		return
	}
	writeJSON(w, action, http.StatusOK, requestIDFromContext(r.Context()))
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON request body: %w", err)
	}
	return nil
}

func decodeOptionalJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("invalid JSON request body: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, value any, status int, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	if requestID != "" {
		w.Header().Set("X-Request-ID", requestID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
