package mappers

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestMapToRemissionNoteData(t *testing.T) {
	test.TestMain(t)

	issuer := fixtures.CreateDefaultIssuer()

	tests := []struct {
		name      string
		req       func() *structs.CreateRemissionNoteRequest
		wantErr   bool
		errorCode string
	}{
		{
			name: "Valid remission note request",
			req: func() *structs.CreateRemissionNoteRequest {
				return fixtures.CreateDefaultRemissionNoteRequest()
			},
			wantErr: false,
		},
		{
			name: "Remission note with all optional fields",
			req: func() *structs.CreateRemissionNoteRequest {
				return fixtures.CreateRemissionNoteRequestWithAllOptionalFields()
			},
			wantErr: false,
		},

		{
			name: "Null remission note request",
			req: func() *structs.CreateRemissionNoteRequest {
				return nil
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note without items",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.Items = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note without summary",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.Summary = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note without receiver",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.Receiver = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},

		{
			name: "Remission note without BienTitulo",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.Receiver.BienTitulo = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note with empty BienTitulo",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				empty := ""
				req.Receiver.BienTitulo = &empty
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note with incorrect BienTitulo length",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				invalidTitulo := "ABC"
				req.Receiver.BienTitulo = &invalidTitulo
				return req
			},
			wantErr:   true,
			errorCode: "InvalidLength",
		},
		{
			name: "Remission note without receiver name",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.Receiver.Name = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note without receiver email",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.Receiver.Email = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note without receiver address",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.Receiver.Address = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note without receiver document type",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.Receiver.DocumentType = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note without receiver document number",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.Receiver.DocumentNumber = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note with invalid email in receiver",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				invalidEmail := "no-es-email"
				req.Receiver.Email = &invalidEmail
				return req
			},
			wantErr:   true,
			errorCode: "InvalidEmail",
		},
		{
			name: "Remission note with invalid phone in receiver",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				invalidPhone := "123"
				req.Receiver.Phone = &invalidPhone
				return req
			},
			wantErr:   true,
			errorCode: "InvalidPhone",
		},
		{
			name: "Remission note with invalid municipality",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
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
			name: "Remission note with nil item",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.Items = []*structs.RemissionNoteItemRequest{nil}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note with item without type",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				item.ItemType = nil
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note with item without quantity",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				item.Quantity = nil
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note with item without unit measure",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				item.UnitMeasure = nil
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note with item without unit price",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				item.UnitPrice = nil
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note with item without description",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				empty := ""
				item.Description = &empty
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "Remission note with item without sales (all zero)",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				zero := 0.0
				item.NonSubjectSale = &zero
				item.ExemptSale = &zero
				item.TaxedSale = &zero
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidValue",
		},
		{
			name: "Remission note with negative taxed sale in item",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				negative := -100.0
				item.TaxedSale = &negative
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAmount",
		},

		{
			name: "Remission note with invalid related document type",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				relDoc := structs.RelatedDocRequest{
					DocumentType:   "99",
					GenerationType: 1,
					DocumentNumber: "12345678",
					EmissionDate:   "2023-01-15",
				}
				req.RelatedDocs = []*structs.RelatedDocRequest{&relDoc}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDTEType",
		},
		{
			name: "Remission note with future date in related document",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				relDoc := structs.RelatedDocRequest{
					DocumentType:   "03",
					GenerationType: 1,
					DocumentNumber: "12345678",
					EmissionDate:   "2099-01-15",
				}
				req.RelatedDocs = []*structs.RelatedDocRequest{&relDoc}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDateTime",
		},
		{
			name: "Remission note with invalid date format in related document",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				relDoc := structs.RelatedDocRequest{
					DocumentType:   "03",
					GenerationType: 2,
					DocumentNumber: "0408DCE7-8E96-47AA-92B2-B0F0C8FBDAF3",
					EmissionDate:   "15/01/2023",
				}
				req.RelatedDocs = []*structs.RelatedDocRequest{&relDoc}
				return req
			},
			wantErr:   true,
			errorCode: "ErrorMapping",
		},

		{
			name: "Remission note with invalid extension",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
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
			name: "Remission note with invalid third-party sale",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
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
			name: "Remission note with invalid payment type",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				payment := structs.PaymentRequest{Code: "77", Amount: 100.0}
				req.Summary.Payments = []*structs.PaymentRequest{&payment}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidLength",
		},
		{
			name: "Remission note with invalid appendix (empty field)",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				appendix := structs.AppendixRequest{Field: "", Label: "Etiqueta", Value: "Valor"}
				req.Appendixes = []*structs.AppendixRequest{&appendix}
				return req
			},
			wantErr:   true,
			errorCode: "ErrorMapping",
		},

		{
			name: "Remission note with valid 2-character BienTitulo",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				titulo := "01"
				req.Receiver.BienTitulo = &titulo
				return req
			},
			wantErr: false,
		},
		{
			name: "Remission note with valid email",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				validEmail := "cliente@empresa.com"
				req.Receiver.Email = &validEmail
				return req
			},
			wantErr: false,
		},
		{
			name: "Remission note with valid phone",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				validPhone := "22345678"
				req.Receiver.Phone = &validPhone
				return req
			},
			wantErr: false,
		},
		{
			name: "Remission note with valid address",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
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
			name: "Remission note with valid related document",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				relDoc := structs.RelatedDocRequest{
					DocumentType:   "03",
					GenerationType: 1,
					DocumentNumber: "12345678",
					EmissionDate:   "2023-01-15",
				}
				req.RelatedDocs = []*structs.RelatedDocRequest{&relDoc}
				return req
			},
			wantErr: false,
		},
		{
			name: "Remission note with valid extension",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				observation := "Entrega parcial del pedido"
				req.Extension = &structs.ExtensionRequest{
					DeliveryName:     "Roberto Sánchez",
					DeliveryDocument: "12345678-9",
					ReceiverName:     "Ana Flores",
					ReceiverDocument: "98765432-1",
					Observation:      &observation,
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "Remission note with valid third-party sale",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				req.ThirdPartySale = &structs.ThirdPartySaleRequest{
					NIT:  "06141804941035",
					Name: "Empresa Tercero S.A. de C.V.",
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "Remission note with valid payment type",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				reference := "REF-001"
				payment := structs.PaymentRequest{Code: "01", Amount: 226.0, Reference: &reference}
				req.Summary.Payments = []*structs.PaymentRequest{&payment}
				return req
			},
			wantErr: false,
		},
		{
			name: "Remission note with valid appendix",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				appendix := structs.AppendixRequest{Field: "guia_despacho", Label: "Guía de despacho", Value: "GD-2024-001"}
				req.Appendixes = []*structs.AppendixRequest{&appendix}
				return req
			},
			wantErr: false,
		},
		{
			name: "Remission note with service type item",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				itemType := 2
				item.ItemType = &itemType
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr: false,
		},

		{
			name: "Item with invalid type (99) returns InvalidItemType error",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				invalid := 99
				item.ItemType = &invalid
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidItemType",
		},
		{
			name: "Item with zero type (0) returns InvalidItemType error",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				zero := 0
				item.ItemType = &zero
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidItemType",
		},
		{
			name: "Item with negative quantity returns InvalidQuantity error",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				negative := -5.0
				item.Quantity = &negative
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidQuantity",
		},
		{
			name: "Item with zero quantity returns InvalidQuantity error",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				zero := 0.0
				item.Quantity = &zero
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidQuantity",
		},
		{
			name: "Item with out of range unit measure (100) returns error",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				invalid := 100
				item.UnitMeasure = &invalid
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidNumberRange",
		},
		{
			name: "Item with zero unit measure returns error",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				zero := 0
				item.UnitMeasure = &zero
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidNumberRange",
		},
		{
			name: "Item with negative unit price returns InvalidAmount error",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				negative := -1.0
				item.UnitPrice = &negative
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAmount",
		},
		{
			name: "Item with discount greater than 100% returns InvalidDiscount error",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				over := 200.0
				item.DiscountAmount = &over
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDiscount",
		},
		{
			name: "Item with negative discount returns InvalidDiscount error",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				negative := -10.0
				item.DiscountAmount = &negative
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDiscount",
		},
		{
			name: "Item with exactly 100% discount is valid",
			req: func() *structs.CreateRemissionNoteRequest {
				req := fixtures.CreateDefaultRemissionNoteRequest()
				item := fixtures.CreateDefaultRemissionNoteItem(0)
				full := 100.0
				item.DiscountAmount = &full
				req.Items = []*structs.RemissionNoteItemRequest{item}
				return req
			},
			wantErr: false,
		},
	}

	mapper := request_mapper.NewRemissionNoteMapper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req()

			got, err := mapper.MapToRemissionNoteData(req, issuer)

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
			assert.NotNil(t, got.Receiver)
			assert.Len(t, got.Items, len(req.Items))
			assert.NotNil(t, got.RemissionSummary)

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

			if req.Appendixes != nil {
				assert.NotNil(t, got.Appendixes)
				assert.Len(t, got.Appendixes, len(req.Appendixes))
			}
		})
	}
}
