package analysis

import (
	"context"
	"errors"
	"testing"
	"time"
)

type schedulerAnalyzerMock struct {
	report *AnalysisReport

	err error

	calls int

	lastOptions AnalysisOptions
}

func (m *schedulerAnalyzerMock) Analyze(
	ctx context.Context,
	options AnalysisOptions,
) (*AnalysisReport, error) {

	m.calls++

	m.lastOptions =
		options

	return m.report, m.err
}

type schedulerMetricsMock struct {
	reports []*AnalysisReport

	successCalls int

	failureCalls int

	lastNamespace string
}

func (m *schedulerMetricsMock) Update(
	report *AnalysisReport,
) {

	m.reports =
		append(
			m.reports,
			report,
		)
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
		&schedulerAnalyzerMock{
			report: report,
		}

	metrics :=
		&schedulerMetricsMock{}

	scheduler :=
		NewScheduler(
			analyzer,
			metrics,
			nil,
			SchedulerConfig{
				Namespace: "cloud-efficiency-engine",

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

	deadline :=
		time.After(
			1 * time.Second,
		)

	for {

		if analyzer.calls >= 1 {
			break
		}

		select {

		case <-deadline:

			t.Fatal(
				"expected initial analysis execution",
			)

		case <-time.After(
			5 * time.Millisecond,
		):
		}
	}

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
			metrics.lastNamespace)
	}

}

func TestSchedulerNormalizedConfig_ShouldApplyDefaults(
	t *testing.T,
) {

	scheduler :=
		NewScheduler(
			&schedulerAnalyzerMock{},
			&schedulerMetricsMock{},
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
}

func TestSchedulerRun_ShouldRecordFailure(
	t *testing.T,
) {

	analyzer :=
		&schedulerAnalyzerMock{
			err: errors.New(
				"analysis failed",
			),
		}

	metrics :=
		&schedulerMetricsMock{}

	scheduler :=
		NewScheduler(
			analyzer,
			metrics,
			nil,
			SchedulerConfig{
				Namespace: "cloud-efficiency-engine",

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

	deadline :=
		time.After(
			1 * time.Second,
		)

	for {

		if analyzer.calls >= 1 {
			break
		}

		select {

		case <-deadline:

			t.Fatal(
				"expected analysis execution",
			)

		case <-time.After(
			5 * time.Millisecond,
		):
		}
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
		&schedulerAnalyzerMock{
			report: nil,
		}

	metrics :=
		&schedulerMetricsMock{}

	scheduler :=
		NewScheduler(
			analyzer,
			metrics,
			nil,
			SchedulerConfig{
				Namespace: "cloud-efficiency-engine",

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

	deadline :=
		time.After(
			1 * time.Second,
		)

	for {

		if analyzer.calls >= 1 {
			break
		}

		select {

		case <-deadline:

			t.Fatal(
				"expected analysis execution",
			)

		case <-time.After(
			5 * time.Millisecond,
		):
		}
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
