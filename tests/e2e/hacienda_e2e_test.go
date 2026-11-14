//go:build e2e

package e2e

import (
	"os"
	"testing"

	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
)

// Suite E2E: emite documentos reales contra el ambiente de pruebas de Hacienda a traves de la API.
// Requisitos: API corriendo (E2E_API_URL), firmador disponible y credenciales E2E_* en .env.
// Ejecucion: go test -tags e2e ./tests/e2e/ -v -count=1
//
// Cada caso pasa si Hacienda otorga sello de recepcion. Un rechazo muestra el codigo y las
// observaciones de Hacienda, que es justamente lo que se busca descubrir.

// contributorReceiver: receptor contribuyente registrado en Hacienda (E2E_RECEPTOR_* en .env).
// Hacienda valida que el NIT exista ("NIT CONTRIBUYENTE NO EXISTE"), por lo que no sirven NIT ficticios.
func contributorReceiver() *structs.ReceiverRequest {
	return &structs.ReceiverRequest{
		Name: str(os.Getenv("E2E_RECEPTOR_NAME")), NRC: str(os.Getenv("E2E_RECEPTOR_NRC")),
		ActivityCode: str(os.Getenv("E2E_RECEPTOR_ACTIVITY")), ActivityDesc: str(os.Getenv("E2E_RECEPTOR_ACTIVITY_DESC")),
		Phone: str("22123456"), Email: str("receptor.e2e@example.com"),
		Address: &structs.AddressRequest{Department: "07", Municipality: "18", District: "01",
			Complement: "Avenida Jose Maria Rivas #25, Cojutepeque"},
	}
}

// companyReceiver: receptor de FE identificado por tipo y numero de documento (NIT, tipo 36)
func companyReceiver() *structs.ReceiverRequest {
	r := contributorReceiver()
	r.DocumentType, r.DocumentNumber = str("36"), str(os.Getenv("E2E_RECEPTOR_NIT"))
	return r
}

// personReceiver: persona natural sin NRC, identificada con DUI (CAT-022 codigo 13)
func personReceiver() *structs.ReceiverRequest {
	return &structs.ReceiverRequest{
		DocumentType: str("13"), DocumentNumber: str("04567890-3"), Name: str("Maria Fernanda Lopez"),
		Phone: str("71234567"), Email: str("maria.lopez@example.com"),
		Address: &structs.AddressRequest{Department: "06", Municipality: "20", District: "01",
			Complement: "Colonia Escalon, Pasaje 3 #12, San Salvador"},
	}
}

// ccfReceiver: el receptor de CCF se identifica solo por NIT
func ccfReceiver() *structs.ReceiverRequest {
	r := contributorReceiver()
	r.NIT = str(os.Getenv("E2E_RECEPTOR_NIT"))
	return r
}

// adjustmentReceiver: NC y ND reciben NIT (la API lo compara con el receptor del CCF relacionado)
// ademas de tipo y numero de documento, que es lo que se envia a Hacienda
func adjustmentReceiver() *structs.ReceiverRequest {
	return remissionReceiver()
}

// remissionReceiver: NR exige tipo/numero de documento y, para tipo 36, tambien el NIT
func remissionReceiver() *structs.ReceiverRequest {
	r := companyReceiver()
	r.NIT = str(os.Getenv("E2E_RECEPTOR_NIT"))
	return r
}

func excludedSubject() *structs.FSEReceiverRequest {
	return &structs.FSEReceiverRequest{
		ReceiverRequest: structs.ReceiverRequest{
			Name: str("Juan Carlos Perez"), Phone: str("22345678"), Email: str("juan@example.com"),
			Address: &structs.AddressRequest{Department: "06", Municipality: "20", District: "01",
				Complement: "Colonia Escalon, Calle El Mirador #45, San Salvador"},
		},
		DocumentType: "13", DocumentNumber: "00000000-0",
		ActivityCode: str("46900"), ActivityDescription: str("Servicios profesionales"),
	}
}

// thirdParty: venta por cuenta de terceros; el NIT debe existir en Hacienda (otro usuario de pruebas)
func thirdParty() *structs.ThirdPartySaleRequest {
	return &structs.ThirdPartySaleRequest{NIT: "07023108921027", Name: "Tercero de Pruebas E2E", FiscalDomicile: 1}
}

func shoe(desc string, qty, price, discount float64) itemSpec {
	return itemSpec{description: desc, itemType: 1, quantity: qty, unitPrice: price, discount: discount, kind: taxedSale}
}

func bread(desc string, qty, price, discount float64) itemSpec {
	return itemSpec{description: desc, itemType: 1, quantity: qty, unitPrice: price, discount: discount, kind: taxedSale}
}

func exempt(desc string, qty, price float64) itemSpec {
	return itemSpec{description: desc, itemType: 2, quantity: qty, unitPrice: price, kind: exemptSale}
}

func nonSubject(desc string, qty, price float64) itemSpec {
	return itemSpec{description: desc, itemType: 1, quantity: qty, unitPrice: price, kind: nonSubjectSale}
}

// charge: cargo que no afecta la base imponible (noGravado), por ejemplo un deposito o un envio
func charge(desc string, amount float64) itemSpec {
	return itemSpec{description: desc, itemType: 2, kind: nonTaxedCharge, charge: amount}
}

func manyItems(n int) []itemSpec {
	items := make([]itemSpec, 0, n)
	for i := 0; i < n; i++ {
		price := 2.15 + float64(i)*1.37
		discount := 0.0
		if i%3 == 0 {
			discount = 0.45
		}
		items = append(items, shoe("Articulo surtido", float64(1+i%4), price, discount))
	}
	return items
}

func remissionFor(receiver *structs.ReceiverRequest) *structs.CreateRemissionNoteRequest {
	return buildRemissionNote(&structs.RemissionNoteReceiverRequest{ReceiverRequest: receiver, BienTitulo: str("04")},
		[]itemSpec{shoe("Mercaderia en consignacion", 10, 25.00, 0)})
}

// ======================= FE =======================

func TestE2E_FacturaElectronica(t *testing.T) {
	client := newAPIClient(t)

	// NR previa para relacionarla desde una FE (Anexo IV: FE que relaciona NRE en el mismo periodo)
	var remission *emittedDTE
	t.Run("NR previa para relacionar", func(t *testing.T) {
		remission = client.emit(t, "/dte/remissionnote", remissionFor(remissionReceiver()))
	})

	cases := []struct {
		name string
		spec invoiceSpec
	}{
		{"venta gravada simple", invoiceSpec{receiver: companyReceiver(), items: []itemSpec{
			shoe("Zapato casual cafe talla 40", 1, 45.00, 0),
		}}},
		{"consumidor final sin datos del receptor (bajo el monto legal)", invoiceSpec{items: []itemSpec{
			bread("Pan dulce surtido", 6, 0.50, 0),
		}}},
		{"persona natural con DUI", invoiceSpec{receiver: personReceiver(), items: []itemSpec{
			shoe("Sandalia de playa", 1, 18.00, 0),
		}}},
		{"guia 5.1: descuento por item", invoiceSpec{receiver: companyReceiver(), items: []itemSpec{
			shoe("Zapato casual", 2, 25.00, 0),
			shoe("Zapato deportivo", 1, 30.00, 3.00),
		}}},
		{"descuento global a ventas gravadas", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{
				shoe("Bota de cuero", 1, 89.99, 0),
				shoe("Sandalia", 2, 19.50, 1.50),
				shoe("Calcetas paquete", 3, 4.25, 0),
			},
			global: globalDiscounts{taxed: 10.00}}},
		{"descuento por item y global combinados", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{
				bread("Pan frances docena", 4, 1.75, 0.35),
				bread("Pastel de tres leches", 1, 22.00, 2.20),
				bread("Semita grande", 2, 3.60, 0),
			},
			global: globalDiscounts{taxed: 2.50}}},
		{"ventas mixtas: gravada, exenta y no sujeta", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{
				shoe("Zapato escolar", 2, 32.00, 2.00),
				exempt("Servicio de reparacion exento", 1, 15.00),
				nonSubject("Venta no sujeta", 1, 5.00),
			}}},
		{"ventas mixtas con descuentos globales en las tres categorias", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{
				shoe("Zapato formal", 1, 65.00, 0),
				exempt("Servicio exento", 2, 12.00),
				nonSubject("Venta no sujeta", 1, 8.00),
			},
			global: globalDiscounts{taxed: 5.00, exempt: 2.00, nonSubject: 1.00}}},
		{"redondeo a 8 decimales con precios no exactos", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{
				bread("Galleta artesanal", 7, 3.17, 0),
				bread("Dona glaseada", 13, 0.87, 0.11),
				bread("Torta de cumpleanos", 3, 19.99, 1.37),
			}}},
		{"cantidades con decimales (pan por libra)", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{
				bread("Pan de molde por libra", 2.75, 1.89, 0),
				bread("Queso duro por libra", 1.333, 4.15, 0.05),
			}}},
		{"cargo que no afecta la base imponible (envio)", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{
				shoe("Zapato deportivo", 1, 60.00, 0),
				charge("Cargo por envio a domicilio", 3.50),
			}}},
		{"dos formas de pago: efectivo y tarjeta de debito", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Zapato de vestir", 2, 48.75, 0)},
			opts:  documentOptions{paymentCodes: []string{"01", "02"}}}},
		{"tres formas de pago: efectivo, credito y transferencia", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Botas industriales", 3, 67.10, 1.30)},
			opts:  documentOptions{paymentCodes: []string{"01", "03", "05"}}}},
		{"condicion credito a 30 dias", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Lote de zapatos para tienda", 10, 28.75, 12.50)},
			opts:  documentOptions{credit: &creditTerms{period: 30, term: "01"}}}},
		{"venta por cuenta de terceros", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Zapato en consignacion", 2, 35.00, 0)},
			opts:  documentOptions{thirdParty: thirdParty()}}},
		{"otros documentos asociados", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Zapato ortopedico", 1, 75.00, 0)},
			opts: documentOptions{otherDocs: []structs.OtherDocRequest{
				{DocumentCode: 1, Description: str("Orden de compra"), Detail: str("OC-2026-0045")},
			}}}},
		{"con apendice", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Zapato casual", 1, 40.00, 0)},
			opts: documentOptions{appendixes: []structs.AppendixRequest{
				{Field: "vendedor", Label: "Vendedor", Value: "Ana Martinez"},
				{Field: "sucursal", Label: "Sucursal", Value: "Cojutepeque centro"},
			}}}},
		{"monto mayor a 1095 (ya no exige extension)", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Mayoreo de calzado", 40, 37.50, 0)}}},
		{"25 items con descuentos variados", invoiceSpec{receiver: companyReceiver(), items: manyItems(25)}},
		{"retencion de IVA 1% (VERIFICAR base en FE)", invoiceSpec{receiver: companyReceiver(),
			items:        []itemSpec{shoe("Venta a gran contribuyente", 5, 120.00, 0)},
			ivaRetention: true}},
		{"retencion de IVA con descuento global", invoiceSpec{receiver: companyReceiver(),
			items:        []itemSpec{shoe("Venta a gran contribuyente", 4, 150.00, 0)},
			global:       globalDiscounts{taxed: 20.00},
			ivaRetention: true}},
		{"tributo adicional turismo 59 (exploratorio)", invoiceSpec{receiver: companyReceiver(),
			items:    []itemSpec{shoe("Servicio turistico gravado", 1, 100.00, 0)},
			extraTax: &extraTax{code: "59", description: "Turismo: por alojamiento (5%)", rate: 0.05}}},
		{"con observaciones", invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Zapato de vestir", 1, 55.00, 5.00)},
			opts:  documentOptions{observation: "Venta de prueba E2E normativa v2.0"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client.emit(t, "/dte/invoices", buildInvoice(tc.spec))
		})
	}

	t.Run("con documento relacionado (NR)", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		client.emit(t, "/dte/invoices", buildInvoice(invoiceSpec{receiver: companyReceiver(),
			items: []itemSpec{shoe("Venta de mercaderia remitida", 10, 25.00, 0)},
			opts:  documentOptions{related: relatedRemission(remission)}}))
	})
}

func relatedRemission(remission *emittedDTE) []structs.RelatedDocRequest {
	return []structs.RelatedDocRequest{{
		DocumentType: "04", GenerationType: 2,
		DocumentNumber: remission.GenerationCode, EmissionDate: remission.EmissionDate,
	}}
}

// ======================= CCF =======================

func TestE2E_CreditoFiscal(t *testing.T) {
	client := newAPIClient(t)

	var remission *emittedDTE
	t.Run("NR previa para relacionar", func(t *testing.T) {
		remission = client.emit(t, "/dte/remissionnote", remissionFor(remissionReceiver()))
	})

	cases := []struct {
		name string
		spec ccfSpec
	}{
		{"guia 5.2", ccfSpec{receiver: ccfReceiver(), items: []itemSpec{
			bread("Pan dulce caja", 3, 20.00, 0), bread("Pastel mediano", 1, 40.00, 5.00),
		}}},
		{"guia 5.2 con descuento global a gravadas", ccfSpec{receiver: ccfReceiver(),
			items:  []itemSpec{bread("Pan dulce caja", 3, 20.00, 0), bread("Pastel mediano", 1, 40.00, 5.00)},
			global: globalDiscounts{taxed: 5.00}}},
		{"descuento por item y global con precios no exactos", ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{
				shoe("Zapato industrial", 6, 41.37, 3.33),
				shoe("Plantilla ortopedica", 11, 7.89, 0.79),
				shoe("Cordones", 25, 0.65, 0),
			},
			global: globalDiscounts{taxed: 7.77}}},
		{"ventas mixtas gravada y exenta", ccfSpec{receiver: ccfReceiver(), items: []itemSpec{
			shoe("Zapato de seguridad", 4, 52.00, 4.00), exempt("Servicio exento", 1, 30.00),
		}}},
		{"ventas mixtas con descuentos globales en las tres categorias", ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{
				shoe("Zapato formal", 2, 70.00, 0), exempt("Servicio exento", 1, 40.00), nonSubject("Venta no sujeta", 1, 20.00),
			},
			global: globalDiscounts{taxed: 10.00, exempt: 4.00, nonSubject: 2.00}}},
		{"cantidades con decimales", ccfSpec{receiver: ccfReceiver(), items: []itemSpec{
			bread("Harina por quintal", 2.5, 38.40, 0), bread("Azucar por libra", 12.75, 0.62, 0.10),
		}}},
		{"con percepcion de IVA 1%", ccfSpec{receiver: ccfReceiver(),
			items:         []itemSpec{shoe("Venta mayorista", 20, 35.00, 0)},
			ivaPerception: true}},
		{"percepcion con descuento global (VERIFICAR base)", ccfSpec{receiver: ccfReceiver(),
			items:         []itemSpec{shoe("Venta mayorista", 20, 35.00, 0)},
			global:        globalDiscounts{taxed: 50.00},
			ivaPerception: true}},
		{"con retencion de IVA 1%", ccfSpec{receiver: ccfReceiver(),
			items:        []itemSpec{shoe("Venta a gran contribuyente", 15, 42.00, 0)},
			ivaRetention: true}},
		{"retencion con descuento global", ccfSpec{receiver: ccfReceiver(),
			items:        []itemSpec{shoe("Venta a gran contribuyente", 15, 42.00, 0)},
			global:       globalDiscounts{taxed: 30.00},
			ivaRetention: true}},
		{"percepcion y retencion juntas", ccfSpec{receiver: ccfReceiver(),
			items:         []itemSpec{shoe("Venta mayorista especial", 10, 80.00, 0)},
			ivaPerception: true, ivaRetention: true}},
		{"cargo que no afecta la base imponible", ccfSpec{receiver: ccfReceiver(), items: []itemSpec{
			bread("Pedido de pan para cafeteria", 100, 0.45, 0), charge("Deposito por canastas retornables", 12.00),
		}}},
		{"dos formas de pago", ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{shoe("Zapato de seguridad", 5, 55.00, 0)},
			opts:  documentOptions{paymentCodes: []string{"01", "05"}}}},
		{"condicion credito", ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{bread("Pedido semanal de pan", 50, 1.10, 2.50)},
			opts:  documentOptions{credit: &creditTerms{period: 15, term: "01"}}}},
		{"venta por cuenta de terceros", ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{shoe("Calzado en consignacion", 6, 33.00, 0)},
			opts:  documentOptions{thirdParty: thirdParty()}}},
		{"con apendice y observaciones", ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{shoe("Zapato casual", 3, 29.90, 0)},
			opts: documentOptions{observation: "CCF de prueba E2E",
				appendixes: []structs.AppendixRequest{{Field: "pedido", Label: "Pedido", Value: "PED-778"}}}}},
		{"monto mayor a 1095", ccfSpec{receiver: ccfReceiver(), items: []itemSpec{shoe("Mayoreo de calzado", 60, 35.00, 0)}}},
		{"25 items con descuentos variados", ccfSpec{receiver: ccfReceiver(), items: manyItems(25)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client.emit(t, "/dte/ccf", buildCCF(tc.spec))
		})
	}

	t.Run("con documento relacionado (NR)", func(t *testing.T) {
		if remission == nil {
			t.Skip("la NR previa no obtuvo sello")
		}
		client.emit(t, "/dte/ccf", buildCCF(ccfSpec{receiver: ccfReceiver(),
			items: []itemSpec{shoe("Venta de mercaderia remitida", 10, 25.00, 0)},
			opts:  documentOptions{related: relatedRemission(remission)}}))
	})
}

// ======================= NC y ND =======================

// emitBase emite el CCF sobre el que se aplicaran las notas; detiene el subtest si no obtiene sello
func emitBase(t *testing.T, client *apiClient, spec ccfSpec) *emittedDTE {
	t.Helper()
	base := client.emit(t, "/dte/ccf", buildCCF(spec))
	if base == nil || base.GenerationCode == "" {
		t.Fatal("el CCF base no obtuvo sello; se omiten sus notas")
	}
	return base
}

func TestE2E_NotasCreditoYDebito(t *testing.T) {
	client := newAPIClient(t)

	t.Run("sobre CCF simple", func(t *testing.T) {
		base := emitBase(t, client, ccfSpec{receiver: ccfReceiver(), items: []itemSpec{
			bread("Pan dulce caja", 3, 20.00, 0), bread("Pastel mediano", 1, 40.00, 5.00),
		}})
		t.Run("NC guia 5.3: devolucion de 1 unidad", func(t *testing.T) {
			client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				items: []itemSpec{bread("Devolucion pan dulce caja", 1, 20.00, 0)}}))
		})
		t.Run("NC rebaja con descuento por item", func(t *testing.T) {
			client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				items: []itemSpec{bread("Rebaja pastel mediano", 1, 10.00, 1.25)}}))
		})
		t.Run("NC con dos items", func(t *testing.T) {
			client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				items: []itemSpec{bread("Devolucion pan", 1, 5.00, 0), bread("Bonificacion pastel", 1, 3.00, 0)}}))
		})
		t.Run("ND cargo adicional", func(t *testing.T) {
			client.emit(t, "/dte/debitnote", buildDebitNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				items: []itemSpec{{description: "Cargo por entrega a domicilio", itemType: 2, quantity: 1, unitPrice: 10.00, kind: taxedSale}}}))
		})
		t.Run("ND con dos items y descuento", func(t *testing.T) {
			client.emit(t, "/dte/debitnote", buildDebitNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				items: []itemSpec{bread("Diferencia de precio pan", 3, 1.50, 0), bread("Ajuste de empaque", 2, 2.25, 0.50)}}))
		})
		t.Run("NC que excede las ventas del CCF debe rechazarse", func(t *testing.T) {
			// Anexo IV: la suma de las notas sobre un CCF no puede exceder sus ventas gravadas
			client.expectRejected(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				items: []itemSpec{bread("Devolucion excesiva", 1, 500.00, 0)}}), "")
		})
	})

	t.Run("sobre CCF con percepcion", func(t *testing.T) {
		base := emitBase(t, client, ccfSpec{receiver: ccfReceiver(), ivaPerception: true,
			items: []itemSpec{shoe("Venta mayorista", 20, 35.00, 0)}})
		t.Run("NC con ajuste de percepcion", func(t *testing.T) {
			client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				ivaPerception: true, items: []itemSpec{shoe("Devolucion de 2 pares", 2, 35.00, 0)}}))
		})
		t.Run("ND con ajuste de percepcion", func(t *testing.T) {
			client.emit(t, "/dte/debitnote", buildDebitNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				ivaPerception: true, items: []itemSpec{shoe("Diferencia de precio", 20, 1.00, 0)}}))
		})
	})

	t.Run("sobre CCF con retencion", func(t *testing.T) {
		base := emitBase(t, client, ccfSpec{receiver: ccfReceiver(), ivaRetention: true,
			items: []itemSpec{shoe("Venta a gran contribuyente", 15, 42.00, 0)}})
		t.Run("NC con ajuste de retencion", func(t *testing.T) {
			client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				ivaRetention: true, items: []itemSpec{shoe("Devolucion de 1 par", 1, 42.00, 0)}}))
		})
	})

	t.Run("sobre CCF con ventas exentas", func(t *testing.T) {
		base := emitBase(t, client, ccfSpec{receiver: ccfReceiver(), items: []itemSpec{
			shoe("Zapato de seguridad", 4, 52.00, 0), exempt("Servicio exento", 2, 30.00),
		}})
		t.Run("NC sobre la venta exenta", func(t *testing.T) {
			client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				items: []itemSpec{exempt("Devolucion servicio exento", 1, 30.00)}}))
		})
		t.Run("NC mixta gravada y exenta", func(t *testing.T) {
			client.emit(t, "/dte/creditnote", buildCreditNote(adjustmentSpec{receiver: adjustmentReceiver(), related: base,
				items: []itemSpec{shoe("Devolucion 1 par", 1, 52.00, 0), exempt("Rebaja servicio", 1, 10.00)}}))
		})
	})
}

// ======================= NR =======================

func TestE2E_NotaRemision(t *testing.T) {
	client := newAPIClient(t)

	receiver := &structs.RemissionNoteReceiverRequest{ReceiverRequest: remissionReceiver(), BienTitulo: str("04")}
	t.Run("traslado de mercaderia", func(t *testing.T) {
		client.emit(t, "/dte/remissionnote", buildRemissionNote(receiver, []itemSpec{
			shoe("Zapato casual para sucursal", 12, 25.00, 0),
			shoe("Bota de cuero para sucursal", 6, 60.00, 0),
		}))
	})
	t.Run("traslado con descuento por item y precios no exactos", func(t *testing.T) {
		client.emit(t, "/dte/remissionnote", buildRemissionNote(receiver, []itemSpec{
			shoe("Sandalia", 15, 13.37, 2.10),
			shoe("Pantufla", 9, 8.99, 0),
		}))
	})
	t.Run("bienes en consignacion (titulo 03)", func(t *testing.T) {
		consignment := &structs.RemissionNoteReceiverRequest{ReceiverRequest: remissionReceiver(), BienTitulo: str("03")}
		client.emit(t, "/dte/remissionnote", buildRemissionNote(consignment, []itemSpec{
			shoe("Zapato en consignacion", 20, 30.00, 0),
		}))
	})
}

// ======================= FSE =======================

func TestE2E_SujetoExcluido(t *testing.T) {
	client := newAPIClient(t)

	cases := []struct {
		name string
		spec fseSpec
	}{
		{"guia 5.4: retencion de renta 10%", fseSpec{receiver: excludedSubject(),
			items:           []itemSpec{{description: "Servicio de limpieza", itemType: 2, quantity: 4, unitPrice: 10.00}},
			incomeRetention: 0.10}},
		{"sin retencion, con descuento por item y global", fseSpec{receiver: excludedSubject(),
			items: []itemSpec{
				{description: "Compra de harina a productor", itemType: 1, quantity: 10, unitPrice: 18.50, discount: 5.00},
				{description: "Compra de huevos a productor", itemType: 1, quantity: 30, unitPrice: 0.15},
			},
			globalDiscount: 3.00}},
		{"cantidades con decimales y precios no exactos", fseSpec{receiver: excludedSubject(),
			items: []itemSpec{{description: "Compra de leche por litro", itemType: 1, quantity: 37.5, unitPrice: 0.83}}}},
		{"dos formas de pago", fseSpec{receiver: excludedSubject(),
			items:        []itemSpec{{description: "Servicio de reparacion de horno", itemType: 2, quantity: 1, unitPrice: 85.00}},
			paymentCodes: []string{"01", "05"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client.emit(t, "/dte/fse", buildFSE(tc.spec))
		})
	}
}
