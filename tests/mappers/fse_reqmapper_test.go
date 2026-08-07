package mappers

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestMapToFSEData(t *testing.T) {
	test.TestMain(t)

	issuer := fixtures.CreateDefaultIssuer()

	tests := []struct {
		name      string
		req       func() *structs.CreateFSERequest
		wantErr   bool
		errorCode string
	}{
		{
			name: "Valid FSE request",
			req: func() *structs.CreateFSERequest {
				return fixtures.CreateDefaultFSERequest()
			},
			wantErr: false,
		},
		{
			name: "FSE with all optional fields",
			req: func() *structs.CreateFSERequest {
				return fixtures.CreateFSERequestWithAllOptionalFields()
			},
			wantErr: false,
		},

		{
			name: "Null FSE request",
			req: func() *structs.CreateFSERequest {
				return nil
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "FSE without items",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Items = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "FSE without summary",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Summary = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "FSE without receiver",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Receiver = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "FSE with empty document type in receiver",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Receiver.DocumentType = ""
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "FSE with empty document number in receiver",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Receiver.DocumentNumber = ""
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},

		{
			name: "FSE without receiver name",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Receiver.Name = nil
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "FSE without receiver address",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Receiver.Address = nil
				return req
			},
			wantErr: false,
		},
		{
			name: "FSE with invalid email in receiver",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				invalidEmail := "no-es-un-email"
				req.Receiver.Email = &invalidEmail
				return req
			},
			wantErr:   true,
			errorCode: "InvalidEmail",
		},
		{
			name: "FSE with invalid phone in receiver",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				invalidPhone := "123"
				req.Receiver.Phone = &invalidPhone
				return req
			},
			wantErr:   true,
			errorCode: "InvalidPhone",
		},
		{
			name: "FSE with invalid municipality in receiver address",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
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
			name: "FSE with invalid item type",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				item := fixtures.CreateDefaultFSEItem(0)
				item.Type = 99
				req.Items = []structs.FSEItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidItemType",
		},
		{
			name: "FSE with negative quantity in item",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				item := fixtures.CreateDefaultFSEItem(0)
				item.Quantity = -1
				req.Items = []structs.FSEItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidQuantity",
		},
		{
			name: "FSE with invalid unit measure",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				item := fixtures.CreateDefaultFSEItem(0)
				item.UnitMeasure = 0
				req.Items = []structs.FSEItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidNumberRange",
		},
		{
			name: "FSE with negative unit price",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				item := fixtures.CreateDefaultFSEItem(0)
				item.UnitPrice = -10.0
				req.Items = []structs.FSEItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAmount",
		},
		{
			name: "FSE with negative discount in item",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				item := fixtures.CreateDefaultFSEItem(0)
				item.Discount = -5.0
				req.Items = []structs.FSEItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDiscount",
		},
		{
			name: "FSE with discount greater than 100% in item",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				item := fixtures.CreateDefaultFSEItem(0)
				item.Discount = 150.0
				req.Items = []structs.FSEItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidDiscount",
		},
		{
			name: "FSE with zero purchase amount in item",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				item := fixtures.CreateDefaultFSEItem(0)
				item.Purchase = 0
				req.Items = []structs.FSEItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "RequiredField",
		},
		{
			name: "FSE with negative purchase amount in item",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				item := fixtures.CreateDefaultFSEItem(0)
				item.Purchase = -50.0
				req.Items = []structs.FSEItemRequest{item}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAmount",
		},

		{
			name: "FSE with invalid extension (empty deliverer name)",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
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
			name: "FSE with invalid appendix (empty field)",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Appendixes = []structs.AppendixRequest{
					{Field: "", Label: "Etiqueta", Value: "Valor"},
				}
				return req
			},
			wantErr: true,
		},
		{
			name: "FSE with appendix label too short",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Appendixes = []structs.AppendixRequest{
					{Field: "campo", Label: "ab", Value: "Valor"},
				}
				return req
			},
			wantErr:   true,
			errorCode: "InvalidAppendixLabel",
		},

		{
			name: "FSE with valid email in receiver",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				validEmail := "sujeto@example.com"
				req.Receiver.Email = &validEmail
				return req
			},
			wantErr: false,
		},
		{
			name: "FSE with valid phone in receiver",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				validPhone := "22345678"
				req.Receiver.Phone = &validPhone
				return req
			},
			wantErr: false,
		},
		{
			name: "FSE with NIT document type (13)",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Receiver.DocumentType = "13"
				req.Receiver.DocumentNumber = "00000000-0"
				return req
			},
			wantErr: false,
		},
		{
			name: "FSE with service type item",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				item := fixtures.CreateDefaultFSEItem(0)
				item.Type = 2
				req.Items = []structs.FSEItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "FSE with zero discount in item",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				item := fixtures.CreateDefaultFSEItem(0)
				item.Discount = 0
				req.Items = []structs.FSEItemRequest{item}
				return req
			},
			wantErr: false,
		},
		{
			name: "FSE with valid extension",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				observation := "Observación válida"
				req.Extension = &structs.ExtensionRequest{
					DeliveryName:     "Pedro González",
					DeliveryDocument: "12345678-9",
					ReceiverName:     "María Flores",
					ReceiverDocument: "98765432-1",
					Observation:      &observation,
				}
				return req
			},
			wantErr: false,
		},
		{
			name: "FSE with valid appendix",
			req: func() *structs.CreateFSERequest {
				req := fixtures.CreateDefaultFSERequest()
				req.Appendixes = []structs.AppendixRequest{
					{Field: "referencia", Label: "Referencia interna", Value: "REF-001"},
				}
				return req
			},
			wantErr: false,
		},
	}

	mapper := request_mapper.NewFSEMapper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req()

			got, err := mapper.MapToFSEData(req, issuer)

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
			assert.NotNil(t, got.FSEReceiver)
			assert.Len(t, got.Items, len(req.Items))
			assert.NotNil(t, got.FSESummary)

			if req.Extension != nil {
				assert.NotNil(t, got.Extension)
			}

			if req.Appendixes != nil {
				assert.NotNil(t, got.Appendixes)
				assert.Len(t, got.Appendixes, len(req.Appendixes))
			}
		})
	}
}
