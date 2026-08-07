package ports

import "github.com/chainedpixel/ordo-factus/internal/domain/auth/models"

// CryptManager determines the behavior of a data encryption manager
type CryptManager interface {
	GenerateAPIKey() (string, error)
	GenerateAPISecret() (string, error)
	EncryptStruct(token string, data models.HaciendaCredentials) (string, error)
	DecryptStruct(token string, data string) (models.HaciendaCredentials, error)
	GenerateBulkAPIKeys(amount int) ([]string, []string, error)
}
