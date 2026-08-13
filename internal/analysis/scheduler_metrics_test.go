package analysis

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type schedulerIntegrationAnalyzer struct{}

func (a *schedulerIntegrationAnalyzer) Analyze(
	ctx context.Context,
	options AnalysisOptions,
) (*AnalysisReport, error) {

	return &AnalysisReport{
		Summary: AnalysisSummary{
			TotalWorkloads: 2,

			OptimizableWorkloads: 1,

			CurrentMonthlyCostUSD: 1000,

			OptimizedMonthlyCostUSD: 750,

			PotentialSavingsUSD: 250,

			SavingsPercentage: 25,
		},

		NamespaceBreakdown: []NamespaceCostBreakdown{
			{
				Namespace: options.Namespace,

				WorkloadCount: 2,

				OptimizableWorkloads: 1,

				CurrentMonthlyCostUSD: 1000,

				OptimizedMonthlyCostUSD: 750,

				PotentialSavingsUSD: 250,

				SavingsPercentage: 25,
			},
		},

		Workloads: []WorkloadAnalysis{
			{
				Workload: domain.WorkloadMetrics{
					Namespace: options.Namespace,

					Name: "payments-api",
				},
			},
		},
	}, nil
}

func TestScheduler_ShouldSendReportToMetricsSink(
	t *testing.T,
) {

	// Arrange

	analyzer :=
		&schedulerIntegrationAnalyzer{}

	metrics :=
		&schedulerMetricsMock{}

	scheduler :=
		NewScheduler(
			analyzer,
			metrics,
			nil,
			SchedulerConfig{
				Namespace: "cloud-efficiency-engine",

				Interval: time.Hour,

				LookbackHours: 24,

				Step: 5 * time.Minute,
			},
		)

	ctx, cancel :=
		context.WithCancel(
			context.Background(),
		)

	done :=
		make(
			chan struct{},
		)

	// Act

	go func() {

		scheduler.Run(
			ctx,
		)

		close(done)
	}()

	deadline :=
		time.After(
			1 * time.Second,
		)

	for {

		if len(metrics.reports) == 1 {
			break
		}

		select {

		case <-deadline:

			t.Fatal(
				"expected scheduled metrics update",
			)

		case <-time.After(
			5 * time.Millisecond,
		):
		}
	}

	cancel()

	select {

	case <-done:
		// expected

	case <-time.After(
		1 * time.Second,
	):

		t.Fatal(
			"scheduler did not stop",
		)
	}

	// Assert

	report :=
		metrics.reports[0]

	if report.Summary.
		PotentialSavingsUSD != 250 {

		t.Fatalf(
			"expected savings 250, got %.2f",
			report.Summary.PotentialSavingsUSD,
		)
	}

	if len(
		report.NamespaceBreakdown,
	) != 1 {

		t.Fatalf(
			"expected 1 namespace breakdown, got %d",
			len(
				report.NamespaceBreakdown,
			),
		)
	}
}
