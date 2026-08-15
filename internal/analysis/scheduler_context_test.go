package analysis

import (
	"context"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type schedulerContextAnalyzerMock struct {
	receivedContext domain.AnalysisContext

	called chan struct{}
}

func (
	m *schedulerContextAnalyzerMock,
) Analyze(
	ctx context.Context,
	options AnalysisOptions,
) (*AnalysisReport, error) {

	m.receivedContext =
		options.Context

	select {

	case <-m.called:

	default:

		close(
			m.called,
		)
	}

	return &AnalysisReport{
		Context: options.Context,

		Summary: AnalysisSummary{},
	}, nil
}

func TestScheduler_ShouldPassAnalysisContext(
	t *testing.T,
) {

	analyzer :=
		&schedulerContextAnalyzerMock{
			called: make(
				chan struct{},
			),
		}

	scheduler :=
		NewScheduler(
			analyzer,
			nil,
			nil,
			SchedulerConfig{
				Namespace: "payments",

				Context: domain.AnalysisContext{
					Provider: domain.CloudProviderAWS,

					Environment: "production",

					AccountID: "123456789",

					Region: "us-east-1",

					ClusterName: "prod-eks",
				},

				Interval: 1 * time.Hour,

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

	go func() {

		scheduler.Run(
			ctx,
		)

		close(done)
	}()

	select {

	case <-analyzer.called:

	case <-time.After(
		1 * time.Second,
	):

		t.Fatal(
			"expected scheduler to execute analysis",
		)
	}

	cancel()

	select {

	case <-done:

	case <-time.After(
		1 * time.Second,
	):

		t.Fatal(
			"scheduler did not stop",
		)
	}

	if analyzer.
		receivedContext.
		Provider !=
		domain.CloudProviderAWS {

		t.Fatalf(
			"expected AWS provider, got %s",
			analyzer.receivedContext.Provider,
		)
	}

	if analyzer.
		receivedContext.
		Region !=
		"us-east-1" {

		t.Fatalf(
			"expected us-east-1, got %s",
			analyzer.receivedContext.Region,
		)
	}

	if analyzer.
		receivedContext.
		AccountID !=
		"123456789" {

		t.Fatalf(
			"expected account 123456789, got %s",
			analyzer.receivedContext.AccountID,
		)
	}
}
