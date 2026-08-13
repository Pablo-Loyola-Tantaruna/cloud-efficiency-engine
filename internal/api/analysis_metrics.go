package api

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

type AnalysisMetrics struct {
	mu sync.RWMutex

	totalAnalyses        uint64
	failedAnalyses       uint64
	workloadsAnalyzed    uint64
	optimizableWorkloads uint64

	currentMonthlyCostUSD   float64
	optimizedMonthlyCostUSD float64
	potentialSavingsUSD     float64
	savingsPercentage       float64
}

func NewAnalysisMetrics() *AnalysisMetrics {
	return &AnalysisMetrics{}
}

func (m *AnalysisMetrics) RecordAnalysis(
	workloads int,
	optimizableWorkloads int,
	currentMonthlyCostUSD float64,
	optimizedMonthlyCostUSD float64,
	potentialSavingsUSD float64,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalAnalyses++

	m.workloadsAnalyzed +=
		uint64(workloads)

	m.optimizableWorkloads +=
		uint64(optimizableWorkloads)

	m.currentMonthlyCostUSD =
		currentMonthlyCostUSD

	m.optimizedMonthlyCostUSD =
		optimizedMonthlyCostUSD

	m.potentialSavingsUSD =
		potentialSavingsUSD

	m.savingsPercentage = 0

	if currentMonthlyCostUSD > 0 {
		m.savingsPercentage =
			(potentialSavingsUSD /
				currentMonthlyCostUSD) *
				100
	}
}

func (m *AnalysisMetrics) RecordAnalysisError() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failedAnalyses++
}

func (m *AnalysisMetrics) WriteMetrics(
	w http.ResponseWriter,
) {
	m.mu.RLock()

	totalAnalyses :=
		m.totalAnalyses

	failedAnalyses :=
		m.failedAnalyses

	workloadsAnalyzed :=
		m.workloadsAnalyzed

	optimizableWorkloads :=
		m.optimizableWorkloads

	currentMonthlyCostUSD :=
		m.currentMonthlyCostUSD

	optimizedMonthlyCostUSD :=
		m.optimizedMonthlyCostUSD

	potentialSavingsUSD :=
		m.potentialSavingsUSD

	savingsPercentage :=
		m.savingsPercentage

	m.mu.RUnlock()

	_, _ =
		fmt.Fprint(
			w,

			"# HELP cee_analysis_total Total number of analysis executions.\n"+
				"# TYPE cee_analysis_total counter\n"+
				"cee_analysis_total "+
				strconv.FormatUint(
					totalAnalyses,
					10,
				)+
				"\n\n"+
				"# HELP cee_analysis_errors_total Total number of failed analysis executions.\n"+
				"# TYPE cee_analysis_errors_total counter\n"+
				"cee_analysis_errors_total "+
				strconv.FormatUint(
					failedAnalyses,
					10,
				)+
				"\n\n"+
				"# HELP cee_workloads_analyzed_total Total number of workloads analyzed.\n"+
				"# TYPE cee_workloads_analyzed_total counter\n"+
				"cee_workloads_analyzed_total "+
				strconv.FormatUint(
					workloadsAnalyzed,
					10,
				)+
				"\n\n"+
				"# HELP cee_optimizable_workloads_total Total number of workloads identified as optimizable.\n"+
				"# TYPE cee_optimizable_workloads_total counter\n"+
				"cee_optimizable_workloads_total "+
				strconv.FormatUint(
					optimizableWorkloads,
					10,
				)+
				"\n\n"+
				"# HELP cee_current_monthly_cost_usd Current estimated monthly workload cost in USD.\n"+
				"# TYPE cee_current_monthly_cost_usd gauge\n"+
				"cee_current_monthly_cost_usd "+
				strconv.FormatFloat(
					currentMonthlyCostUSD,
					'f',
					2,
					64,
				)+
				"\n\n"+
				"# HELP cee_optimized_monthly_cost_usd Current estimated optimized monthly workload cost in USD.\n"+
				"# TYPE cee_optimized_monthly_cost_usd gauge\n"+
				"cee_optimized_monthly_cost_usd "+
				strconv.FormatFloat(
					optimizedMonthlyCostUSD,
					'f',
					2,
					64,
				)+
				"\n\n"+
				"# HELP cee_potential_savings_usd Current estimated monthly savings in USD.\n"+
				"# TYPE cee_potential_savings_usd gauge\n"+
				"cee_potential_savings_usd "+
				strconv.FormatFloat(
					potentialSavingsUSD,
					'f',
					2,
					64,
				)+
				"\n\n"+
				"# HELP cee_savings_percentage Current estimated monthly savings percentage.\n"+
				"# TYPE cee_savings_percentage gauge\n"+
				"cee_savings_percentage "+
				strconv.FormatFloat(
					savingsPercentage,
					'f',
					2,
					64,
				)+
				"\n",
		)
}

func (m *AnalysisMetrics) Handler(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"text/plain; version=0.0.4; charset=utf-8",
	)

	w.WriteHeader(
		http.StatusOK,
	)

	m.WriteMetrics(
		w,
	)
}
