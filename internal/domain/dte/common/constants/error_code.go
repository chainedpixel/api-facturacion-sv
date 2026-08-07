package constants

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/config"
)

// GetErrorMessage retrieves the error message according to the configured language
func GetErrorMessage(errorCode string, params ...interface{}) string {
	message := config.Translate(fmt.Sprintf("validation_errors.%s", errorCode), params...)

	if config.Server.Debug {
		return fmt.Sprintf("[%s] %s", errorCode, message)
	}

	return message
}
