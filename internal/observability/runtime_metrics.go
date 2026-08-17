package observability

import (
	"sync"
	"time"

	"cloud-efficiency-engine/internal/domain"
	"github.com/prometheus/client_golang/prometheus"
)

type RuntimeMetrics struct {
	Registry *prometheus.Registry

	executionsTotal      *prometheus.CounterVec
	executionDuration    *prometheus.HistogramVec
	verificationsTotal   *prometheus.CounterVec
	verificationDuration *prometheus.HistogramVec
	providerCallsTotal   *prometheus.CounterVec
	providerDuration     *prometheus.HistogramVec
	realizedSavings      *prometheus.CounterVec
}

var (
	defaultRuntimeOnce     sync.Once
	defaultRuntimeMetrics  *RuntimeMetrics
	defaultRuntimeRegistry *prometheus.Registry
)

func NewRuntimeMetrics(registry *prometheus.Registry) *RuntimeMetrics {
	if registry == nil {
		registry = prometheus.NewRegistry()
	}

	m := &RuntimeMetrics{
		Registry: registry,
		executionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "cee_execution_total", Help: "Total FinOps action executions by provider, action, and outcome."},
			[]string{"provider", "action", "outcome"},
		),
		executionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "cee_execution_duration_seconds", Help: "Execution duration in seconds by provider and action."},
			[]string{"provider", "action"},
		),
		verificationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "cee_verification_total", Help: "Total verification attempts by provider and outcome."},
			[]string{"provider", "outcome"},
		),
		verificationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "cee_verification_duration_seconds", Help: "Verification duration in seconds by provider."},
			[]string{"provider"},
		),
		providerCallsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "cee_provider_operation_total", Help: "Total provider operations by provider, operation, and outcome."},
			[]string{"provider", "operation", "outcome"},
		),
		providerDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{Name: "cee_provider_operation_duration_seconds", Help: "Provider operation duration in seconds."},
			[]string{"provider", "operation"},
		),
		realizedSavings: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "cee_realized_savings_monthly_usd_total", Help: "Monthly savings in USD realized by successful FinOps actions."},
			[]string{"provider", "action"},
		),
	}

	for _, collector := range []prometheus.Collector{
		m.executionsTotal,
		m.executionDuration,
		m.verificationsTotal,
		m.verificationDuration,
		m.providerCallsTotal,
		m.providerDuration,
		m.realizedSavings,
	} {
		registry.MustRegister(collector)
	}

	return m
}

func DefaultRuntimeMetrics() *RuntimeMetrics {
	defaultRuntimeOnce.Do(func() {
		defaultRuntimeRegistry = prometheus.NewRegistry()
		defaultRuntimeMetrics = NewRuntimeMetrics(defaultRuntimeRegistry)
	})
	return defaultRuntimeMetrics
}

func DefaultRuntimeRegistry() *prometheus.Registry {
	DefaultRuntimeMetrics()
	return defaultRuntimeRegistry
}

func (m *RuntimeMetrics) RecordExecution(action domain.Action, outcome string, duration time.Duration, realized bool) {
	if m == nil {
		return
	}
	provider := string(action.Provider)
	if provider == "" {
		provider = "unknown"
	}
	name := string(action.Type)
	if name == "" {
		name = "unknown"
	}
	m.executionsTotal.WithLabelValues(provider, name, outcome).Inc()
	m.executionDuration.WithLabelValues(provider, name).Observe(duration.Seconds())
	if realized && action.MonthlySavingsUSD > 0 {
		m.realizedSavings.WithLabelValues(provider, name).Add(action.MonthlySavingsUSD)
	}
}

func (m *RuntimeMetrics) RecordVerification(provider domain.CloudProvider, outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	value := string(provider)
	if value == "" {
		value = "unknown"
	}
	m.verificationsTotal.WithLabelValues(value, outcome).Inc()
	m.verificationDuration.WithLabelValues(value).Observe(duration.Seconds())
}

func (m *RuntimeMetrics) RecordProviderOperation(provider domain.CloudProvider, operation, outcome string, duration time.Duration) {
	if m == nil {
		return
	}
	value := string(provider)
	if value == "" {
		value = "unknown"
	}
	m.providerCallsTotal.WithLabelValues(value, operation, outcome).Inc()
	m.providerDuration.WithLabelValues(value, operation).Observe(duration.Seconds())
}
