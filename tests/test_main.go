package test

import (
	"fmt"
	"testing"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/logs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	"github.com/sirupsen/logrus"
)

// TestMain configures the test environment for all tests.
func TestMain(m *testing.T) {
	rootPath := utils.FindProjectRoot()

	config.InitEnvTesting()

	base.SkipMXValidation = true

	err := utils.TimeInit()
	if err != nil {
		panic("Error initializing time: " + err.Error())
	}

	langPath := fmt.Sprintf("%s/%s", rootPath, config.Server.AppLangPath)
	err = config.InitTranslations(langPath, "en")
	if err != nil {
		panic("Error initializing translations: " + err.Error())
	}

	err = logs.InitLogger(logrus.DebugLevel.String(), config.Log.Path)
	if err != nil {
		panic("Error initializing logger: " + err.Error())
	}
}
