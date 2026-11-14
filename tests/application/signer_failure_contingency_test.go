package application

import (
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/application/dte"
	appPorts "github.com/chainedpixel/api-facturacion-sv/internal/application/ports"
	domainDTE "github.com/chainedpixel/api-facturacion-sv/internal/domain/core/dte"
	"github.com/chainedpixel/api-facturacion-sv/internal/infrastructure/api/helpers"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

// TestSignerFailureReleasesReservationAndSkipsContingency checks that a signing failure is not treated as a
// contingency: the reservation is released and the contingency handler does not store anything, because a contingency
// document must be delivered already signed.
func TestSignerFailureReleasesReservationAndSkipsContingency(t *testing.T) {
	test.TestMain(t)

	for name, signerErr := range map[string]error{
		"signer unreachable":    fmt.Errorf("%w: error calling signer service: %w", appPorts.ErrSigning, errors.New("dial tcp: connection refused")),
		"signer error response": fmt.Errorf("%w: signer service error - code: %s, messages: %v", appPorts.ErrSigning, "COD_810", []string{"bad password"}),
		"wrapped by transmitter": fmt.Errorf("transmission failed: %w",
			fmt.Errorf("%w: unexpected error response from signer service (status %d): %s", appPorts.ErrSigning, 500, "boom")),
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			authManager := mocks.NewMockAuthManager(ctrl)
			dteService := mocks.NewMockDTEService(ctrl)
			dteManager := mocks.NewMockDTEManager(ctrl)
			transmitter := mocks.NewMockBaseTransmitter(ctrl)
			seqManager := mocks.NewMockSequentialNumberManager(ctrl)

			document := map[string]interface{}{
				"identificacion": map[string]interface{}{
					"tipoDte":          "01",
					"numeroControl":    testControlNumber,
					"codigoGeneracion": testGenerationCode,
				},
			}
			authManager.EXPECT().GetIssuer(gomock.Any(), uint(1)).Return(fixtures.CreateDefaultIssuer(), nil)
			dteService.EXPECT().Create(gomock.Any(), gomock.Any(), uint(1)).Return(document, nil)
			transmitter.EXPECT().RetryTransmission(gomock.Any(), gomock.Any(), "token", "NIT-1").Return(nil, signerErr)
			seqManager.EXPECT().ReleaseReservation(gomock.Any(), testControlNumber, gomock.Any(), gomock.Any(), uint(1)).Return(nil)
			dteManager.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			requestMapper := &mapper.MapperAdapter{
				MapFunc: func(interface{}, *domainDTE.IssuerDTE, ...interface{}) (interface{}, error) { return document, nil },
			}
			useCase := dte.NewGenericDTEUseCase(authManager, dteManager, transmitter, dteService, seqManager, requestMapper,
				func(domain interface{}) interface{} { return domain }, nil, nil)

			_, _, err := useCase.Create(advContext(), struct{}{})
			require.Error(t, err)
			assert.True(t, errors.Is(err, appPorts.ErrSigning))

			contingencyManager := mocks.NewMockContingencyManager(ctrl)
			contingencyManager.EXPECT().StoreDocumentInContingency(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			assert.Nil(t, helpers.NewContingencyHandler(contingencyManager).HandleContingency(advContext(), document, "01", err))
		})
	}
}
