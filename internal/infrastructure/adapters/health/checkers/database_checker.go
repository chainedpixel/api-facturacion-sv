package checkers

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/health"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/dimiro1/health/db"
	"gorm.io/gorm"
)

type databaseChecker struct {
	db *gorm.DB
}

func NewDatabaseChecker(db *gorm.DB) health.ComponentChecker {
	return &databaseChecker{db: db}
}

func (c *databaseChecker) Name() string {
	return "database"
}

func (c *databaseChecker) Check() models.Health {
	sql, err := c.db.DB()
	if err != nil {
		return models.Health{
			Status:  constants.StatusDown,
			Details: utils.TranslateHealthError("FailedToGetDBConnection"),
		}
	}

	checker := db.NewMySQLChecker(sql)
	health := checker.Check()

	if health.IsDown() {
		details := utils.TranslateHealthDown(c.Name())

		if health.GetInfo("error") != nil {
			details = fmt.Sprintf("%s: %v", details, health.GetInfo("error"))
		}

		return models.Health{
			Status:  constants.StatusDown,
			Details: details,
		}
	}

	return models.Health{
		Status:  constants.StatusUp,
		Details: utils.TranslateHealthUp(c.Name()),
	}

}
