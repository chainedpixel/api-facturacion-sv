package mappers

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestMapToDebitNoteData(t *testing.T) {
	test.TestMain(t)

	issuer := fixtures.CreateDefaultIssuer()

	tests := []struct {
		name      string
		req       func() *structs.CreateDebitNoteRequest
		wantErr   bool
		errorCode string
	}{
		{
			name: "Valid debit note request",
			req: func() *structs.CreateDebitNoteRequest {
				return fixtures.CreateDefaultDebitNoteRequest()
			},
			wantErr: false,
		},
		{
			name: "Debit note with all optional fields",
			req: func() *structs.CreateDebitNoteRequest {
				return fixtures.CreateDebitNoteRequestWithAllOptionalFields()
			},
			wantErr: false,
		},

		{
			name: "Null debit note request",
			req: func() *structs.CreateDebitNoteRequest {
				return nil
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without items",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Items = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without summary",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Summary = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without receiver",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Receiver = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without related documents",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.RelatedDocs = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note with empty related documents",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.RelatedDocs = []structs.RelatedDocRequest{}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},

		{
			name: "Debit note without receiver name",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Receiver.Name = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without receiver email",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Receiver.Email = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without receiver address",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Receiver.Address = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without receiver NIT",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Receiver.NIT = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without receiver NRC",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Receiver.NRC = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without receiver activity code",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Receiver.ActivityCode = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without receiver activity description",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Receiver.ActivityDesc = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note without receiver trade name",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Receiver.CommercialName = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note with invalid NIT",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				invalidNIT := "123456"
				req.Receiver.NIT = &invalidNIT
				return req
			},
			wantErr:   true,
			errorCode: "InvalidPattern",
		},
		{
			name: "Debit note with invalid NRC",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				invalidNRC := "ABC123"
				req.Receiver.NRC = &invalidNRC
				return req
			},
			wantErr:   true,
			errorCode: "InvalidFormat",
		},
		{
			name: "Debit note with invalid email",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				invalidEmail := "no-es-email"
				req.Receiver.Email = &invalidEmail
				return req
			},
			wantErr:   true,
			errorCode: "InvalidEmail",
		},
		{
			name: "Debit note with invalid phone",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				invalidPhone := "123"
				req.Receiver.Phone = &invalidPhone
				return req
			},
			wantErr:   true,
			errorCode: "InvalidPhone",
		},
		{
			name: "Debit note with invalid municipality",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
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
			name: "Debit note with invalid item type",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				item := fixtures.CreateDefaultDebitNoteItem(0)
				item.Type = 99
				req.Items = []structs.DebitNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidItemType",
		},
		{
			name: "Debit note with negative quantity in item",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				item := fixtures.CreateDefaultDebitNoteItem(0)
				item.Quantity = -5
				req.Items = []structs.DebitNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidQuantity",
		},
		{
			name: "Debit note with invalid unit measure",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				item := fixtures.CreateDefaultDebitNoteItem(0)
				item.UnitMeasure = 0
				req.Items = []structs.DebitNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidNumberRange",
		},
		{
			name: "Debit note with negative unit price",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				item := fixtures.CreateDefaultDebitNoteItem(0)
				item.UnitPrice = -10.0
				req.Items = []structs.DebitNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAmount",
		},
		{
			name: "Debit note with negative discount in item",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				item := fixtures.CreateDefaultDebitNoteItem(0)
				item.Discount = -5.0
				req.Items = []structs.DebitNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDiscount",
		},
		{
			name: "Debit note with discount greater than 100% in item",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				item := fixtures.CreateDefaultDebitNoteItem(0)
				item.Discount = 150.0
				req.Items = []structs.DebitNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDiscount",
		},

		{
			name: "Debit note with invalid related document type",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.RelatedDocs = []structs.RelatedDocRequest{
					{
						DocumentType:   "99",
						GenerationType: 1,
						DocumentNumber: "12345678",
						EmissionDate:   "2023-01-15",
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDTEType",
		},
		{
			name: "Debit note with invalid generation type in related document",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.RelatedDocs = []structs.RelatedDocRequest{
					{
						DocumentType:   "03",
						GenerationType: 99,
						DocumentNumber: "12345678",
						EmissionDate:   "2023-01-15",
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidLength",
		},
		{
			name: "Debit note with empty document number in related document",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.RelatedDocs = []structs.RelatedDocRequest{
					{
						DocumentType:   "03",
						GenerationType: 1,
						DocumentNumber: "",
						EmissionDate:   "2023-01-15",
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Debit note with invalid date format in related document",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.RelatedDocs = []structs.RelatedDocRequest{
					{
						DocumentType:   "03",
						GenerationType: 2,
						DocumentNumber: "0408DCE7-8E96-47AA-92B2-B0F0C8FBDAF3",
						EmissionDate:   "15/01/2023",
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "ErrorMapping",
		},
		{
			name: "Debit note with future date in related document",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.RelatedDocs = []structs.RelatedDocRequest{
					{
						DocumentType:   "03",
						GenerationType: 1,
						DocumentNumber: "12345678",
						EmissionDate:   "2099-01-15",
					},
				}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDateTime",
		},

		{
			name: "Debit note with invalid operation condition",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Summary.OperationCondition = 99
				return req
			},
			wantErr:   true,
			errorCode: "InvalidNumberRange",
		},

		{
			name: "Debit note with invalid third-party sale",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
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
			name: "Debit note with invalid appendix (empty field)",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Appendixes = []structs.AppendixRequest{
					{Field: "", Label: "Etiqueta", Value: "Valor"},
				}
				return req
			},
			wantErr:   true,
			errorCode: "ErrorMapping",
		},
		{
			name: "Debit note with invalid payment type",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Payments = []structs.PaymentRequest{
					{Code: "77", Amount: 100.0},
				}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidLength",
		},
		{
			name: "Debit note with invalid associated document code",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				description := "Documento adicional"
				detail := "Detalle"
				req.OtherDocs = []structs.OtherDocRequest{
					{DocumentCode: 99, Description: &description, Detail: &detail},
				}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAssociatedDocumentCode",
		},

		{
			name: "Debit note with valid NIT",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				validNIT := "06141804941035"
				req.Receiver.NIT = &validNIT
				return req
			},
			wantErr: false,
		},
		{
			name: "Debit note with valid email",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				validEmail := "cliente@empresa.com"
				req.Receiver.Email = &validEmail
				return req
			},
			wantErr: false,
		},
		{
			name: "Debit note with valid phone",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				validPhone := "22345678"
				req.Receiver.Phone = &validPhone
				return req
			},
			wantErr: false,
		},
		{
			name: "Debit note with valid address",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
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
			name: "Debit note with valid CCF related document",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.RelatedDocs = []structs.RelatedDocRequest{
					{
						DocumentType:   "03",
						GenerationType: 1,
						DocumentNumber: "12345678",
						EmissionDate:   "2023-01-15",
					},
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "Debit note with multiple valid related documents",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.RelatedDocs = []structs.RelatedDocRequest{
					{
						DocumentType:   "03",
						GenerationType: 1,
						DocumentNumber: "12345678",
						EmissionDate:   "2023-01-15",
					},
					{
						DocumentType:   "07",
						GenerationType: 1,
						DocumentNumber: "87654321",
						EmissionDate:   "2023-01-16",
					},
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "Debit note with valid operation condition",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Summary.OperationCondition = 2
				return req
			},
			wantErr: false,
		},
		{
			name: "Debit note with service item type",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				item := fixtures.CreateDefaultDebitNoteItem(0)
				item.Type = 2
				req.Items = []structs.DebitNoteItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "Debit note with valid payment type",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				reference := "REF-001"
				req.Payments = []structs.PaymentRequest{
					{Code: "01", Amount: 113.0, Reference: &reference},
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "Debit note with valid appendix",
			req: func() *structs.CreateDebitNoteRequest {
				req := fixtures.CreateDefaultDebitNoteRequest()
				req.Appendixes = []structs.AppendixRequest{
					{Field: "orden_compra", Label: "Orden de compra", Value: "OC-2024-001"},
				}
				return req
			},
			wantErr: false,
		},
	}

	mapper := request_mapper.NewDebitNoteMapper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req()

			got, err := mapper.MapToDebitNoteData(req, issuer)

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
			assert.Equal(t, 4, got.InputDataCommon.Identification.GetVersion())
			assert.NotNil(t, got.InputDataCommon.Receiver)
			assert.NotNil(t, got.InputDataCommon.RelatedDocs)
			assert.Len(t, got.Items, len(req.Items))
			assert.NotNil(t, got.DebitSummary)

			if req.ThirdPartySale != nil {
				assert.NotNil(t, got.ThirdPartySale)
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
