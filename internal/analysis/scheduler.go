package analysis

import (
	"context"
	"log/slog"
	"time"
)

type Analyzer interface {
	Analyze(
		ctx context.Context,
		options AnalysisOptions,
	) (*AnalysisReport, error)
}

type MetricsSink interface {
	Update(
		report *AnalysisReport,
	)
}

type SchedulerTelemetry interface {
	RecordSchedulerSuccess(
		namespace string,
		startedAt time.Time,
	)

	RecordSchedulerFailure(
		namespace string,
		startedAt time.Time,
	)
}

type SchedulerConfig struct {
	Namespace     string
	Interval      time.Duration
	LookbackHours int
	Step          time.Duration
}

type Scheduler struct {
	engine  Analyzer
	metrics MetricsSink
	logger  *slog.Logger
	config  SchedulerConfig
}

func NewScheduler(
	engine Analyzer,
	metrics MetricsSink,
	logger *slog.Logger,
	config SchedulerConfig,
) *Scheduler {

	return &Scheduler{
		engine:  engine,
		metrics: metrics,
		logger:  logger,
		config:  config,
	}
}

func (s *Scheduler) Run(
	ctx context.Context,
) {

	config :=
		s.normalizedConfig()

	s.runOnce(
		ctx,
		config,
	)

	ticker :=
		time.NewTicker(
			config.Interval,
		)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			s.runOnce(
				ctx,
				config,
			)
		}
	}
}

func (s *Scheduler) normalizedConfig() SchedulerConfig {

	config :=
		s.config

	if config.Interval <= 0 {

		config.Interval =
			5 * time.Minute
	}

	if config.LookbackHours <= 0 {

		config.LookbackHours =
			24
	}

	if config.Step <= 0 {

		config.Step =
			5 * time.Minute
	}

	return config
}

func (s *Scheduler) runOnce(
	ctx context.Context,
	config SchedulerConfig,
) {

	startedAt :=
		time.Now().UTC()

	end :=
		startedAt

	start :=
		end.Add(
			-time.Duration(
				config.LookbackHours,
			) * time.Hour,
		)

	report, err :=
		s.engine.Analyze(
			ctx,
			AnalysisOptions{
				Namespace: config.Namespace,

				Start: start,

				End: end,

				Step: config.Step,
			},
		)

	if err != nil {

		s.recordSchedulerFailure(
			config.Namespace,
			startedAt,
		)

		if s.logger != nil {

			s.logger.Error(
				"scheduled_analysis_failed",

				"namespace",
				config.Namespace,

				"error",
				err,

				"duration_ms",
				time.Since(
					startedAt,
				).Milliseconds(),
			)
		}

		return
	}

	if report == nil {

		s.recordSchedulerFailure(
			config.Namespace,
			startedAt,
		)

		if s.logger != nil {

			s.logger.Error(
				"scheduled_analysis_failed",

				"namespace",
				config.Namespace,

				"error",
				"analysis returned nil report",

				"duration_ms",
				time.Since(
					startedAt,
				).Milliseconds(),
			)
		}

		return
	}

	if s.metrics != nil {

		s.metrics.Update(
			report,
		)
	}

	s.recordSchedulerSuccess(
		config.Namespace,
		startedAt,
	)

	if s.logger != nil {

		s.logger.Info(
			"scheduled_analysis_completed",

			"namespace",
			config.Namespace,

			"workloads",
			report.Summary.TotalWorkloads,

			"optimizable_workloads",
			report.Summary.OptimizableWorkloads,

			"current_monthly_cost_usd",
			report.Summary.CurrentMonthlyCostUSD,

			"optimized_monthly_cost_usd",
			report.Summary.OptimizedMonthlyCostUSD,

			"potential_savings_usd",
			report.Summary.PotentialSavingsUSD,

			"savings_percentage",
			report.Summary.SavingsPercentage,

			"duration_ms",
			time.Since(
				startedAt,
			).Milliseconds(),
		)
	}
}

func (s *Scheduler) recordSchedulerSuccess(
	namespace string,
	startedAt time.Time,
) {

	telemetry, ok :=
		s.metrics.(SchedulerTelemetry)

	if !ok {
		return
	}

	telemetry.RecordSchedulerSuccess(
		namespace,
		startedAt,
	)
}

func (s *Scheduler) recordSchedulerFailure(
	namespace string,
	startedAt time.Time,
) {

	telemetry, ok :=
		s.metrics.(SchedulerTelemetry)

	if !ok {
		return
	}

	telemetry.RecordSchedulerFailure(
		namespace,
		startedAt,
	)
}
