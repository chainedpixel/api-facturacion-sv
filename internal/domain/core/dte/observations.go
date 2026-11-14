package dte

import "encoding/json"

// ObservationsFromHaciendaResponse returns the observations of a stored Hacienda response, or an empty list when the
// response has none or cannot be read.
func ObservationsFromHaciendaResponse(raw string) []string {
	observations := []string{}
	if raw == "" {
		return observations
	}

	var response struct {
		Observations []string `json:"observaciones"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		return observations
	}

	for _, observation := range response.Observations {
		if observation != "" {
			observations = append(observations, observation)
		}
	}
	return observations
}
