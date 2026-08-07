package containers

import (
	"github.com/chainedpixel/ordo-factus/config/drivers"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	contiPorts "github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency"
	dtePorts "github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/repositories"
	"gorm.io/gorm"
)

type RepositoryContainer struct {
	connection *drivers.DbConnection
	db         *gorm.DB

	authRepo                   auth.AuthRepositoryPort
	sequentialNumberRepo       ports.SequentialNumberRepositoryPort
	failedSequentialNumberRepo ports.FailedSequenceNumberRepositoryPort
	reservedSequenceRepo       dtePorts.ReservedSequenceRepositoryPort
	dteRepo                    dtePorts.DTERepositoryPort
	contingencyRepo            contiPorts.ContingencyRepositoryPort
}

func NewRepositoryContainer(connection *drivers.DbConnection) *RepositoryContainer {
	return &RepositoryContainer{
		connection: connection,
		db:         connection.Db,
	}
}

func (c *RepositoryContainer) Initialize() {
	c.authRepo = repositories.NewAuthRepository(c.db)
	c.sequentialNumberRepo = repositories.NewControlNumberRepository(c.db)
	c.reservedSequenceRepo = repositories.NewReservedSequenceRepository(c.db)
	c.dteRepo = repositories.NewDTERepository(c.db)
	c.contingencyRepo = repositories.NewContingencyRepository(c.db)
	c.failedSequentialNumberRepo = repositories.NewFailedSequenceNumberRepository(c.db)
}

func (c *RepositoryContainer) FailedSequentialNumberRepo() ports.FailedSequenceNumberRepositoryPort {
	return c.failedSequentialNumberRepo
}

func (c *RepositoryContainer) ContingencyRepo() contiPorts.ContingencyRepositoryPort {
	return c.contingencyRepo
}

func (c *RepositoryContainer) DTERepo() dtePorts.DTERepositoryPort {
	return c.dteRepo
}

func (c *RepositoryContainer) AuthRepo() auth.AuthRepositoryPort {
	return c.authRepo
}

func (c *RepositoryContainer) SequentialNumberRepo() ports.SequentialNumberRepositoryPort {
	return c.sequentialNumberRepo
}

func (c *RepositoryContainer) ReservedSequenceRepo() dtePorts.ReservedSequenceRepositoryPort {
	return c.reservedSequenceRepo
}
