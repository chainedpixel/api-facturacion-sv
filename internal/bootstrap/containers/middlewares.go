package containers

import (
	"github.com/chainedpixel/ordo-factus/config/drivers"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/middleware"
)

type MiddlewareContainer struct {
	services   *ServicesContainer
	connection *drivers.DbConnection

	corsMid    *middleware.CorsMiddleware
	authMid    *middleware.AuthMiddleware
	tokenMid   *middleware.TokenExtractor
	errorMid   *middleware.ErrorMiddleware
	metricMid  *middleware.MetricsMiddleware
	timeoutMid *middleware.TimeoutMiddleware
	dbMid      *middleware.DBConnectionMiddleware
}

func NewMiddlewareContainer(services *ServicesContainer, connection *drivers.DbConnection) *MiddlewareContainer {
	return &MiddlewareContainer{
		services:   services,
		connection: connection,
	}
}

func (c *MiddlewareContainer) Initialize() {
	c.corsMid = middleware.NewCorsMiddleware(
		[]string{"*"},
		nil,
		nil,
	)
	c.tokenMid = middleware.NewTokenExtractor()
	c.errorMid = middleware.NewErrorMiddleware()
	c.authMid = middleware.NewAuthMiddleware(c.services.TokenManager())
	c.metricMid = middleware.NewMetricsMiddleware(c.services.CacheManager())
	c.dbMid = middleware.NewDBConnectionMiddleware(c.connection)
	c.timeoutMid = middleware.NewTimeoutMiddleware()
}

func (c *MiddlewareContainer) TimeoutMiddleware() *middleware.TimeoutMiddleware {
	return c.timeoutMid
}
func (c *MiddlewareContainer) DBConnectionMiddleware() *middleware.DBConnectionMiddleware {
	return c.dbMid
}

func (c *MiddlewareContainer) CorsMiddleware() *middleware.CorsMiddleware {
	return c.corsMid
}

func (c *MiddlewareContainer) AuthMiddleware() *middleware.AuthMiddleware {
	return c.authMid
}

func (c *MiddlewareContainer) TokenExtractor() *middleware.TokenExtractor {
	return c.tokenMid
}

func (c *MiddlewareContainer) ErrorMiddleware() *middleware.ErrorMiddleware {
	return c.errorMid
}

func (c *MiddlewareContainer) MetricsMiddleware() *middleware.MetricsMiddleware {
	return c.metricMid
}
