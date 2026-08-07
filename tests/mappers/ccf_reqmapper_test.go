package mappers

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestMapToCCFData(t *testing.T) {
	test.TestMain(t)

	issuer := fixtures.CreateDefaultIssuer()

	tests := []struct {
		name      string
		req       func() *structs.CreateCreditFiscalRequest
		wantErr   bool
		errorCode string
	}{
		{
			name: "Valid CCF request",
			req: func() *structs.CreateCreditFiscalRequest {
				return fixtures.CreateDefaultCreditFiscalRequest()
			},
			wantErr: false,
		},
		{
			name: "Null CCF request",
			req: func() *structs.CreateCreditFiscalRequest {
				return nil
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF without items",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Items = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF without summary",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Summary = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF without receiver",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF without commercial name",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver.CommercialName = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF with TotalIVA not zero",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Summary.TotalIVA = 10.0
				return req
			},
			wantErr:   true,
			errorCode: "InvalidField",
		},
		{
			name: "CCF with DocumentType not nil",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				docType := "36"
				req.Receiver.DocumentType = &docType
				return req
			},
			wantErr:   true,
			errorCode: "InvalidField",
		},
		{
			name: "CCF with DocumentNumber not nil",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				docNumber := "06141804941035"
				req.Receiver.DocumentNumber = &docNumber
				return req
			},
			wantErr:   true,
			errorCode: "InvalidField",
		},

		{
			name: "CCF without receiver name",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver.Name = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF without receiver address",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver.Address = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF without receiver NIT",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver.NIT = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF without receiver NRC",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver.NRC = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF without activity code",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver.ActivityCode = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF without activity description",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver.ActivityDesc = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF with invalid NIT format",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				invalidNIT := "1234"
				req.Receiver.NIT = &invalidNIT
				return req
			},
			wantErr:   true,
			errorCode: "InvalidPattern",
		},
		{
			name: "CCF with invalid NRC format",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				invalidNRC := "ABC123"
				req.Receiver.NRC = &invalidNRC
				return req
			},
			wantErr:   true,
			errorCode: "InvalidFormat",
		},
		{
			name: "CCF with invalid email format",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				invalidEmail := "not-an-email"
				req.Receiver.Email = &invalidEmail
				return req
			},
			wantErr:   true,
			errorCode: "InvalidEmail",
		},
		{
			name: "CCF with invalid phone",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				invalidPhone := "123"
				req.Receiver.Phone = &invalidPhone
				return req
			},
			wantErr:   true,
			errorCode: "InvalidPhone",
		},

		{
			name: "CCF with invalid municipality",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver.Address = &structs.AddressRequest{
					Department:   "06",
					Municipality: "99",
					Complement:   "Dirección de prueba",
				}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidMunicipality",
		},
		{
			name: "CCF with empty address fields",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver.Address = &structs.AddressRequest{
					Department:   "",
					Municipality: "",
					Complement:   "",
				}
				return req
			},
			wantErr:   true,
			errorCode: "ErrorMapping",
		},

		{
			name: "CCF with invalid item type",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.Type = 99
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidItemType",
		},
		{
			name: "CCF with negative quantity",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.Quantity = -5
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidQuantity",
		},
		{
			name: "CCF with invalid unit measure",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.UnitMeasure = 0
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidNumberRange",
		},

		{
			name: "CCF with negative amount",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.UnitPrice = -10.0
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAmount",
		},
		{
			name: "CCF with negative discount",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.Discount = -5.0
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDiscount",
		},
		{
			name: "CCF with invalid discount (>100%)",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.Discount = 150.0
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDiscount",
		},
		{
			name: "CCF with invalid payment condition",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Summary.OperationCondition = 99
				return req
			},
			wantErr:   true,
			errorCode: "InvalidNumberRange",
		},

		{
			name: "CCF with all valid optional fields",
			req: func() *structs.CreateCreditFiscalRequest {
				return fixtures.CreateCCFRequestWithAllOptionalFields()
			},
			wantErr: false,
		},
		{
			name: "CCF with invalid extension",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Extension = &structs.ExtensionRequest{
					DeliveryName:     "",
					DeliveryDocument: "123456",
					ReceiverName:     "Ana López",
					ReceiverDocument: "98765432-1",
				}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "CCF with invalid third party sale",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.ThirdPartySale = &structs.ThirdPartySaleRequest{
					NIT:  "",
					Name: "Empresa Tercero",
				}
				return req
			},
			wantErr:   true,
			errorCode: "ErrorMapping",
		},
		{
			name: "CCF with invalid appendix",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Appendixes = []structs.AppendixRequest{
					{
						Field: "",
						Label: "Etiqueta",
						Value: "Valor",
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "ErrorMapping",
		},
		{
			name: "CCF with invalid appendix label length",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Appendixes = []structs.AppendixRequest{
					{
						Field: "campo",
						Label: "ab",
						Value: "Valor",
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAppendixLabel",
		},
		{
			name: "CCF with invalid payment type",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Payments = []structs.PaymentRequest{
					{
						Code:   "77",
						Amount: 100.0,
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidLength",
		},
		{
			name: "CCF with invalid associated document code",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				description := "Documento adicional"
				detail := "Detalle del documento adicional"
				req.OtherDocs = []structs.OtherDocRequest{
					{
						DocumentCode: 99,
						Description:  &description,
						Detail:       &detail,
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAssociatedDocumentCode",
		},
		{
			name: "CCF with missing doctor for medical document",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				description := "Documento médico"
				detail := "Detalle médico"
				req.OtherDocs = []structs.OtherDocRequest{
					{
						DocumentCode: 3,
						Description:  &description,
						Detail:       &detail,
						Doctor:       nil,
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},

		{
			name: "CCF with valid NIT",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				validNIT := "06141804941035"
				req.Receiver.NIT = &validNIT
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid NRC",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				validNRC := "1234567"
				req.Receiver.NRC = &validNRC
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid email",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				validEmail := "cliente@gmail.com"
				req.Receiver.Email = &validEmail
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid phone",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				validPhone := "22345678"
				req.Receiver.Phone = &validPhone
				return req
			},
			wantErr: false,
		},

		{
			name: "CCF with valid address",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Receiver.Address = &structs.AddressRequest{
					Department:   "06",
					Municipality: "20",
					Complement:   "Colonia Escalón, Calle La Reforma #123",
				}
				return req
			},
			wantErr: false,
		},

		{
			name: "CCF with valid item type (product)",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.Type = 1
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid item type (service)",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.Type = 2
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid item type (both)",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.Type = 3
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid unit measure",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.UnitMeasure = 59
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr: false,
		},

		{
			name: "CCF with valid amount",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.UnitPrice = 100.50
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid discount",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				item := fixtures.CreateDefaultCreditItem(0)
				item.Discount = 10.5
				req.Items = []structs.CreditItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid payment condition (credit)",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Summary.OperationCondition = 2
				return req
			},
			wantErr: false,
		},

		{
			name: "CCF with valid extension",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				observation := "Observación válida"
				vehiculePlate := "P123456"
				req.Extension = &structs.ExtensionRequest{
					DeliveryName:     "Juan Martínez",
					DeliveryDocument: "12345678-9",
					ReceiverName:     "María González",
					ReceiverDocument: "98765432-1",
					Observation:      &observation,
					VehiculePlate:    &vehiculePlate,
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid third party sale",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.ThirdPartySale = &structs.ThirdPartySaleRequest{
					NIT:  "06141804941035",
					Name: "Tercero Válido, S.A. de C.V.",
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid appendix",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.Appendixes = []structs.AppendixRequest{
					{
						Field: "campo_valido",
						Label: "Etiqueta válida",
						Value: "Valor válido para este apéndice",
					},
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid payment type",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				reference := "REF-123"
				req.Payments = []structs.PaymentRequest{
					{
						Code:      "01",
						Amount:    100.0,
						Reference: &reference,
					},
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "CCF with valid related document",
			req: func() *structs.CreateCreditFiscalRequest {
				req := fixtures.CreateDefaultCreditFiscalRequest()
				req.RelatedDocs = []structs.RelatedDocRequest{
					{
						DocumentType:   "01",
						GenerationType: 1,
						DocumentNumber: "000123456",
						EmissionDate:   "2023-01-15",
					},
				}
				return req
			},
			wantErr: false,
		},
	}

	mapper := request_mapper.NewCCFMapper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req()

			got, err := mapper.MapToCCFData(req, issuer)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorCode != "" {
					test.AssertErrorCode(t, err, tt.errorCode)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotNil(t, got.InputDataCommon)
			assert.NotNil(t, got.InputDataCommon.Issuer)
			assert.NotNil(t, got.InputDataCommon.Identification)
			assert.NotNil(t, got.InputDataCommon.Receiver)
			assert.Len(t, got.Items, len(req.Items))
			assert.NotNil(t, got.CreditSummary)

			if req.ThirdPartySale != nil {
				assert.NotNil(t, got.ThirdPartySale)
			}

			if req.Extension != nil {
				assert.NotNil(t, got.Extension)
			}

			if req.RelatedDocs != nil {
				assert.NotNil(t, got.RelatedDocs)
				assert.Len(t, got.RelatedDocs, len(req.RelatedDocs))
			}

			if req.OtherDocs != nil {
				assert.NotNil(t, got.OtherDocs)
				assert.Len(t, got.OtherDocs, len(req.OtherDocs))
			}

			if req.Appendixes != nil {
				assert.NotNil(t, got.Appendixes)
				assert.Len(t, got.Appendixes, len(req.Appendixes))
			}
		})
	}
}
