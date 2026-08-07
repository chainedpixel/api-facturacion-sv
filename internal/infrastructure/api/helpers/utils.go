package helpers

import (
	"net/http"

	"github.com/chainedpixel/ordo-factus/internal/application/dte"
	"github.com/gorilla/mux"
)

// DocumentConfig contains the configuration for handling a specific type of document
type DocumentConfig struct {
	UseCase         *dte.GenericDTEUseCase
	RequestType     interface{}
	DocumentType    string
	UsesContingency bool
}

func GetRequestVar(r *http.Request, key string) string {
	vars := mux.Vars(r)
	return vars[key]
}
