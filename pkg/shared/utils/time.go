package utils

import (
	"encoding/json"
	"time"
)

var timezone *time.Location

func TimeInit() error {
	var err error
	timezone, err = time.LoadLocation("America/El_Salvador")
	if err != nil {
		return err
	}

	return nil
}

// TimeNow returns the current time in the configured timezone
func TimeNow() time.Time {
	return time.Now().In(timezone)
}

var processingDateLayouts = []string{"02/01/2006 15:04:05", "2006-01-02 15:04:05", time.RFC3339}

// ParseProcessingDate parses the processing date returned by Hacienda in the configured timezone. It returns
// false when the value is empty or has an unknown layout.
func ParseProcessingDate(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range processingDateLayouts {
		if parsed, err := time.ParseInLocation(layout, value, TimeNow().Location()); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

// ExtractGenerationDateTime returns the generation date and time of a DTE JSON in the configured timezone.
// It returns false when the JSON has no generation date.
func ExtractGenerationDateTime(jsonData string) (time.Time, bool) {
	var dte struct {
		Identification struct {
			EmissionDate string `json:"fecEmi"`
			EmissionTime string `json:"horEmi"`
		} `json:"identificacion"`
	}
	if err := json.Unmarshal([]byte(jsonData), &dte); err != nil || dte.Identification.EmissionDate == "" {
		return time.Time{}, false
	}

	emissionTime := dte.Identification.EmissionTime
	if emissionTime == "" {
		emissionTime = "00:00:00"
	}
	generation, err := time.ParseInLocation("2006-01-02 15:04:05", dte.Identification.EmissionDate+" "+emissionTime, TimeNow().Location())
	if err != nil {
		return time.Time{}, false
	}
	return generation, true
}

// GenerationDateOrDefault returns the generation date of a DTE JSON or the fallback when it is missing.
func GenerationDateOrDefault(jsonData string, fallback time.Time) time.Time {
	if generation, ok := ExtractGenerationDateTime(jsonData); ok {
		return generation
	}
	return fallback
}
