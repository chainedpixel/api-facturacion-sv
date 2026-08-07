// bootstrap/app.go
package bootstrap

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/bootstrap/containers"

	"github.com/chainedpixel/ordo-factus/cmd/setup"
	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/config/drivers"
	errPackage "github.com/chainedpixel/ordo-factus/config/error"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/server"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// Application represents the complete application
type Application struct {
	server       *server.Server
	container    *containers.Container
	dbConnection *drivers.DbConnection
}

// SupportedDrivers holds the configuration for supported database drivers
var SupportedDrivers = map[string]drivers.DriverConfig{
	"mysql":    drivers.NewMysqlDriver(),
	"postgres": drivers.NewPostgresDriver(),
}

// NewApplication creates a new instance of the application
func NewApplication() *Application {
	return &Application{}
}

// Initialize initializes all application components
func (app *Application) Initialize() error {
	rootPath := utils.FindProjectRoot()

	err := config.InitEnvConfig(rootPath)
	if err != nil {
		return fmt.Errorf("error initializing environment configuration: %w", err)
	}

	err = logs.InitLogger(config.Log.Level, config.Log.Path)
	if err != nil {
		return fmt.Errorf("error initializing logger: %w", err)
	}
	logs.Info("Logger initialized successfully")

	err = utils.TimeInit()
	if err != nil {
		logs.Fatal("Failed to initialize global time", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("error initializing global time: %w", err)
	}

	langPath := fmt.Sprintf("%s/%s", rootPath, config.Server.AppLangPath)
	if err = config.InitTranslations(langPath, config.Server.AppLang); err != nil {
		log.Fatalf("Failed to initialize translation system: %v", err)
	}

	app.dbConnection, err = app.initDatabaseConfigurations()
	if err != nil {
		logs.Fatal("Failed to initialize database configurations", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("error initializing database configurations: %w", err)
	}

	app.container = containers.NewContainer(app.dbConnection)
	err = app.container.Initialize()
	if err != nil {
		logs.Error("Failed to initialize container", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("error initializing container: %w", err)
	}

	app.server = server.Initialize(app.container)

	err = setup.SetupJobs(
		app.container.Services().ContingencyManager(),
		app.container.Repositories().ReservedSequenceRepo(),
		config.Server.AmbientCode,
		app.dbConnection,
		app.container.Services().CacheManager(),
		app.container.Services().EventBus(),
	)
	if err != nil {
		logs.Error("Failed to setup jobs", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("error setting up jobs: %w", err)
	}

	return nil
}

// Start starts the application and handles signals for a graceful shutdown
func (app *Application) Start() error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)

	go func() {
		logs.Info("Server started successfully", map[string]interface{}{"port": config.Server.Port})
		serverErrors <- app.server.Start()
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-signals:
		logs.Info("Shutdown signal received", map[string]interface{}{"signal": sig.String()})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.server.Shutdown(ctx); err != nil {
			logs.Error("Server shutdown error", map[string]interface{}{"error": err.Error()})
			return fmt.Errorf("server shutdown error: %w", err)
		}

		if err := app.dbConnection.Close(); err != nil {
			logs.Error("Database connection close error", map[string]interface{}{"error": err.Error()})
			return fmt.Errorf("database connection close error: %w", err)
		}

		logs.Info("Shutdown completed", nil)
	}

	return nil
}

// initDatabaseConfigurations initializes the database configurations
func (app *Application) initDatabaseConfigurations() (*drivers.DbConnection, error) {
	driver := app.selectDatabaseDriver()
	if driver == nil {
		logs.Fatal("Invalid database driver", nil)
		return nil, errPackage.ErrUnrecognizedDriver
	}
	logs.Info("Database driver initialized successfully")

	dbConnection := drivers.NewDatabaseConnection(driver)
	if dbConnection.Err != nil {
		logs.Fatal("Failed to connect to the database", map[string]interface{}{"error": dbConnection.Err.Error()})
		return nil, dbConnection.Err
	}
	logs.Info("Database connection initialized successfully")

	if err := dbConnection.Open(); err != nil {
		logs.Fatal("Failed to open database connection", map[string]interface{}{"error": err.Error()})
		return nil, err
	}

	if config.Server.RunMigration {
		err := database.RunMigrations(dbConnection.Db)
		if err != nil {
			logs.Fatal("Failed to run migrations", map[string]interface{}{"error": err.Error()})
			return nil, err
		}
	}

	return dbConnection, nil
}

// selectDatabaseDriver selects the database driver according to the environment configuration
func (app *Application) selectDatabaseDriver() drivers.DriverConfig {
	driver, ok := SupportedDrivers[config.Database.Driver]
	if !ok {
		return nil
	}
	return driver
}
