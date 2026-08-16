package azure

import (
	"context"
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

// WorkloadSource discovers workloads that belong to the analysis target.
// For Azure Kubernetes Service, workload discovery is intentionally delegated
// to the Kubernetes control plane instead of treating Azure VMs as workloads.
type WorkloadSource interface {
	ListWorkloads(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
		namespace string,
	) ([]domain.WorkloadMetrics, error)
}

type KubernetesWorkloadSource struct {
	metrics WorkloadMetricsReader
}

// WorkloadMetricsReader is the minimal adapter needed to reuse the existing
// Kubernetes workload metrics implementation for AKS/GKE/EKS runtimes.
type WorkloadMetricsReader interface {
	GetWorkloads(
		ctx context.Context,
		analysisContext domain.AnalysisContext,
		namespace string,
	) ([]domain.WorkloadMetrics, error)
}

func NewKubernetesWorkloadSource(
	metrics WorkloadMetricsReader,
) (*KubernetesWorkloadSource, error) {

	if metrics == nil {
		return nil, fmt.Errorf(
			"Azure Kubernetes workload metrics source must not be nil",
		)
	}

	return &KubernetesWorkloadSource{
		metrics: metrics,
	}, nil
}

func (s *KubernetesWorkloadSource) ListWorkloads(
	ctx context.Context,
	analysisContext domain.AnalysisContext,
	namespace string,
) ([]domain.WorkloadMetrics, error) {

	if s == nil || s.metrics == nil {
		return nil, fmt.Errorf(
			"Azure Kubernetes workload source is not configured",
		)
	}

	return s.metrics.GetWorkloads(
		ctx,
		analysisContext,
		namespace,
	)
}

var _ WorkloadSource = (*KubernetesWorkloadSource)(nil)
