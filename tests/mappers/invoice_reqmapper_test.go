package mappers

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestMapToInvoiceData(t *testing.T) {
	test.TestMain(t)

	issuer := fixtures.CreateDefaultIssuer()

	tests := []struct {
		name      string
		req       func() *structs.CreateInvoiceRequest
		wantErr   bool
		errorCode string
	}{
		{
			name: "Valid invoice request",
			req: func() *structs.CreateInvoiceRequest {
				return fixtures.CreateDefaultInvoiceRequest()
			},
			wantErr: false,
		},
		{
			name: "Null invoice request",
			req: func() *structs.CreateInvoiceRequest {
				return nil
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Invoice without items",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				req.Items = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Invoice without summary",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				req.Summary = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},

		{
			name: "Invoice with DocumentType but no DocumentNumber",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				docType := "36"
				req.Receiver.DocumentType = &docType
				req.Receiver.DocumentNumber = nil
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDocumentTypeAndNumber",
		},
		{
			name: "Invoice with invalid email format",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				invalidEmail := "not-an-email"
				req.Receiver.Email = &invalidEmail
				return req
			},
			wantErr:   true,
			errorCode: "InvalidEmail",
		},
		{
			name: "Invoice with invalid phone",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				invalidPhone := "123"
				req.Receiver.Phone = &invalidPhone
				return req
			},
			wantErr:   true,
			errorCode: "InvalidPhone",
		},

		{
			name: "Invoice with invalid municipality",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with empty address fields",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with invalid item type",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.Type = 99
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidItemType",
		},
		{
			name: "Invoice with negative quantity",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.Quantity = -5
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidQuantity",
		},
		{
			name: "Invoice with invalid unit measure",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.UnitMeasure = 0
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidNumberRange",
		},

		{
			name: "Invoice with negative amount",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.UnitPrice = -10.0
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAmount",
		},
		{
			name: "Invoice with negative discount",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.Discount = -5.0
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDiscount",
		},
		{
			name: "Invoice with invalid discount (>100%)",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.Discount = 150.0
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDiscount",
		},

		{
			name: "Invoice with all valid optional fields",
			req: func() *structs.CreateInvoiceRequest {
				return fixtures.CreateInvoiceRequestWithAllOptionalFields()
			},
			wantErr: false,
		},
		{
			name: "Invoice with invalid third party sale",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with invalid appendix",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with invalid appendix label length",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with invalid payment type",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with invalid associated document code",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with missing doctor for medical document",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with invalid doctor info",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				nit := "12345678901234"
				req.OtherDocs = []structs.OtherDocRequest{
					{
						DocumentCode: 3,
						Doctor: &structs.DoctorRequest{
							Name:        "",
							NIT:         &nit,
							ServiceType: 1,
						},
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "ErrorMapping",
		},
		{
			name: "Invoice with missing doctor documents",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				req.OtherDocs = []structs.OtherDocRequest{
					{
						DocumentCode: 3,
						Doctor: &structs.DoctorRequest{
							Name:        "Dr. Juan Pérez",
							ServiceType: 1,
						},
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "ErrorMapping",
		},
		{
			name: "Invoice with valid DocumentType and DocumentNumber",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				docType := "36"
				docNumber := "06141804941035"
				req.Receiver.DocumentType = &docType
				req.Receiver.DocumentNumber = &docNumber
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with valid email format",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				validEmail := "test.valid@google.com"
				req.Receiver.Email = &validEmail
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with valid phone",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				validPhone := "22123456"
				req.Receiver.Phone = &validPhone
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with NIT",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				validNIT := "06141804941035"
				req.Receiver.NIT = &validNIT
				return req
			},
			wantErr: true,
		},

		{
			name: "Invoice with valid municipality",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with valid complete address",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				req.Receiver.Address = &structs.AddressRequest{
					Department:   "05",
					Municipality: "23",
					Complement:   "Residencial Santa Elena, Calle Principal #45",
				}
				return req
			},
			wantErr: false,
		},

		{
			name: "Invoice with valid item type (product)",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.Type = 1
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with valid item type (service)",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.Type = 2
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with positive quantity",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.Quantity = 10.5
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with valid unit measure",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.UnitMeasure = 59
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr: false,
		},

		{
			name: "Invoice with positive amount",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.UnitPrice = 100.50
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with zero discount",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.Discount = 0.0
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with valid discount (10%)",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				item := fixtures.CreateDefaultInvoiceItem(0)
				item.Discount = 10.0
				req.Items = []structs.InvoiceItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with valid third party sale",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				req.ThirdPartySale = &structs.ThirdPartySaleRequest{
					NIT:  "06141804941035",
					Name: "Tercero Válido, S.A. de C.V.",
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with valid appendix",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with valid payment type",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
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
			name: "Invoice with valid associated document code",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				description := "Documento adicional válido"
				detail := "Detalle válido"
				req.OtherDocs = []structs.OtherDocRequest{
					{
						DocumentCode: 1,
						Description:  &description,
						Detail:       &detail,
					},
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "Invoice with valid doctor for medical document",
			req: func() *structs.CreateInvoiceRequest {
				req := fixtures.CreateDefaultInvoiceRequest()
				nit := "06141804941035"
				req.OtherDocs = []structs.OtherDocRequest{
					{
						DocumentCode: 3,
						Doctor: &structs.DoctorRequest{
							Name:        "Dr. Juan Pérez",
							NIT:         &nit,
							ServiceType: 1,
						},
					},
				}
				return req
			},
			wantErr: false,
		},
	}

	mapper := request_mapper.NewInvoiceMapper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req()

			got, err := mapper.MapToInvoiceData(req, issuer)

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
			assert.Equal(t, 2, got.InputDataCommon.Identification.GetVersion())
			assert.NotNil(t, got.InputDataCommon.Receiver)
			assert.Len(t, got.Items, len(req.Items))
			assert.NotNil(t, got.InvoiceSummary)

			if req.ThirdPartySale != nil {
				assert.NotNil(t, got.ThirdPartySale)
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
