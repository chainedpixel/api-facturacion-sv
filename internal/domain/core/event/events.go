package event

import (
	"fmt"
	"time"
)

const (
	EventContingencyActivated    = "contingency.activated"
	EventEmissionFailure         = "emission.failure"
	EventRetransmissionJobFailed = "retransmission_job.failed"
)

type RejectedDocSummary struct {
	ControlNumber string
	ErrorCode     string
	Description   string
	Observations  []string
}

type ContingencyActivatedEvent struct {
	BranchID        uint
	BranchAddress   string
	NIT             string
	ClientName      string
	ContingencyType string
	Reason          string
	AffectedDocs    int
	OccurredAtTime  time.Time
}

func (e ContingencyActivatedEvent) Name() string          { return EventContingencyActivated }
func (e ContingencyActivatedEvent) OccurredAt() time.Time { return e.OccurredAtTime }
func (e ContingencyActivatedEvent) AggregateID() string   { return fmt.Sprintf("branch:%d", e.BranchID) }
func (e ContingencyActivatedEvent) Payload() map[string]any {
	return map[string]any{
		"branch_id":        e.BranchID,
		"branch_address":   e.BranchAddress,
		"nit":              e.NIT,
		"client_name":      e.ClientName,
		"contingency_type": e.ContingencyType,
		"reason":           e.Reason,
		"affected_docs":    e.AffectedDocs,
	}
}

type EmissionFailureEvent struct {
	BranchID             uint
	BranchAddress        string
	NIT                  string
	ClientName           string
	ControlNumber        string
	GenerationCode       string
	DTEType              string
	ErrorCode            string
	LastError            string
	HaciendaObservations []string
	Attempts             int
	OccurredAtTime       time.Time
}

func (e EmissionFailureEvent) Name() string          { return EventEmissionFailure }
func (e EmissionFailureEvent) OccurredAt() time.Time { return e.OccurredAtTime }
func (e EmissionFailureEvent) AggregateID() string {
	if e.ControlNumber != "" {
		return fmt.Sprintf("dte:%s", e.ControlNumber)
	}
	return fmt.Sprintf("dte:%s", e.GenerationCode)
}
func (e EmissionFailureEvent) Payload() map[string]any {
	return map[string]any{
		"branch_id":             e.BranchID,
		"branch_address":        e.BranchAddress,
		"nit":                   e.NIT,
		"client_name":           e.ClientName,
		"control_number":        e.ControlNumber,
		"generation_code":       e.GenerationCode,
		"dte_type":              e.DTEType,
		"error_code":            e.ErrorCode,
		"last_error":            e.LastError,
		"hacienda_observations": e.HaciendaObservations,
		"attempts":              e.Attempts,
	}
}

type RetransmissionJobFailedEvent struct {
	JobName         string
	FailedDocuments []RejectedDocSummary
	LastError       string
	OccurredAtTime  time.Time
}

func (e RetransmissionJobFailedEvent) Name() string          { return EventRetransmissionJobFailed }
func (e RetransmissionJobFailedEvent) OccurredAt() time.Time { return e.OccurredAtTime }
func (e RetransmissionJobFailedEvent) AggregateID() string   { return fmt.Sprintf("job:%s", e.JobName) }
func (e RetransmissionJobFailedEvent) Payload() map[string]any {
	return map[string]any{
		"job_name":         e.JobName,
		"failed_documents": e.FailedDocuments,
		"last_error":       e.LastError,
	}
}
