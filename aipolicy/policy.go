package aipolicy

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

// TimeWindow defines the allowed hours window within a given timezone.
type TimeWindow struct {
	Timezone  string `json:"timezone"`
	StartHour int    `json:"start_hour"`
	EndHour   int    `json:"end_hour"`
}

// Conditions holds all optional policy conditions evaluated server-side.
type Conditions struct {
	TimeWindow *TimeWindow `json:"time_window,omitempty"`
}

// Policy maps to the policies table.
type Policy struct {
	ID                    uuid.UUID  `json:"id"`
	PolicyID              string     `json:"policy_id"`
	Name                  string     `json:"name"`
	RemoteMCPService      string     `json:"remote_mcp_service"`
	ResourceAccessRequest string     `json:"resource_access_request"`
	Environment           string     `json:"environment"`
	Enabled               bool       `json:"enabled"`
	Priority              int        `json:"priority"`
	Conditions            Conditions `json:"conditions"`
	Description           string     `json:"description"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// DecideRequest is the payload for POST /decide.
type DecideRequest struct {
	RemoteMCPService      string `json:"remote_mcp_service"`
	ResourceAccessRequest string `json:"resource_access_request"`
	Environment           string `json:"environment"`
}

// DecideResponse is returned by POST /decide.
type DecideResponse struct {
	Allowed bool `json:"allowed"`
}

// evaluateConditions returns true if all conditions in c pass.
// A policy with no conditions is unconditionally allowed.
func evaluateConditions(c Conditions) bool {
	if c.TimeWindow != nil {
		return evaluateTimeWindow(c.TimeWindow)
	}
	return true
}

func evaluateTimeWindow(tw *TimeWindow) bool {
	loc, err := time.LoadLocation(tw.Timezone)
	if err != nil {
		return false
	}
	hour := time.Now().In(loc).Hour()
	return hour >= tw.StartHour && hour < tw.EndHour
}
