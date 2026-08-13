package observability

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"cloud-efficiency-engine/internal/analysis"
)

type Metrics struct {
	registry *prometheus.Registry

	currentMonthlyCost *prometheus.GaugeVec

	optimizedMonthlyCost *prometheus.GaugeVec

	potentialSavings *prometheus.GaugeVec

	savingsPercentage *prometheus.GaugeVec

	workloadCount *prometheus.GaugeVec

	optimizableWorkloadCount *prometheus.GaugeVec

	workloadCurrentMonthlyCost *prometheus.GaugeVec

	workloadOptimizedMonthlyCost *prometheus.GaugeVec

	workloadPotentialSavings *prometheus.GaugeVec

	workloadSavingsPercentage *prometheus.GaugeVec

	workloadOptimizable *prometheus.GaugeVec

	recommendationCount *prometheus.GaugeVec

	recommendationWorkloadCount *prometheus.GaugeVec

	recommendationByRule *prometheus.GaugeVec

	recommendationBySeverity *prometheus.GaugeVec

	recommendationByConfidence *prometheus.GaugeVec

	schedulerRunsTotal *prometheus.CounterVec

	schedulerSuccessTotal *prometheus.CounterVec

	schedulerFailureTotal *prometheus.CounterVec

	schedulerLastSuccessTimestamp *prometheus.GaugeVec

	schedulerLastFailureTimestamp *prometheus.GaugeVec

	schedulerLastDurationSeconds *prometheus.GaugeVec

	updateMutex sync.Mutex

	schedulerMutex sync.Mutex

	activeNamespaces map[string]struct{}

	activeWorkloads map[workloadKey]struct{}

	activeRecommendationRules map[recommendationRuleKey]struct{}

	activeRecommendationSeverities map[recommendationSeverityKey]struct{}

	activeRecommendationConfidences map[recommendationConfidenceKey]struct{}
}

type workloadKey struct {
	namespace string
	workload  string
}

type recommendationRuleKey struct {
	rule string
}

type recommendationSeverityKey struct {
	severity string
}

type recommendationConfidenceKey struct {
	confidence string
}

func NewMetrics() *Metrics {

	registry :=
		prometheus.NewRegistry()

	metrics :=
		&Metrics{
			registry: registry,

			currentMonthlyCost: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_current_monthly_cost_usd",

					Help: "Current estimated monthly infrastructure cost in USD by namespace.",
				},
				[]string{
					"namespace",
				},
			),

			optimizedMonthlyCost: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_optimized_monthly_cost_usd",

					Help: "Estimated optimized monthly infrastructure cost in USD by namespace.",
				},
				[]string{
					"namespace",
				},
			),

			potentialSavings: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_potential_savings_usd",

					Help: "Estimated potential monthly savings in USD by namespace.",
				},
				[]string{
					"namespace",
				},
			),

			savingsPercentage: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_savings_percentage",

					Help: "Estimated percentage of monthly cost that could be saved by namespace.",
				},
				[]string{
					"namespace",
				},
			),

			workloadCount: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_workload_count",

					Help: "Number of analyzed workloads by namespace.",
				},
				[]string{
					"namespace",
				},
			),

			optimizableWorkloadCount: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_optimizable_workload_count",

					Help: "Number of optimizable workloads by namespace.",
				},
				[]string{
					"namespace",
				},
			),

			workloadCurrentMonthlyCost: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_workload_current_monthly_cost_usd",

					Help: "Current estimated monthly infrastructure cost in USD by workload.",
				},
				[]string{
					"namespace",
					"workload",
				},
			),

			workloadOptimizedMonthlyCost: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_workload_optimized_monthly_cost_usd",

					Help: "Estimated optimized monthly infrastructure cost in USD by workload.",
				},
				[]string{
					"namespace",
					"workload",
				},
			),

			workloadPotentialSavings: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_workload_potential_savings_usd",

					Help: "Estimated potential monthly savings in USD by workload.",
				},
				[]string{
					"namespace",
					"workload",
				},
			),

			workloadSavingsPercentage: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_workload_savings_percentage",

					Help: "Estimated percentage of monthly cost that could be saved by workload.",
				},
				[]string{
					"namespace",
					"workload",
				},
			),

			workloadOptimizable: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_workload_optimizable",

					Help: "Whether a workload has one or more optimization recommendations.",
				},
				[]string{
					"namespace",
					"workload",
				},
			),

			recommendationCount: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_recommendation_count",

					Help: "Current number of optimization recommendations.",
				},
				[]string{},
			),

			recommendationWorkloadCount: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_recommendation_workload_count",

					Help: "Current number of workloads with one or more optimization recommendations.",
				},
				[]string{},
			),

			recommendationByRule: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_recommendation_rule_count",

					Help: "Current number of recommendations grouped by optimization rule.",
				},
				[]string{
					"rule",
				},
			),

			recommendationBySeverity: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_recommendation_severity_count",

					Help: "Current number of recommendations grouped by severity.",
				},
				[]string{
					"severity",
				},
			),

			recommendationByConfidence: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_recommendation_confidence_count",

					Help: "Current number of recommendations grouped by confidence.",
				},
				[]string{
					"confidence",
				},
			),

			schedulerRunsTotal: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "cee_scheduler_runs_total",

					Help: "Total number of scheduled analysis executions.",
				},
				[]string{
					"namespace",
				},
			),

			schedulerSuccessTotal: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "cee_scheduler_success_total",

					Help: "Total number of successful scheduled analyses.",
				},
				[]string{
					"namespace",
				},
			),

			schedulerFailureTotal: prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "cee_scheduler_failure_total",

					Help: "Total number of failed scheduled analyses.",
				},
				[]string{
					"namespace",
				},
			),

			schedulerLastSuccessTimestamp: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_scheduler_last_success_timestamp",

					Help: "Unix timestamp of the latest successful scheduled analysis.",
				},
				[]string{
					"namespace",
				},
			),

			schedulerLastFailureTimestamp: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_scheduler_last_failure_timestamp",

					Help: "Unix timestamp of the latest failed scheduled analysis.",
				},
				[]string{
					"namespace",
				},
			),

			schedulerLastDurationSeconds: prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "cee_scheduler_last_duration_seconds",

					Help: "Duration in seconds of the latest scheduled analysis.",
				},
				[]string{
					"namespace",
				},
			),

			activeNamespaces: make(
				map[string]struct{},
			),

			activeWorkloads: make(
				map[workloadKey]struct{},
			),

			activeRecommendationRules: make(
				map[recommendationRuleKey]struct{},
			),

			activeRecommendationSeverities: make(
				map[recommendationSeverityKey]struct{},
			),

			activeRecommendationConfidences: make(
				map[recommendationConfidenceKey]struct{},
			),
		}

	metrics.registry.MustRegister(
		metrics.currentMonthlyCost,
	)

	metrics.registry.MustRegister(
		metrics.optimizedMonthlyCost,
	)

	metrics.registry.MustRegister(
		metrics.potentialSavings,
	)

	metrics.registry.MustRegister(
		metrics.savingsPercentage,
	)

	metrics.registry.MustRegister(
		metrics.workloadCount,
	)

	metrics.registry.MustRegister(
		metrics.optimizableWorkloadCount,
	)

	metrics.registry.MustRegister(
		metrics.workloadCurrentMonthlyCost,
	)

	metrics.registry.MustRegister(
		metrics.workloadOptimizedMonthlyCost,
	)

	metrics.registry.MustRegister(
		metrics.workloadPotentialSavings,
	)

	metrics.registry.MustRegister(
		metrics.workloadSavingsPercentage,
	)

	metrics.registry.MustRegister(
		metrics.workloadOptimizable,
	)

	metrics.registry.MustRegister(
		metrics.recommendationCount,
	)

	metrics.registry.MustRegister(
		metrics.recommendationWorkloadCount,
	)

	metrics.registry.MustRegister(
		metrics.recommendationByRule,
	)

	metrics.registry.MustRegister(
		metrics.recommendationBySeverity,
	)

	metrics.registry.MustRegister(
		metrics.recommendationByConfidence,
	)

	metrics.registry.MustRegister(
		metrics.schedulerRunsTotal,
	)

	metrics.registry.MustRegister(
		metrics.schedulerSuccessTotal,
	)

	metrics.registry.MustRegister(
		metrics.schedulerFailureTotal,
	)

	metrics.registry.MustRegister(
		metrics.schedulerLastSuccessTimestamp,
	)

	metrics.registry.MustRegister(
		metrics.schedulerLastFailureTimestamp,
	)

	metrics.registry.MustRegister(
		metrics.schedulerLastDurationSeconds,
	)

	return metrics
}

func (m *Metrics) Update(
	report *analysis.AnalysisReport,
) {

	if report == nil {
		return
	}

	m.updateMutex.Lock()
	defer m.updateMutex.Unlock()

	currentNamespaces :=
		make(
			map[string]struct{},
			len(
				report.NamespaceBreakdown,
			),
		)

	currentWorkloads :=
		make(
			map[workloadKey]struct{},
			len(
				report.Workloads,
			),
		)

	currentRecommendationRules :=
		make(
			map[recommendationRuleKey]struct{},
		)

	currentRecommendationSeverities :=
		make(
			map[recommendationSeverityKey]struct{},
		)

	currentRecommendationConfidences :=
		make(
			map[recommendationConfidenceKey]struct{},
		)

	totalRecommendations := 0

	workloadsWithRecommendations := 0

	recommendationsByRule :=
		make(
			map[string]int,
		)

	recommendationsBySeverity :=
		make(
			map[string]int,
		)

	recommendationsByConfidence :=
		make(
			map[string]int,
		)

	for _, breakdown := range report.NamespaceBreakdown {

		namespace :=
			breakdown.Namespace

		if namespace == "" {
			namespace = "unknown"
		}

		currentNamespaces[namespace] = struct{}{}

		m.currentMonthlyCost.
			WithLabelValues(
				namespace,
			).
			Set(
				breakdown.CurrentMonthlyCostUSD,
			)

		m.optimizedMonthlyCost.
			WithLabelValues(
				namespace,
			).
			Set(
				breakdown.OptimizedMonthlyCostUSD,
			)

		m.potentialSavings.
			WithLabelValues(
				namespace,
			).
			Set(
				breakdown.PotentialSavingsUSD,
			)

		m.savingsPercentage.
			WithLabelValues(
				namespace,
			).
			Set(
				breakdown.SavingsPercentage,
			)

		m.workloadCount.
			WithLabelValues(
				namespace,
			).
			Set(
				float64(
					breakdown.WorkloadCount,
				),
			)

		m.optimizableWorkloadCount.
			WithLabelValues(
				namespace,
			).
			Set(
				float64(
					breakdown.OptimizableWorkloads,
				),
			)
	}

	for _, workload := range report.Workloads {

		namespace :=
			workload.Workload.Namespace

		if namespace == "" {
			namespace = "unknown"
		}

		name :=
			workload.Workload.Name

		if name == "" {
			name = "unknown"
		}

		key :=
			workloadKey{
				namespace: namespace,

				workload: name,
			}

		currentWorkloads[key] =
			struct{}{}

		var currentCost float64

		var optimizedCost float64

		var potentialSavings float64

		var savingsPercentage float64

		if workload.Cost != nil {

			currentCost =
				workload.Cost.
					CurrentMonthlyCostUSD

			optimizedCost =
				workload.Cost.
					OptimizedMonthlyCostUSD

			potentialSavings =
				workload.Cost.
					PotentialSavingsUSD

			savingsPercentage =
				workload.Cost.
					SavingsPercentage
		}

		optimizable :=
			float64(0)

		if len(
			workload.Recommendations,
		) > 0 {

			optimizable = 1

			workloadsWithRecommendations++
		}

		m.workloadCurrentMonthlyCost.
			WithLabelValues(
				namespace,
				name,
			).
			Set(
				currentCost,
			)

		m.workloadOptimizedMonthlyCost.
			WithLabelValues(
				namespace,
				name,
			).
			Set(
				optimizedCost,
			)

		m.workloadPotentialSavings.
			WithLabelValues(
				namespace,
				name,
			).
			Set(
				potentialSavings,
			)

		m.workloadSavingsPercentage.
			WithLabelValues(
				namespace,
				name,
			).
			Set(
				savingsPercentage,
			)

		m.workloadOptimizable.
			WithLabelValues(
				namespace,
				name,
			).
			Set(
				optimizable,
			)

		for _, recommendation := range workload.Recommendations {

			totalRecommendations++

			rule :=
				recommendation.Rule

			if rule == "" {
				rule = "unknown"
			}

			severity :=
				string(
					recommendation.Severity,
				)

			if severity == "" {
				severity = "UNKNOWN"
			}

			confidence :=
				string(
					recommendation.Confidence,
				)

			if confidence == "" {
				confidence = "UNKNOWN"
			}

			recommendationsByRule[rule]++

			recommendationsBySeverity[severity]++

			recommendationsByConfidence[confidence]++

			currentRecommendationRules[recommendationRuleKey{
				rule: rule,
			}] = struct{}{}

			currentRecommendationSeverities[recommendationSeverityKey{
				severity: severity,
			}] = struct{}{}

			currentRecommendationConfidences[recommendationConfidenceKey{
				confidence: confidence,
			}] = struct{}{}
		}
	}

	m.recommendationCount.
		WithLabelValues().
		Set(
			float64(
				totalRecommendations,
			),
		)

	m.recommendationWorkloadCount.
		WithLabelValues().
		Set(
			float64(
				workloadsWithRecommendations,
			),
		)

	for rule, count := range recommendationsByRule {

		m.recommendationByRule.
			WithLabelValues(
				rule,
			).
			Set(
				float64(count),
			)
	}

	for severity, count := range recommendationsBySeverity {

		m.recommendationBySeverity.
			WithLabelValues(
				severity,
			).
			Set(
				float64(count),
			)
	}

	for confidence, count := range recommendationsByConfidence {

		m.recommendationByConfidence.
			WithLabelValues(
				confidence,
			).
			Set(
				float64(count),
			)
	}

	for rule := range m.activeRecommendationRules {

		if _, exists :=
			currentRecommendationRules[rule]; exists {

			continue
		}

		m.recommendationByRule.
			DeleteLabelValues(
				rule.rule,
			)
	}

	for severity := range m.activeRecommendationSeverities {

		if _, exists :=
			currentRecommendationSeverities[severity]; exists {

			continue
		}

		m.recommendationBySeverity.
			DeleteLabelValues(
				severity.severity,
			)
	}

	for confidence := range m.activeRecommendationConfidences {

		if _, exists :=
			currentRecommendationConfidences[confidence]; exists {

			continue
		}

		m.recommendationByConfidence.
			DeleteLabelValues(
				confidence.confidence,
			)
	}

	for namespace := range m.activeNamespaces {

		if _, exists :=
			currentNamespaces[namespace]; exists {

			continue
		}

		m.currentMonthlyCost.
			DeleteLabelValues(
				namespace,
			)

		m.optimizedMonthlyCost.
			DeleteLabelValues(
				namespace,
			)

		m.potentialSavings.
			DeleteLabelValues(
				namespace,
			)

		m.savingsPercentage.
			DeleteLabelValues(
				namespace,
			)

		m.workloadCount.
			DeleteLabelValues(
				namespace,
			)

		m.optimizableWorkloadCount.
			DeleteLabelValues(
				namespace,
			)
	}

	for workload := range m.activeWorkloads {

		if _, exists :=
			currentWorkloads[workload]; exists {

			continue
		}

		m.workloadCurrentMonthlyCost.
			DeleteLabelValues(
				workload.namespace,
				workload.workload,
			)

		m.workloadOptimizedMonthlyCost.
			DeleteLabelValues(
				workload.namespace,
				workload.workload,
			)

		m.workloadPotentialSavings.
			DeleteLabelValues(
				workload.namespace,
				workload.workload,
			)

		m.workloadSavingsPercentage.
			DeleteLabelValues(
				workload.namespace,
				workload.workload,
			)

		m.workloadOptimizable.
			DeleteLabelValues(
				workload.namespace,
				workload.workload,
			)
	}

	m.activeNamespaces =
		currentNamespaces

	m.activeWorkloads =
		currentWorkloads

	m.activeRecommendationRules =
		currentRecommendationRules

	m.activeRecommendationSeverities =
		currentRecommendationSeverities

	m.activeRecommendationConfidences =
		currentRecommendationConfidences
}

func (m *Metrics) RecordSchedulerSuccess(
	namespace string,
	startedAt time.Time,
) {

	m.schedulerMutex.Lock()
	defer m.schedulerMutex.Unlock()

	now :=
		time.Now().UTC()

	m.schedulerRunsTotal.
		WithLabelValues(
			namespace,
		).
		Inc()

	m.schedulerSuccessTotal.
		WithLabelValues(
			namespace,
		).
		Inc()

	m.schedulerLastSuccessTimestamp.
		WithLabelValues(
			namespace,
		).
		Set(
			float64(
				now.Unix(),
			),
		)

	m.schedulerLastDurationSeconds.
		WithLabelValues(
			namespace,
		).
		Set(
			now.Sub(
				startedAt,
			).Seconds(),
		)
}

func (m *Metrics) RecordSchedulerFailure(
	namespace string,
	startedAt time.Time,
) {

	m.schedulerMutex.Lock()
	defer m.schedulerMutex.Unlock()

	now :=
		time.Now().UTC()

	m.schedulerRunsTotal.
		WithLabelValues(
			namespace,
		).
		Inc()

	m.schedulerFailureTotal.
		WithLabelValues(
			namespace,
		).
		Inc()

	m.schedulerLastFailureTimestamp.
		WithLabelValues(
			namespace,
		).
		Set(
			float64(
				now.Unix(),
			),
		)

	m.schedulerLastDurationSeconds.
		WithLabelValues(
			namespace,
		).
		Set(
			now.Sub(
				startedAt,
			).Seconds(),
		)
}

func (m *Metrics) Handler() http.Handler {

	return promhttp.HandlerFor(
		m.registry,
		promhttp.HandlerOpts{},
	)
}
