package checkers

import (
	"encoding/json"
	"fmt"
	"github.com/MarlonG1/api-facturacion-sv/config"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/health"
	"github.com/MarlonG1/api-facturacion-sv/pkg/shared/utils"
	"net/http"
	"strings"
	"time"

	"github.com/MarlonG1/api-facturacion-sv/internal/domain/health/constants"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/health/models"
)

type signerChecker struct {
	client *http.Client
}

type StandardHealthResponse struct {
	Status string `json:"status"`
}

type NestedHealthResponse struct {
	Status string `json:"status"`
	Body   struct {
		Status string `json:"status"`
	} `json:"body"`
}

func NewSignerChecker() health.ComponentChecker {
	return &signerChecker{
		client: &http.Client{Timeout: 2 * time.Second},
	}
}

func (c *signerChecker) Name() string {
	return "dte_signer"
}

func (c *signerChecker) Check() models.Health {
	resp, err := c.client.Get(config.Signer.Health)
	if err != nil {
		return models.Health{
			Status:  constants.StatusDown,
			Details: fmt.Sprintf("%s: %v", utils.TranslateHealthDown(c.Name()), err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.Health{
			Status:  constants.StatusDown,
			Details: fmt.Sprintf("%s: HTTP %d", utils.TranslateHealthDown(c.Name()), resp.StatusCode),
		}
	}

	var rawResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawResponse); err != nil {
		return models.Health{
			Status:  constants.StatusDown,
			Details: fmt.Sprintf("%s: %v", utils.TranslateHealthDown(c.Name()), err),
		}
	}

	if isHealthy := c.checkHealthStatus(rawResponse); isHealthy {
		return models.Health{
			Status:  constants.StatusUp,
			Details: utils.TranslateHealthUp(c.Name()),
		}
	}

	return models.Health{
		Status:  constants.StatusDown,
		Details: utils.TranslateHealthDown(c.Name()),
	}
}

func (c *signerChecker) checkHealthStatus(response map[string]interface{}) bool {
	if status, ok := response["status"].(string); ok {
		if strings.ToUpper(status) == "UP" {
			return true
		}
	}

	if body, ok := response["body"].(map[string]interface{}); ok {
		if bodyStatus, ok := body["status"].(string); ok {
			if strings.ToUpper(bodyStatus) == "UP" {
				return true
			}
		}
	}

	return false
}
