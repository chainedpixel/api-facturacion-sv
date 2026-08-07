package checkers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	health2 "github.com/chainedpixel/ordo-factus/internal/domain/health"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/dimiro1/health"
)

// haciendaChecker implements both the ports.ComponentChecker and health.Checker interfaces
type haciendaChecker struct {
	client *http.Client
}

func NewHaciendaChecker() health2.ComponentChecker {
	return &haciendaChecker{
		client: &http.Client{Timeout: 2 * time.Second},
	}
}

func (c *haciendaChecker) Name() string {
	return "hacienda"
}

// Check implements ports.ComponentChecker.Check
func (c *haciendaChecker) Check() models.Health {
	health := c.checkHealth()

	status := constants.StatusUp
	details := utils.TranslateHealthUp(c.Name())

	if health.IsDown() {
		status = constants.StatusDown
		details = utils.TranslateHealthDown(c.Name())

		if health.GetInfo("error") != nil {
			details = fmt.Sprintf("%s: %v", details, health.GetInfo("error"))
		}
	}

	return models.Health{
		Status:  status,
		Details: details,
	}
}

func (c *haciendaChecker) checkHealth() health.Health {
	result := health.NewHealth()

	endpoints := map[string]string{
		"signing":     config.MHPaths.AuthURL,
		"reception":   config.MHPaths.ReceptionURL,
		"contingency": config.MHPaths.ContingencyURL,
	}

	for name, url := range endpoints {
		if err := c.checkEndpoint(url); err != nil {
			logs.Error(fmt.Sprintf("Hacienda %s endpoint unavailable", name), map[string]interface{}{
				"error": err.Error(),
				"url":   url,
			})

			result.Down()

			if strings.Contains(err.Error(), "dial tcp") {
				result.AddInfo("error", utils.TranslateHealthError("NotInternet", name, err.Error()))
				return result
			}

			result.AddInfo("error", utils.TranslateHealthError("HaciendaEndpointUnavailable", name, err.Error()))
			return result
		}
	}

	if err := c.checkAuthProcessing(); err != nil {
		logs.Error("Hacienda signing processing check failed", map[string]interface{}{
			"error": err.Error(),
		})

		result.Down()
		result.AddInfo("error", utils.TranslateHealthError("HaciendaEndpointAuthFailed", err.Error()))
		return result
	}

	result.Up()
	return result
}

// Keep the original helper methods unchanged
func (c *haciendaChecker) checkEndpoint(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logs.Error("Failed to close response body", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}(resp.Body)

	if resp.StatusCode >= 500 {
		return fmt.Errorf("%s", utils.TranslateHealthError("HaciendaServiceUnavailable", resp.StatusCode))
	}

	return nil
}

func (c *haciendaChecker) checkAuthProcessing() error {
	dummyAuth := struct {
		User string `json:"user"`
		Pwd  string `json:"pwd"`
	}{
		User: "test_user",
		Pwd:  "test_password",
	}

	jsonData, err := json.Marshal(dummyAuth)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx,
		"POST",
		config.MHPaths.AuthURL,
		bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logs.Error("Failed to close response body", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusUnauthorized &&
		resp.StatusCode != http.StatusBadRequest &&
		resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", utils.TranslateHealthError("UnexpectedHaciendaServiceResponse", resp.StatusCode))
	}

	return nil
}
