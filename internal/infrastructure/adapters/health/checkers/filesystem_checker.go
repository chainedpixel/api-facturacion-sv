package checkers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chainedpixel/ordo-factus/config"
	health2 "github.com/chainedpixel/ordo-factus/internal/domain/health"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/dimiro1/health"

	"github.com/chainedpixel/ordo-factus/internal/domain/health/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type fileSystemChecker struct {
	logPath string
}

func NewFileSystemChecker() health2.ComponentChecker {
	logFilePath := filepath.Join(utils.FindProjectRoot()+config.Log.Path, "dte_microservice.log")

	return &fileSystemChecker{
		logPath: logFilePath,
	}
}

func (c *fileSystemChecker) Name() string {
	return "filesystem"
}

// Check verifies whether the filesystem has write permissions
// by attempting to write to the log file.
// Returns a health status with the result of the check.
func (c *fileSystemChecker) Check() models.Health {
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

func (c *fileSystemChecker) checkHealth() health.Health {
	result := health.NewHealth()

	if err := c.checkFileSystem(); err != nil {
		result.Down()
		result.AddInfo("error", err.Error())
	}

	result.Up()
	return result
}

func (c *fileSystemChecker) checkFileSystem() error {
	dir := filepath.Dir(c.logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return errors.New(utils.TranslateHealthError("FailedToCreateLogDir"))
	}

	file, err := os.OpenFile(c.logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return errors.New(utils.TranslateHealthError("SystemDontHavePermissions"))
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logs.Error("Error closing file")
		}
	}(file)

	return nil
}
