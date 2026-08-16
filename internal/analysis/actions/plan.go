package actions

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"cloud-efficiency-engine/internal/domain"
)

func BuildActionPlan(provider domain.CloudProvider, cluster string, actions []domain.Action) (domain.ActionPlan, error) {
	cluster = strings.TrimSpace(cluster)
	if !provider.IsValid() {
		return domain.ActionPlan{}, fmt.Errorf("provider must be valid")
	}
	if cluster == "" {
		return domain.ActionPlan{}, fmt.Errorf("cluster must not be empty")
	}
	if len(actions) == 0 {
		return domain.ActionPlan{}, fmt.Errorf("at least one action is required")
	}

	validated := make([]domain.Action, len(actions))
	copy(validated, actions)
	var monthly float64
	var annual float64
	for i := range validated {
		if err := validated[i].Validate(); err != nil {
			return domain.ActionPlan{}, fmt.Errorf("action %q: %w", validated[i].ID, err)
		}
		if validated[i].Provider != provider || validated[i].Cluster != cluster {
			return domain.ActionPlan{}, fmt.Errorf("action %q does not belong to provider %q and cluster %q", validated[i].ID, provider, cluster)
		}
		monthly += validated[i].MonthlySavingsUSD
		annual += validated[i].AnnualizedSavingsUSD
	}

	plan := domain.ActionPlan{
		ID:                        actionPlanID(provider, cluster, validated),
		Provider:                  provider,
		Cluster:                   cluster,
		Status:                    domain.ActionPlanPreview,
		Actions:                   validated,
		TotalMonthlySavingsUSD:    monthly,
		TotalAnnualizedSavingsUSD: annual,
		RequiresApproval:          true,
	}
	if err := plan.Validate(); err != nil {
		return domain.ActionPlan{}, err
	}
	return plan, nil
}

func actionPlanID(provider domain.CloudProvider, cluster string, actions []domain.Action) string {
	h := sha256.New()
	_, _ = h.Write([]byte(string(provider)))
	_, _ = h.Write([]byte("\x00"))
	_, _ = h.Write([]byte(cluster))
	for _, action := range actions {
		_, _ = h.Write([]byte("\x00"))
		_, _ = h.Write([]byte(action.ID))
	}
	return "plan-" + hex.EncodeToString(h.Sum(nil))[:16]
}
