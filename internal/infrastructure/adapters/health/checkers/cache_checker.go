package checkers

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/health"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/dimiro1/health/redis"
)

type redisChecker struct {
}

func NewRedisChecker() health.ComponentChecker {
	return &redisChecker{}
}

func (c *redisChecker) Name() string {
	return "redis"
}

func (c *redisChecker) Check() models.Health {
	checker := redis.NewChecker("tcp", ":6379")
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
