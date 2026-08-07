package drivers

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresDriver struct{}

func NewPostgresDriver() *PostgresDriver {
	return &PostgresDriver{}
}

func (p *PostgresDriver) GetDSN() gorm.Dialector {
	return postgres.Open(p.GetStringConnection())
}

func (p *PostgresDriver) GetStringConnection() string {
	return fmt.Sprintf("host=%s users=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/El_Salvador  options='-c client_encoding=%s'",
		config.Database.User,
		config.Database.Password,
		config.Database.Host,
		config.Database.Port,
		config.Database.Name,
		config.Database.Charset,
	)
}

func (p *PostgresDriver) GetHost() string {
	return config.Database.Port
}

func (p *PostgresDriver) GetDriverName() string {
	return "PostgreSQL"
}
