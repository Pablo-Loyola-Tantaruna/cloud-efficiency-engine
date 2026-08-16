package domain

import (
	"fmt"
	"strings"
	"time"
)

type ActionApproval struct {
	PlanID     string    `json:"planId"`
	ApprovedBy string    `json:"approvedBy"`
	ApprovedAt time.Time `json:"approvedAt"`
	Comment    string    `json:"comment,omitempty"`
}

func (a ActionApproval) Validate() error {
	if strings.TrimSpace(a.PlanID) == "" {
		return fmt.Errorf("approval plan id must not be empty")
	}
	if strings.TrimSpace(a.ApprovedBy) == "" {
		return fmt.Errorf("approval user must not be empty")
	}
	if a.ApprovedAt.IsZero() {
		return fmt.Errorf("approval timestamp must not be zero")
	}
	return nil
}
