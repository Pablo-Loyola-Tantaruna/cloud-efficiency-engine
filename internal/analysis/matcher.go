package analysis

import (
	"fmt"

	"cloud-efficiency-engine/internal/domain"
)

func workloadKey(
	namespace string,
	name string,
) string {

	return namespace + "/" + name
}

func MatchHistory(
	workload domain.WorkloadMetrics,
	histories []domain.WorkloadHistory,
) (*domain.WorkloadHistory, error) {

	key := workloadKey(
		workload.Namespace,
		workload.Name,
	)

	for index := range histories {

		history := &histories[index]

		if workloadKey(
			history.Namespace,
			history.Name,
		) == key {

			return history, nil
		}
	}

	return nil, fmt.Errorf(
		"history not found for workload %s",
		key,
	)
}
