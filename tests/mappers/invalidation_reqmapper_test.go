package mappers

import (
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestMapToInvalidationData(t *testing.T) {
	test.TestMain(t)

	issuer := fixtures.CreateDefaultIssuer()

	invoiceDTE := createInvoiceDTE()
	ccfDTE := createCCFDTE()

	tests := []struct {
		name      string
		req       func() *structs.CreateInvalidationRequest
		baseDTE   *dte.DTEDetails
		wantErr   bool
		errorCode string
	}{
		{
			name: "Valid invalidation request for invoice with replacement",
			req: func() *structs.CreateInvalidationRequest {
				return fixtures.CreateDefaultInvalidationRequest()
			},
			baseDTE: invoiceDTE,
			wantErr: false,
		},
		{
			name: "Valid invalidation request for CCF with replacement",
			req: func() *structs.CreateInvalidationRequest {
				return fixtures.CreateDefaultInvalidationRequest()
			},
			baseDTE: ccfDTE,
			wantErr: false,
		},
		{
			name: "Null invalidation request",
			req: func() *structs.CreateInvalidationRequest {
				return nil
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Invalidation with null base DTE",
			req: func() *structs.CreateInvalidationRequest {
				return fixtures.CreateDefaultInvalidationRequest()
			},
			baseDTE:   nil,
			wantErr:   true,
			errorCode: "ErrorMapping",
		},

		{
			name: "Invalidation with invalid type",
			req: func() *structs.CreateInvalidationRequest {
				return fixtures.CreateInvalidationWithInvalidType()
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "InvalidInvalidationType",
		},
		{
			name: "Type 3 invalidation without reason",
			req: func() *structs.CreateInvalidationRequest {
				return fixtures.CreateInvalidationTypeWithoutReason()
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "InvalidInvalidationType3",
		},
		{
			name: "Type 2 invalidation with replacement code",
			req: func() *structs.CreateInvalidationRequest {
				return fixtures.CreateInvalidationType2WithReplacementCode()
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "InvalidInvalidationType2",
		},
		{
			name: "Valid type 2 invalidation (annulment)",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.Type = 2
				req.ReplacementGenerationCode = nil
				return req
			},
			baseDTE: invoiceDTE,
			wantErr: false,
		},
		{
			name: "Valid type 3 invalidation (definitive)",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.Type = 3
				reason := "Documento con errores graves no recuperables"
				req.Reason.Reason = &reason
				return req
			},
			baseDTE: invoiceDTE,
			wantErr: false,
		},

		{
			name: "Invalidation without responsible name",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.ResponsibleName = ""
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Invalidation without responsible document type",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.ResponsibleDocType = ""
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Invalidation without responsible document number",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.ResponsibleNumDoc = ""
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Invalidation with invalid responsible document type",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.ResponsibleDocType = "99"
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "InvalidDocumentForReceiver",
		},

		{
			name: "Invalidation without requestor name",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.RequestorName = ""
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Invalidation without requestor document type",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.RequestorDocType = ""
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Invalidation without requestor document number",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.RequestorNumDoc = ""
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Invalidation with invalid requestor document type",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.RequestorDocType = "99"
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "InvalidDocumentForReceiver",
		},

		{
			name: "Type 3 invalidation with empty reason",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.Type = 3
				emptyReason := ""
				req.Reason.Reason = &emptyReason
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "InvalidInvalidationReason",
		},
		{
			name: "Type 3 invalidation with reason too short",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.Type = 3
				shortReason := "abcd"
				req.Reason.Reason = &shortReason
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "InvalidInvalidationReason",
		},
		{
			name: "Type 3 invalidation with reason too long",
			req: func() *structs.CreateInvalidationRequest {
				req := fixtures.CreateDefaultInvalidationRequest()
				req.Reason.Type = 3
				tooLongReason := createStringWithLength(251)
				req.Reason.Reason = &tooLongReason
				return req
			},
			baseDTE:   invoiceDTE,
			wantErr:   true,
			errorCode: "InvalidInvalidationReason",
		},
	}

	mapper := request_mapper.NewInvalidationMapper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req()

			if req != nil {
				validationErr := mapper.ValidateInvalidationReRequest(req)
				if validationErr != nil {
					if tt.wantErr {
						assert.Error(t, validationErr)
						if tt.errorCode != "" {
							test.AssertErrorCode(t, validationErr, tt.errorCode)
						}
						return
					}
					t.Fatalf("Unexpected validation error: %v", validationErr)
				}
			}

			emissionDate := time.Now()
			got, err := mapper.MapToInvalidationData(req, issuer, tt.baseDTE, emissionDate)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorCode != "" {
					test.AssertErrorCode(t, err, tt.errorCode)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)

			if tt.baseDTE != nil {
				assert.Equal(t, tt.baseDTE.DTEType, got.Document.Type.GetValue())
				assert.Equal(t, tt.baseDTE.ID, got.Document.GenerationCode.GetValue())
				assert.Equal(t, tt.baseDTE.ControlNumber, got.Document.ControlNumber.GetValue())
			}

			assert.NotNil(t, got.Identification)
			assert.NotNil(t, got.Reason)
			assert.NotNil(t, got.Issuer)
			assert.NotNil(t, got.Document)

			assert.Equal(t, req.Reason.Type, int(got.Reason.Type.GetValue()))

			if req.ReplacementGenerationCode != nil {
				assert.NotNil(t, got.Document.ReplacementCode)
				assert.Equal(t, *req.ReplacementGenerationCode, got.Document.ReplacementCode.GetValue())
			} else {
				assert.Nil(t, got.Document.ReplacementCode)
			}

			if req.Reason.Type == 3 {
				assert.NotNil(t, got.Reason.Reason)
				assert.Equal(t, *req.Reason.Reason, got.Reason.Reason.GetValue())
			}
		})
	}
}

// Helper functions to create test DTEs
func createInvoiceDTE() *dte.DTEDetails {
	return &dte.DTEDetails{
		ID:             "FF54E9DB-79C3-42CE-B432-EC522C97EFB9",
		DTEType:        "01",
		ControlNumber:  "DTE-01-00000000-000000000000001",
		ReceptionStamp: utils.ToStringPointer("2025AAFEEE1A566A44F19A622C0C35C8A1B6FAZM"),
		JSONData: `{
			"receptor": {
				"nombre": "Cliente Ejemplo",
				"telefono": "22123456",
				"correo": "cliente@example.com",
				"tipoDocumento": "13",
				"numDocumento": "01234567-8"
			},
			"resumen": {
				"totalIva": 13.00
			}
		}`,
	}
}

func createCCFDTE() *dte.DTEDetails {
	return &dte.DTEDetails{
		ID:             "AD54E9BB-79A3-42AE-B432-EC522C97EFB7",
		DTEType:        "03",
		ControlNumber:  "DTE-03-00000000-000000000000001",
		ReceptionStamp: utils.ToStringPointer("2025BBFEEE1A566A44F19A622C0C35C8A1B6FAZM"),
		JSONData: `{
			"receptor": {
				"nombre": "Empresa Cliente, S.A. de C.V.",
				"telefono": "22123456",
				"correo": "empresa@example.com",
				"nit": "06141804941035",
				"nrc": "1234567"
			},
			"resumen": {
				"totalIva": 13.00
			}
		}`,
	}
}

// Helper function to create a string of a given length
func createStringWithLength(length int) string {
	s := ""
	for i := 0; i < length; i++ {
		s += "a"
	}
	return s
}
