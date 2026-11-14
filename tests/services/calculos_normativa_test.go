package services

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/ccf"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/fse"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/invoice"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
	test "github.com/chainedpixel/api-facturacion-sv/tests"
	"github.com/chainedpixel/api-facturacion-sv/tests/fixtures"
	"github.com/chainedpixel/api-facturacion-sv/tests/mocks"
)

// Se validan los ejemplos numericos de calculos-dte-v2.md (seccion 5), tomados del Anexo IV de la
// Normativa DTE v2.0. Cada caso recorre el flujo real: request -> mapper -> validadores de dominio.
// Un caso valido que se rechace, o uno invalido que se acepte, indica una regla mal aplicada.

func newControlNumberMock(t *testing.T, controlNumber string) *mocks.MockSequentialNumberManager {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mock := mocks.NewMockSequentialNumberManager(ctrl)
	mock.EXPECT().
		GetNextControlNumber(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(controlNumber, nil).
		AnyTimes()
	return mock
}

func itemCode(code string) *string { return &code }

// --- 5.1 FE (precio con IVA incluido) ---

func guideInvoiceRequest() *structs.CreateInvoiceRequest {
	return &structs.CreateInvoiceRequest{
		ModelType: 1,
		Receiver:  fixtures.CreateDefaultReceiver(),
		Items: []structs.InvoiceItemRequest{
			{
				ItemRequest: structs.ItemRequest{
					Number: 1, Type: 1, Description: "Zapato casual", Quantity: 2,
					UnitMeasure: 59, UnitPrice: 25.00, Discount: 0, Code: itemCode("ZAP-001"),
				},
				TaxedSale: 50.00,
				IVAItem:   5.75221239,
			},
			{
				ItemRequest: structs.ItemRequest{
					Number: 2, Type: 1, Description: "Zapato deportivo", Quantity: 1,
					UnitMeasure: 59, UnitPrice: 30.00, Discount: 3.00, Code: itemCode("ZAP-002"),
				},
				TaxedSale: 27.00,
				IVAItem:   3.10619469,
			},
		},
		Summary: &structs.InvoiceSummaryRequest{
			SummaryRequest: structs.SummaryRequest{
				TotalTaxed:         77.00,
				SubTotalSales:      77.00,
				TotalDiscount:      3.00,
				SubTotal:           77.00,
				TotalOperation:     77.00,
				TotalToPay:         77.00,
				OperationCondition: 1,
				PaymentTypes:       []structs.PaymentRequest{{Code: "01", Amount: 77.00}},
			},
			TotalIVA: 8.86,
		},
	}
}

func TestGuide_FE_Example51(t *testing.T) {
	test.TestMain(t)

	data, err := request_mapper.NewInvoiceMapper().MapToInvoiceData(guideInvoiceRequest(), fixtures.CreateDefaultIssuer())
	require.NoError(t, err)

	service := invoice.NewInvoiceService(newControlNumberMock(t, "DTE-01-M001P001-000000000000001"))
	_, err = service.Create(context.Background(), data, 1)
	require.NoError(t, err, "la FE del ejemplo 5.1 de la guia es valida y no debe rechazarse")
}

// --- 5.2 CCF (precio sin IVA) ---

func guideCCFRequest() *structs.CreateCreditFiscalRequest {
	return &structs.CreateCreditFiscalRequest{
		ModelType: 1,
		Receiver:  fixtures.CreateDefaultReceiverWithoutDocsFields(),
		Items: []structs.CreditItemRequest{
			{
				ItemRequest: structs.ItemRequest{
					Number: 1, Type: 1, Description: "Pan dulce caja", Quantity: 3,
					UnitMeasure: 59, UnitPrice: 20.00, Discount: 0, Code: itemCode("PAN-001"),
					Taxes: []string{"20"},
				},
				TaxedSale: 60.00,
			},
			{
				ItemRequest: structs.ItemRequest{
					Number: 2, Type: 1, Description: "Pastel mediano", Quantity: 1,
					UnitMeasure: 59, UnitPrice: 40.00, Discount: 5.00, Code: itemCode("PAN-002"),
					Taxes: []string{"20"},
				},
				TaxedSale: 35.00,
			},
		},
		Summary: &structs.CreditSummaryRequest{
			SummaryRequest: structs.SummaryRequest{
				TotalTaxed:         95.00,
				SubTotalSales:      95.00,
				TotalDiscount:      5.00,
				SubTotal:           95.00,
				TotalOperation:     107.35,
				TotalToPay:         107.35,
				OperationCondition: 1,
				Taxes:              []structs.TaxRequest{{Code: "20", Description: "Impuesto al Valor Agregado 13%", Value: 12.35}},
				PaymentTypes:       []structs.PaymentRequest{{Code: "01", Amount: 107.35}},
			},
		},
	}
}

func createCCF(t *testing.T, req *structs.CreateCreditFiscalRequest) error {
	t.Helper()
	data, err := request_mapper.NewCCFMapper().MapToCCFData(req, fixtures.CreateDefaultIssuer())
	if err != nil {
		return err
	}
	service := ccf.NewCCFService(newControlNumberMock(t, "DTE-03-M001P001-000000000000001"))
	_, err = service.Create(context.Background(), data, 1)
	return err
}

func TestGuide_CCF_Example52(t *testing.T) {
	test.TestMain(t)

	require.NoError(t, createCCF(t, guideCCFRequest()),
		"el CCF del ejemplo 5.2 de la guia es valido y no debe rechazarse")
}

func TestGuide_CCF_Example52_WithGlobalTaxedDiscount(t *testing.T) {
	test.TestMain(t)

	req := guideCCFRequest()
	req.Summary.TaxedDiscount = 5.00
	req.Summary.TotalDiscount = 10.00
	req.Summary.SubTotal = 90.00
	req.Summary.Taxes[0].Value = 11.70
	req.Summary.TotalOperation = 101.70
	req.Summary.TotalToPay = 101.70
	req.Summary.PaymentTypes[0].Amount = 101.70

	require.NoError(t, createCCF(t, req),
		"el CCF del ejemplo 5.2 con descuento global de 5.00 es valido y no debe rechazarse")
}

func TestGuide_CCF_TotalOperationMustIncludeTaxes(t *testing.T) {
	test.TestMain(t)

	// Campo 150: montoTotalOperacion = subTotal + tributos. Si se omite el IVA (95.00 en vez de 107.35)
	// Hacienda lo rechaza, por lo que el microservicio tambien debe rechazarlo.
	req := guideCCFRequest()
	req.Summary.TotalOperation = 95.00
	req.Summary.TotalToPay = 95.00
	req.Summary.PaymentTypes[0].Amount = 95.00

	require.Error(t, createCCF(t, req),
		"un CCF cuyo monto total de la operacion no incluye los tributos debe rechazarse")
}

// --- 5.4 FSE ---

func TestGuide_FSE_Example54(t *testing.T) {
	test.TestMain(t)

	req := &structs.CreateFSERequest{
		Receiver: fixtures.CreateDefaultFSEReceiver(),
		Items: []structs.FSEItemRequest{
			{
				ItemRequest: structs.ItemRequest{
					Number: 1, Type: 2, Description: "Servicio de limpieza", Quantity: 4,
					UnitMeasure: 59, UnitPrice: 10.00, Discount: 0, Code: itemCode("SRV-001"),
				},
				Purchase: 40.00,
			},
		},
		Summary: &structs.FSESummaryRequest{
			SummaryRequest: structs.SummaryRequest{
				SubTotal:           40.00,
				TotalDiscount:      0,
				TotalToPay:         36.00,
				OperationCondition: 1,
				PaymentTypes:       []structs.PaymentRequest{{Code: "01", Amount: 36.00}},
			},
			TotalPurchase:   40.00,
			IncomeRetention: 4.00,
		},
	}

	data, err := request_mapper.NewFSEMapper().MapToFSEData(req, fixtures.CreateDefaultIssuer())
	require.NoError(t, err)

	service := fse.NewFSEService(newControlNumberMock(t, "DTE-14-M001P001-000000000000001"))
	_, err = service.Create(context.Background(), data, 1)
	require.NoError(t, err, "la FSE del ejemplo 5.4 de la guia es valida y no debe rechazarse")
}

// --- Casos negativos: cada uno viola una regla de la guia y debe rechazarse ---

func createInvoice(t *testing.T, req *structs.CreateInvoiceRequest) error {
	t.Helper()
	data, err := request_mapper.NewInvoiceMapper().MapToInvoiceData(req, fixtures.CreateDefaultIssuer())
	if err != nil {
		return err
	}
	service := invoice.NewInvoiceService(newControlNumberMock(t, "DTE-01-M001P001-000000000000001"))
	_, err = service.Create(context.Background(), data, 1)
	return err
}

func createFSE(t *testing.T, req *structs.CreateFSERequest) error {
	t.Helper()
	data, err := request_mapper.NewFSEMapper().MapToFSEData(req, fixtures.CreateDefaultIssuer())
	if err != nil {
		return err
	}
	service := fse.NewFSEService(newControlNumberMock(t, "DTE-14-M001P001-000000000000001"))
	_, err = service.Create(context.Background(), data, 1)
	return err
}

func guideFSERequest() *structs.CreateFSERequest {
	return &structs.CreateFSERequest{
		Receiver: fixtures.CreateDefaultFSEReceiver(),
		Items: []structs.FSEItemRequest{{
			ItemRequest: structs.ItemRequest{
				Number: 1, Type: 2, Description: "Servicio de limpieza", Quantity: 4,
				UnitMeasure: 59, UnitPrice: 10.00, Code: itemCode("SRV-001"),
			},
			Purchase: 40.00,
		}},
		Summary: &structs.FSESummaryRequest{
			SummaryRequest: structs.SummaryRequest{
				SubTotal: 40.00, TotalToPay: 36.00, OperationCondition: 1,
				PaymentTypes: []structs.PaymentRequest{{Code: "01", Amount: 36.00}},
			},
			TotalPurchase: 40.00, IncomeRetention: 4.00,
		},
	}
}

func TestGuide_NegativeCases(t *testing.T) {
	test.TestMain(t)

	cases := []struct {
		name string
		rule string
		run  func(t *testing.T) error
	}{
		{
			name: "FE con tributo 20 en el resumen",
			rule: "2.6: en FE solo secciones 1 y 3 de CAT-015, excepto 20 y C3",
			run: func(t *testing.T) error {
				req := guideInvoiceRequest()
				req.Summary.Taxes = []structs.TaxRequest{{Code: "20", Description: "IVA", Value: 10.01}}
				req.Summary.TotalOperation = 87.01
				req.Summary.TotalToPay = 87.01
				req.Summary.PaymentTypes[0].Amount = 87.01
				return createInvoice(t, req)
			},
		},
		{
			name: "FE con tributo 20 en el item",
			rule: "2.6: en FE solo secciones 1 y 3 de CAT-015, excepto 20 y C3",
			run: func(t *testing.T) error {
				req := guideInvoiceRequest()
				req.Items[0].Type = 2
				req.Items[0].Taxes = []string{"20"}
				return createInvoice(t, req)
			},
		},
		{
			name: "FE con totalIva calculado sin dividir entre 1.13 (77 x 13% = 10.01)",
			rule: "2.3/3.7: totalIva = suma de ivaItem = (ventaGravada / 1.13) x 13%",
			run: func(t *testing.T) error {
				req := guideInvoiceRequest()
				req.Summary.TotalIVA = 10.01
				return createInvoice(t, req)
			},
		},
		{
			name: "FE con montoTotalOperacion distinto de subTotal + tributos",
			rule: "3.8: montoTotalOperacion = subTotal + tributos",
			run: func(t *testing.T) error {
				req := guideInvoiceRequest()
				req.Summary.TotalOperation = 80.00
				req.Summary.TotalToPay = 80.00
				req.Summary.PaymentTypes[0].Amount = 80.00
				return createInvoice(t, req)
			},
		},
		{
			name: "FE con subTotalVentas distinto de la suma de operaciones",
			rule: "3.3: subTotalVentas = totalNoSuj + totalExenta + totalGravada",
			run: func(t *testing.T) error {
				req := guideInvoiceRequest()
				req.Summary.SubTotalSales = 70.00
				return createInvoice(t, req)
			},
		},
		{
			name: "CCF con ventaGravada distinta de precio x cantidad - descuento",
			rule: "2.1: ventaGravada = (precioUni x cantidad) - montoDescu",
			run: func(t *testing.T) error {
				req := guideCCFRequest()
				req.Items[1].TaxedSale = 40.00 // deberia ser 35.00
				req.Summary.TotalTaxed = 100.00
				req.Summary.SubTotalSales = 100.00
				req.Summary.SubTotal = 100.00
				req.Summary.Taxes[0].Value = 13.00
				req.Summary.TotalOperation = 113.00
				req.Summary.TotalToPay = 113.00
				req.Summary.PaymentTypes[0].Amount = 113.00
				return createCCF(t, req)
			},
		},
		{
			name: "CCF con IVA del resumen calculado sin restar el descuento global",
			rule: "3.6: tributo 20 = (totalGravada - descuGravada) x 13%",
			run: func(t *testing.T) error {
				req := guideCCFRequest()
				req.Summary.TaxedDiscount = 5.00
				req.Summary.TotalDiscount = 10.00
				req.Summary.SubTotal = 90.00
				req.Summary.TotalOperation = 102.35
				req.Summary.TotalToPay = 102.35
				req.Summary.PaymentTypes[0].Amount = 102.35
				return createCCF(t, req)
			},
		},
		{
			name: "CCF con suma de pagos distinta del total a pagar",
			rule: "3.12: condicion 1, la suma de montoPago debe ser igual a totalPagar",
			run: func(t *testing.T) error {
				req := guideCCFRequest()
				req.Summary.PaymentTypes[0].Amount = 100.00
				return createCCF(t, req)
			},
		},
		{
			name: "FSE con totalPagar distinto de subTotal - reteRenta",
			rule: "3.10: FSE totalPagar = subTotal - reteRenta",
			run: func(t *testing.T) error {
				req := guideFSERequest()
				req.Summary.TotalToPay = 40.00
				req.Summary.PaymentTypes[0].Amount = 40.00
				return createFSE(t, req)
			},
		},
		{
			name: "FSE con compra distinta de precio x cantidad - descuento",
			rule: "2.1: compra = (precioUni x cantidad) - montoDescu",
			run: func(t *testing.T) error {
				req := guideFSERequest()
				req.Items[0].Purchase = 45.00
				req.Summary.TotalPurchase = 45.00
				req.Summary.SubTotal = 45.00
				req.Summary.TotalToPay = 41.00
				req.Summary.PaymentTypes[0].Amount = 41.00
				return createFSE(t, req)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, tc.run(t), "debe rechazarse (regla %s)", tc.rule)
		})
	}
}
