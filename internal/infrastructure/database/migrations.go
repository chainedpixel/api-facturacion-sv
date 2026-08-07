package database

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

// modelsToMigrate contains all models that should be migrated
var modelsToMigrate = []schema.Tabler{
	&db_models.User{},
	&db_models.BranchOffice{},
	&db_models.Address{},
	&db_models.DTEDetails{},
	&db_models.DTEDocument{},
	&db_models.ContingencyDocument{},
	&db_models.ControlNumberSequence{},
	&db_models.FailedSequenceNumber{},
	&db_models.ReservedSequenceNumber{},
	&db_models.DomainEvent{},
	&db_models.UserNotification{},
	&db_models.NotifiableUser{},
	&db_models.DTEBalanceControl{},
	&db_models.DTEBalanceTransaction{},
}

// RunMigrations runs all database migrations
func RunMigrations(db *gorm.DB) error {
	logs.Info("Starting database migrations")

	for i, model := range modelsToMigrate {
		tn := model.TableName()
		logs.Info(fmt.Sprintf("Starting model migration #%d: %s", i+1, tn))

		if err := db.AutoMigrate(model); err != nil {
			logs.Error("Failed to migrate model", map[string]interface{}{
				"index": i + 1,
				"model": tn,
				"error": err.Error(),
			})
			return err
		}

		logs.Info(fmt.Sprintf("Successfully migrated model %s", tn))
	}

	logs.Info("All migrations completed successfully")
	return nil
}
