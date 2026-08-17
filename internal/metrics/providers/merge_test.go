package providers

import (
	"testing"

	"cloud-efficiency-engine/internal/domain"
)

func TestMergeMetrics_ShouldMergeSameWorkloadAcrossMetrics(t *testing.T) {
	metric := func(container, kind string, value string) prometheusResult {
		return prometheusResult{
			Metric: map[string]string{
				"namespace":     "payments",
				"workload":      "payments-api",
				"node_group":    "general-a",
				"container":     container,
				"workload_kind": kind,
			},
			Value: []interface{}{
				float64(1700000000),
				value,
			},
		}
	}

	result, err := mergeMetrics(
		[]prometheusResult{metric("api", "Deployment", "1000")},
		[]prometheusResult{metric("api", "Deployment", "150")},
		[]prometheusResult{metric("api", "Deployment", "1073741824")},
		[]prometheusResult{metric("api", "Deployment", "536870912")},
		[]prometheusResult{metric("api", "Deployment", "3")},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 workload, got %d", len(result))
	}

	workload := result[0]
	if workload.Namespace != "payments" {
		t.Errorf("expected namespace payments, got %s", workload.Namespace)
	}
	if workload.Name != "payments-api" {
		t.Errorf("expected workload payments-api, got %s", workload.Name)
	}
	if workload.NodeGroup != "general-a" {
		t.Errorf("expected node group general-a, got %s", workload.NodeGroup)
	}
	if workload.ContainerName != "api" {
		t.Errorf("expected container api, got %s", workload.ContainerName)
	}
	if workload.Type != domain.WorkloadDeployment {
		t.Errorf("expected workload type Deployment, got %s", workload.Type)
	}
	if workload.CPURequestMillicores != 1000 {
		t.Errorf("expected CPU request 1000, got %d", workload.CPURequestMillicores)
	}
	if workload.CPUUsageMillicores != 150 {
		t.Errorf("expected CPU usage 150, got %d", workload.CPUUsageMillicores)
	}
	if workload.MemoryRequestBytes != 1073741824 {
		t.Errorf("expected memory request 1073741824, got %d", workload.MemoryRequestBytes)
	}
	if workload.MemoryUsageBytes != 536870912 {
		t.Errorf("expected memory usage 536870912, got %d", workload.MemoryUsageBytes)
	}
	if workload.Replicas != 3 {
		t.Errorf("expected replicas 3, got %d", workload.Replicas)
	}
}

func TestMergeMetrics_ShouldClearAmbiguousNodeGroupPlacement(t *testing.T) {
	metric := func(nodeGroup string) prometheusResult {
		return prometheusResult{
			Metric: map[string]string{
				"namespace":  "payments",
				"workload":   "payments-api",
				"node_group": nodeGroup,
			},
			Value: []interface{}{
				float64(1700000000),
				"100",
			},
		}
	}

	result, err := mergeMetrics(
		[]prometheusResult{metric("general-a")},
		[]prometheusResult{metric("general-b")},
		[]prometheusResult{metric("general-a")},
		[]prometheusResult{metric("general-b")},
		[]prometheusResult{metric("general-a")},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 workload, got %d", len(result))
	}
	if result[0].NodeGroup != "" {
		t.Fatalf("expected ambiguous node group placement to be empty, got %q", result[0].NodeGroup)
	}
}

func TestMergeMetrics_ShouldClearAmbiguousContainerTarget(t *testing.T) {
	metric := func(container string) prometheusResult {
		return prometheusResult{
			Metric: map[string]string{
				"namespace":     "payments",
				"workload":      "payments-api",
				"container":     container,
				"workload_kind": "Deployment",
			},
			Value: []interface{}{
				float64(1700000000),
				"100",
			},
		}
	}

	result, err := mergeMetrics(
		[]prometheusResult{metric("api")},
		[]prometheusResult{metric("sidecar")},
		[]prometheusResult{metric("api")},
		[]prometheusResult{metric("sidecar")},
		[]prometheusResult{metric("api")},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 workload, got %d", len(result))
	}
	if result[0].ContainerName != "" {
		t.Fatalf("expected ambiguous container target to be empty, got %q", result[0].ContainerName)
	}
	if result[0].Type != domain.WorkloadDeployment {
		t.Fatalf("expected workload type Deployment, got %s", result[0].Type)
	}
}

func TestNormalizeWorkloadType(t *testing.T) {
	cases := []struct {
		input string
		want  domain.WorkloadType
	}{
		{input: "Deployment", want: domain.WorkloadDeployment},
		{input: "statefulset", want: domain.WorkloadStatefulSet},
		{input: "DaemonSet", want: domain.WorkloadDaemonSet},
		{input: "Job", want: domain.WorkloadJob},
		{input: "SomethingElse", want: domain.WorkloadUnknown},
	}

	for _, testCase := range cases {
		if got := normalizeWorkloadType(testCase.input); got != testCase.want {
			t.Fatalf("normalizeWorkloadType(%q) = %q, want %q", testCase.input, got, testCase.want)
		}
	}
}
