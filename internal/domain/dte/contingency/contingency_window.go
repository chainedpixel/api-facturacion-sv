package contingency

import "time"

const (
	// TransmissionWindow is the time allowed to transmit contingency documents after the contingency
	// event obtains its reception stamp.
	TransmissionWindow = 72 * time.Hour
	// TechnicalReportThreshold is the contingency duration after which a technical report is required.
	TechnicalReportThreshold = 72 * time.Hour
	// ContingencyExpiredCode identifies expired contingency documents in administrator alerts.
	ContingencyExpiredCode = "CONTINGENCY_72H_EXPIRED"
)

// IsContingencyWindowExpired reports whether the transmission window elapsed since the event stamp.
// Documents whose event has no stamp yet never expire.
func IsContingencyWindowExpired(eventStampedAt *time.Time, now time.Time) bool {
	return eventStampedAt != nil && now.Sub(*eventStampedAt) > TransmissionWindow
}
