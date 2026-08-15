package analysis

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud-efficiency-engine/internal/domain"
)

type schedulerAnalyzerMock struct {
	report *AnalysisReport

	err error

	calls int

	lastOptions AnalysisOptions

	called chan struct{}
}

func newSchedulerAnalyzerMock(
	report *AnalysisReport,
	err error,
) *schedulerAnalyzerMock {

	return &schedulerAnalyzerMock{
		report: report,

		err: err,

		called: make(
			chan struct{},
		),
	}
}

func (m *schedulerAnalyzerMock) Analyze(
	ctx context.Context,
	options AnalysisOptions,
) (*AnalysisReport, error) {

	m.calls++

	m.lastOptions =
		options

	select {

	case <-m.called:

	default:

		close(
			m.called,
		)
	}

	return m.report, m.err
}

type schedulerMetricsMock struct {
	reports []*AnalysisReport

	successCalls int

	failureCalls int

	lastNamespace string

	updated chan struct{}
}

func newSchedulerMetricsMock() *schedulerMetricsMock {

	return &schedulerMetricsMock{
		updated: make(
			chan struct{},
		),
	}
}

func (m *schedulerMetricsMock) Update(
	report *AnalysisReport,
) {

	m.reports =
		append(
			m.reports,
			report,
		)

	select {

	case <-m.updated:

	default:

		close(
			m.updated,
		)
	}
}

func (
	m *schedulerMetricsMock,
) RecordSchedulerSuccess(
	namespace string,
	startedAt time.Time,
) {

	m.successCalls++

	m.lastNamespace =
		namespace
}

func (
	m *schedulerMetricsMock,
) RecordSchedulerFailure(
	namespace string,
	startedAt time.Time,
) {

	m.failureCalls++

	m.lastNamespace =
		namespace
}

func waitForSignal(
	t *testing.T,
	signal <-chan struct{},
	message string,
) {

	t.Helper()

	select {

	case <-signal:

	case <-time.After(
		1 * time.Second,
	):

		t.Fatal(
			message,
		)
	}
}

func stopScheduler(
	t *testing.T,
	cancel context.CancelFunc,
	done <-chan struct{},
) {

	t.Helper()

	cancel()

	select {

	case <-done:

	case <-time.After(
		1 * time.Second,
	):

		t.Fatal(
			"scheduler did not stop after cancellation",
		)
	}
}

func TestSchedulerRun_ShouldExecuteAnalysisImmediately(
	t *testing.T,
) {

	report :=
		&AnalysisReport{
			Summary: AnalysisSummary{
				TotalWorkloads: 3,
			},
		}

	analyzer :=
		newSchedulerAnalyzerMock(
			report,
			nil,
		)

	metrics :=
		newSchedulerMetricsMock()

	scheduler :=
		NewScheduler(
			analyzer,
			metrics,
			nil,
			SchedulerConfig{
				Namespace: "cloud-efficiency-engine",

				Context: domain.AnalysisContext{
					Provider: domain.CloudProviderKubernetes,

					Environment: "test",

					ClusterName: "test-cluster",
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

	waitForSignal(
		t,
		analyzer.called,
		"expected initial analysis execution",
	)

	stopScheduler(
		t,
		cancel,
		done,
	)

	if analyzer.calls != 1 {

		t.Fatalf(
			"expected 1 analysis call, got %d",
			analyzer.calls,
		)
	}

	if len(metrics.reports) != 1 {

		t.Fatalf(
			"expected 1 metrics update, got %d",
			len(metrics.reports),
		)
	}

	if metrics.successCalls != 1 {

		t.Fatalf(
			"expected 1 scheduler success call, got %d",
			metrics.successCalls,
		)
	}

	if metrics.failureCalls != 0 {

		t.Fatalf(
			"expected 0 scheduler failure calls, got %d",
			metrics.failureCalls,
		)
	}

	if metrics.lastNamespace !=
		"cloud-efficiency-engine" {

		t.Fatalf(
			"expected namespace cloud-efficiency-engine, got %s",
			metrics.lastNamespace,
		)
	}

	if analyzer.lastOptions.Context.Provider !=
		domain.CloudProviderKubernetes {

		t.Fatalf(
			"expected kubernetes provider, got %s",
			analyzer.lastOptions.Context.Provider,
		)
	}

	if analyzer.lastOptions.Context.ClusterName !=
		"test-cluster" {

		t.Fatalf(
			"expected test-cluster, got %s",
			analyzer.lastOptions.Context.ClusterName,
		)
	}
}

func TestSchedulerNormalizedConfig_ShouldApplyDefaults(
	t *testing.T,
) {

	scheduler :=
		NewScheduler(
			newSchedulerAnalyzerMock(
				nil,
				nil,
			),
			newSchedulerMetricsMock(),
			nil,
			SchedulerConfig{},
		)

	config :=
		scheduler.normalizedConfig()

	if config.Interval !=
		5*time.Minute {

		t.Fatalf(
			"expected default interval 5m, got %s",
			config.Interval,
		)
	}

	if config.LookbackHours !=
		24 {

		t.Fatalf(
			"expected default lookback 24h, got %d",
			config.LookbackHours,
		)
	}

	if config.Step !=
		5*time.Minute {

		t.Fatalf(
			"expected default step 5m, got %s",
			config.Step,
		)
	}

	if config.Context.Provider !=
		domain.CloudProviderKubernetes {

		t.Fatalf(
			"expected kubernetes provider, got %s",
			config.Context.Provider,
		)
	}

	if config.Context.Environment !=
		"unknown" {

		t.Fatalf(
			"expected unknown environment, got %s",
			config.Context.Environment,
		)
	}
}

func TestSchedulerRun_ShouldRecordFailure(
	t *testing.T,
) {

	analyzer :=
		newSchedulerAnalyzerMock(
			nil,
			errors.New(
				"analysis failed",
			),
		)

	metrics :=
		newSchedulerMetricsMock()

	scheduler :=
		NewScheduler(
			analyzer,
			metrics,
			nil,
			SchedulerConfig{
				Namespace: "cloud-efficiency-engine",

				Context: domain.AnalysisContext{
					Provider: domain.CloudProviderKubernetes,

					Environment: "test",
				},

				Interval: 1 * time.Hour,
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

	waitForSignal(
		t,
		analyzer.called,
		"expected analysis execution",
	)

	stopScheduler(
		t,
		cancel,
		done,
	)

	if len(metrics.reports) != 0 {

		t.Fatalf(
			"expected no metrics updates, got %d",
			len(metrics.reports),
		)
	}

	if metrics.successCalls != 0 {

		t.Fatalf(
			"expected 0 success calls, got %d",
			metrics.successCalls,
		)
	}

	if metrics.failureCalls != 1 {

		t.Fatalf(
			"expected 1 failure call, got %d",
			metrics.failureCalls,
		)
	}
}

func TestSchedulerRun_ShouldRecordFailureWhenReportIsNil(
	t *testing.T,
) {

	analyzer :=
		newSchedulerAnalyzerMock(
			nil,
			nil,
		)

	metrics :=
		newSchedulerMetricsMock()

	scheduler :=
		NewScheduler(
			analyzer,
			metrics,
			nil,
			SchedulerConfig{
				Namespace: "cloud-efficiency-engine",

				Context: domain.AnalysisContext{
					Provider: domain.CloudProviderKubernetes,

					Environment: "test",
				},

				Interval: 1 * time.Hour,
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

	waitForSignal(
		t,
		analyzer.called,
		"expected analysis execution",
	)

	stopScheduler(
		t,
		cancel,
		done,
	)

	if metrics.successCalls != 0 {

		t.Fatalf(
			"expected 0 success calls, got %d",
			metrics.successCalls,
		)
	}

	if metrics.failureCalls != 1 {

		t.Fatalf(
			"expected 1 failure call, got %d",
			metrics.failureCalls,
		)
	}

	if len(metrics.reports) != 0 {

		t.Fatalf(
			"expected no metrics updates for nil report, got %d",
			len(metrics.reports),
		)
	}
}
