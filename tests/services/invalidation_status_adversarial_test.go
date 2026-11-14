package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/config"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invalidation"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/shared_error"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/utils"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

const (
	adversarialOriginalCode    = "DTE-01-00000001-000000000000001"
	adversarialReplacementCode = "DTE-01-00000001-000000000000002"
)

var adversarialStatusCodes = map[string]string{
	constants.DocumentReceived: "",
	constants.DocumentInvalid:  "DocumentAlreadyInvalid",
	constants.DocumentRejected: "DocumentReject",
	constants.DocumentPending:  "DocumentPending",
	constants.DocumentExpired:  "DocumentExpired",
	"":                         "",
	"expired":                  "",
	"UNKNOWN":                  "",
}

func adversarialValidateStatus(t *testing.T, req structs.CreateInvalidationRequest, original, replacement string) error {
	t.Helper()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDTE := mocks.NewMockDTEManager(ctrl)
	mockDTE.EXPECT().VerifyStatus(gomock.Any(), gomock.Any(), adversarialOriginalCode).Return(original, nil).Times(1)
	if replacement != "" && original == constants.DocumentReceived && req.Reason.Type != 2 {
		mockDTE.EXPECT().VerifyStatus(gomock.Any(), gomock.Any(), adversarialReplacementCode).Return(replacement, nil).Times(1)
	}

	return invalidation.NewInvalidationService(mockDTE, nil).ValidateStatus(context.Background(), 1, req)
}

func assertAdversarialServiceError(t *testing.T, err error, wantCode string) {
	t.Helper()
	require.Error(t, err)
	var serviceErr *shared_error.ServiceError
	require.True(t, errors.As(err, &serviceErr), "expected ServiceError, got %T", err)
	assert.Equal(t, wantCode, serviceErr.Code)
	assert.NotContains(t, serviceErr.Message, "No se encontr")
	assert.NotContains(t, strings.ToLower(serviceErr.Message), "unknownerror")
	assert.NotContains(t, serviceErr.Message, "service_errors.")
	assert.NotContains(t, serviceErr.Message, "%!")
	assert.NotContains(t, serviceErr.Message, "[service_errors")
}

// TestInvalidationValidateStatusAdversarialOriginal tables every status of the original document.
func TestInvalidationValidateStatusAdversarialOriginal(t *testing.T) {
	test.TestMain(t)

	for status, wantCode := range adversarialStatusCodes {
		t.Run(fmt.Sprintf("original status %q", status), func(t *testing.T) {
			req := structs.CreateInvalidationRequest{
				GenerationCode: adversarialOriginalCode,
				Reason:         &structs.ReasonRequest{Type: 2},
			}
			err := adversarialValidateStatus(t, req, status, "")
			if wantCode == "" {
				assert.NoError(t, err)
				return
			}
			assertAdversarialServiceError(t, err, wantCode)
			assert.Contains(t, err.Error(), "document to invalidate")
		})
	}
}

// TestInvalidationValidateStatusAdversarialReplacement tables every status of the replacement document.
func TestInvalidationValidateStatusAdversarialReplacement(t *testing.T) {
	test.TestMain(t)

	for _, reasonType := range []int{1, 3} {
		for status, wantCode := range adversarialStatusCodes {
			t.Run(fmt.Sprintf("type %d replacement status %q", reasonType, status), func(t *testing.T) {
				replacement := adversarialReplacementCode
				req := structs.CreateInvalidationRequest{
					GenerationCode:            adversarialOriginalCode,
					ReplacementGenerationCode: &replacement,
					Reason:                    &structs.ReasonRequest{Type: reasonType},
				}
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()
				mockDTE := mocks.NewMockDTEManager(ctrl)
				mockDTE.EXPECT().VerifyStatus(gomock.Any(), gomock.Any(), adversarialOriginalCode).Return(constants.DocumentReceived, nil)
				mockDTE.EXPECT().VerifyStatus(gomock.Any(), gomock.Any(), adversarialReplacementCode).Return(status, nil)

				err := invalidation.NewInvalidationService(mockDTE, nil).ValidateStatus(context.Background(), 1, req)
				if wantCode == "" {
					assert.NoError(t, err)
					return
				}
				assertAdversarialServiceError(t, err, wantCode)
				assert.Contains(t, err.Error(), "replacement document")
			})
		}
	}
}

// TestInvalidationValidateStatusReplacementEdgeCases covers nil replacement and type 2 with an expired replacement.
func TestInvalidationValidateStatusReplacementEdgeCases(t *testing.T) {
	test.TestMain(t)

	t.Run("type 1 with nil replacement and received original", func(t *testing.T) {
		req := structs.CreateInvalidationRequest{GenerationCode: adversarialOriginalCode, Reason: &structs.ReasonRequest{Type: 1}}
		assert.NoError(t, adversarialValidateStatus(t, req, constants.DocumentReceived, ""))
	})

	t.Run("type 2 ignores expired replacement", func(t *testing.T) {
		replacement := adversarialReplacementCode
		req := structs.CreateInvalidationRequest{
			GenerationCode:            adversarialOriginalCode,
			ReplacementGenerationCode: &replacement,
			Reason:                    &structs.ReasonRequest{Type: 2},
		}
		assert.NoError(t, adversarialValidateStatus(t, req, constants.DocumentReceived, constants.DocumentExpired))
	})

	t.Run("expired original short-circuits before replacement lookup", func(t *testing.T) {
		replacement := adversarialReplacementCode
		req := structs.CreateInvalidationRequest{
			GenerationCode:            adversarialOriginalCode,
			ReplacementGenerationCode: &replacement,
			Reason:                    &structs.ReasonRequest{Type: 1},
		}
		err := adversarialValidateStatus(t, req, constants.DocumentExpired, constants.DocumentExpired)
		assertAdversarialServiceError(t, err, "DocumentExpired")
		assert.Contains(t, err.Error(), "document to invalidate")
	})
}

// TestInvalidationDocumentExpiredDefinedInBothLanguages checks that en and es define the expired message with the same params.
func TestInvalidationDocumentExpiredDefinedInBothLanguages(t *testing.T) {
	test.TestMain(t)

	messages := map[string]string{}
	for _, lang := range []string{"en", "es"} {
		v := viper.New()
		v.SetConfigFile(fmt.Sprintf("%s/%s/%s.yaml", utils.FindProjectRoot(), config.Server.AppLangPath, lang))
		v.SetConfigType("yaml")
		require.NoError(t, v.ReadInConfig())
		messages[lang] = v.GetString("service_errors.documentexpired")
		assert.NotEmpty(t, messages[lang], lang)
	}

	assert.Equal(t, strings.Count(messages["en"], "%s"), strings.Count(messages["es"], "%s"))
	assert.Equal(t, 1, strings.Count(messages["es"], "%s"))
}
