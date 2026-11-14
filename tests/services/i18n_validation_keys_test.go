package services

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/stretchr/testify/assert"
)

// TestReplacementCodeKeysTranslate guarantees the replacement-code keys resolve under validation_errors.
func TestReplacementCodeKeysTranslate(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		key    string
		params []interface{}
		want   string
	}{
		{"ReplacementCodeNotAllowedForType", []interface{}{"05"}, "must be null when the invalidated document is of type 05"},
		{"InvalidReplacementCode1And3", nil, "required for invalidation types 1 and 3"},
		{"InvalidInvalidationType2", nil, "must be null for type 2"},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			msg := constants.GetErrorMessage(tc.key, tc.params...)
			assert.Contains(t, msg, tc.want)
			assert.NotContains(t, msg, "unknownerror")
			assert.NotContains(t, msg, "validation_errors.")
		})
	}
}

// TestInvalidationDTEErrorKeysTranslate checks that every key raised with NewDTEErrorSimple or NewValidationError
// in the invalidation domain resolves under validation_errors.
func TestInvalidationDTEErrorKeysTranslate(t *testing.T) {
	test.TestMain(t)

	pattern := regexp.MustCompile(`New(?:DTEErrorSimple|ValidationError)\("([A-Za-z0-9_]+)"`)
	root := filepath.Join(utils.FindProjectRoot(), "internal", "domain", "dte", "invalidation")

	keys := map[string]bool{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return walkErr
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, match := range pattern.FindAllStringSubmatch(string(content), -1) {
			keys[match[1]] = true
		}
		return nil
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, keys)

	for key := range keys {
		msg := config.Translate("validation_errors." + key)
		assert.False(t, strings.HasPrefix(msg, "[validation_errors."), "key %s is not defined under validation_errors", key)
	}
}

// TestInvalidationStatusServiceKeysTranslate checks that the status rejection keys resolve under service_errors.
func TestInvalidationStatusServiceKeysTranslate(t *testing.T) {
	test.TestMain(t)

	for _, key := range []string{"DocumentAlreadyInvalid", "DocumentReject", "DocumentPending", "DocumentExpired"} {
		msg := config.Translate("service_errors." + key)
		assert.False(t, strings.HasPrefix(msg, "[service_errors."), "key %s is not defined under service_errors", key)
	}
	assert.Contains(t, config.Translate("service_errors.DocumentExpired"), "contingency")
}

// TestInvalidCreditPaymentAmountKeyTranslates checks that the credit payment amount key resolves with its parameter.
func TestInvalidCreditPaymentAmountKeyTranslates(t *testing.T) {
	test.TestMain(t)

	msg := constants.GetErrorMessage("InvalidCreditPaymentAmount", 25.5)
	assert.Contains(t, msg, "must be 0.00")
	assert.Contains(t, msg, "25.5")
	assert.NotContains(t, msg, "validation_errors.")
}
