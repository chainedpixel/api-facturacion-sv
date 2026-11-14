//go:build e2e

package e2e

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// Calculadora de la suite: arma cada request a partir de los items aplicando las formulas de
// calculos-dte-v2.md (Anexo IV de la Normativa DTE v2.0). Cuerpo con 8 decimales y resumen con 2,
// calculando el resumen desde los valores ya redondeados de los items (regla 7.3).

type saleKind int

const (
	taxedSale saleKind = iota
	exemptSale
	nonSubjectSale
	nonTaxedCharge // cargo que no afecta la base imponible (noGravado): precio 0 y unidad de medida 99
)

type itemSpec struct {
	description string
	itemType    int // CAT-011: 1 bienes, 2 servicios
	quantity    float64
	unitPrice   float64
	discount    float64 // montoDescu: monto, no porcentaje
	kind        saleKind
	charge      float64 // monto del cargo cuando kind = nonTaxedCharge
	unitMeasure int     // CAT-014; 59 (unidad) si no se indica
}

type globalDiscounts struct {
	nonSubject float64
	exempt     float64
	taxed      float64
}

type creditTerms struct {
	period int    // plazo
	term   string // CAT-018: 01 dias, 02 meses, 03 anios
}

// extraTax es un tributo adicional del resumen (por ejemplo 59, turismo 5% sobre ventas gravadas)
type extraTax struct {
	code        string
	description string
	rate        float64
}

// documentOptions agrupa las secciones opcionales comunes a FE y CCF
type documentOptions struct {
	credit       *creditTerms
	paymentCodes []string // varias formas de pago (CAT-017); el total se reparte entre ellas
	thirdParty   *structs.ThirdPartySaleRequest
	related      []structs.RelatedDocRequest // los items se vinculan al primer documento relacionado
	otherDocs    []structs.OtherDocRequest
	appendixes   []structs.AppendixRequest
	observation  string
	// zeroCreditPayment envia la condicion credito con un unico pago de 0.00 (el total queda por cobrar)
	zeroCreditPayment bool
}

var (
	ivaRate    = decimal.RequireFromString("0.13")
	ivaDivisor = decimal.RequireFromString("1.13")
	onePercent = decimal.RequireFromString("0.01")
	hundred    = decimal.NewFromInt(100)
	cashCode   = "01" // CAT-017 billetes y monedas
)

func d(v float64) decimal.Decimal       { return decimal.NewFromFloat(v) }
func r8(v decimal.Decimal) float64      { return v.Round(8).InexactFloat64() }
func r2(v decimal.Decimal) float64      { return v.Round(2).InexactFloat64() }
func str(v string) *string              { return &v }
func intPtr(v int) *int                 { return &v }
func code(prefix string, i int) *string { return str(fmt.Sprintf("%s-%03d", prefix, i)) }

// itemSale: (precioUni x cantidad) - montoDescu, con 8 decimales
func itemSale(item itemSpec) decimal.Decimal {
	return d(item.unitPrice).Mul(d(item.quantity)).Sub(d(item.discount)).Round(8)
}

// amountInWords produce un texto alfanumerico simple para totalLetras
func amountInWords(total float64) string {
	value := d(total).Round(2)
	cents := value.Sub(value.Floor()).Mul(hundred).IntPart()
	return fmt.Sprintf("%d DOLARES CON %02d CENTAVOS", value.IntPart(), cents)
}

type saleTotals struct {
	nonSubject, exempt, taxed decimal.Decimal
	nonTaxed                  decimal.Decimal
	itemDiscounts             decimal.Decimal
}

func (s saleTotals) subTotalSales() decimal.Decimal { return s.nonSubject.Add(s.exempt).Add(s.taxed) }

// baseItem arma los campos comunes del item; los cargos van con precio 0, cantidad 1 y unidad 99
func baseItem(it itemSpec, number int, prefix string) structs.ItemRequest {
	unitMeasure := it.unitMeasure
	if unitMeasure == 0 {
		unitMeasure = 59
	}
	req := structs.ItemRequest{
		Number: number, Type: it.itemType, Description: it.description, Quantity: it.quantity,
		UnitMeasure: unitMeasure, UnitPrice: it.unitPrice, Discount: it.discount, Code: code(prefix, number),
	}
	if it.kind == nonTaxedCharge {
		req.Quantity, req.UnitPrice, req.Discount, req.UnitMeasure = 1, 0, 0, 99
	}
	return req
}

// saleAmounts reparte la venta del item en su categoria y acumula los totales
type saleAmounts struct{ nonSubject, exempt, taxed, nonTaxed float64 }

func accumulate(it itemSpec, totals *saleTotals) (saleAmounts, decimal.Decimal) {
	var out saleAmounts
	sale := itemSale(it)
	switch it.kind {
	case taxedSale:
		out.taxed = r8(sale)
		totals.taxed = totals.taxed.Add(sale)
	case exemptSale:
		out.exempt = r8(sale)
		totals.exempt = totals.exempt.Add(sale)
	case nonSubjectSale:
		out.nonSubject = r8(sale)
		totals.nonSubject = totals.nonSubject.Add(sale)
	case nonTaxedCharge:
		out.nonTaxed = it.charge
		totals.nonTaxed = totals.nonTaxed.Add(d(it.charge))
		return out, decimal.Zero
	}
	totals.itemDiscounts = totals.itemDiscounts.Add(d(it.discount))
	return out, sale
}

// discountSummary calcula subTotal, totalDescu y porcentajeDescuento (campos 130-135 y 139)
func discountSummary(totals saleTotals, global globalDiscounts) (subTotal, totalDiscount, percentage decimal.Decimal) {
	globalSum := d(global.nonSubject).Add(d(global.exempt)).Add(d(global.taxed))
	subTotal = totals.subTotalSales().Round(2).Sub(globalSum)
	totalDiscount = totals.itemDiscounts.Add(globalSum)
	if totals.subTotalSales().GreaterThan(decimal.Zero) && globalSum.GreaterThan(decimal.Zero) {
		percentage = globalSum.Div(totals.subTotalSales()).Mul(hundred)
	}
	return subTotal, totalDiscount, percentage
}

// payments reparte el total entre las formas de pago; condicion credito usa tarjeta con plazo
func payments(total float64, opts documentOptions) []structs.PaymentRequest {
	if opts.credit != nil {
		period, term := opts.credit.period, opts.credit.term
		if opts.zeroCreditPayment {
			total = 0
		}
		return []structs.PaymentRequest{{Code: "03", Amount: total, Period: &period, Term: &term}}
	}
	codes := opts.paymentCodes
	if len(codes) == 0 {
		codes = []string{cashCode}
	}
	share := d(total).Div(decimal.NewFromInt(int64(len(codes)))).Round(2)
	result := make([]structs.PaymentRequest, 0, len(codes))
	assigned := decimal.Zero
	for i, c := range codes {
		amount := share
		if i == len(codes)-1 {
			amount = d(total).Sub(assigned) // el ultimo absorbe el redondeo para que la suma sea exacta
		}
		assigned = assigned.Add(amount)
		payment := structs.PaymentRequest{Code: c, Amount: r2(amount)}
		if c != cashCode {
			payment.Reference = str(fmt.Sprintf("REF-%s-%d", c, i+1))
		}
		result = append(result, payment)
	}
	return result
}

func operationCondition(opts documentOptions) int {
	if opts.credit == nil {
		return 1
	}
	return 2
}

func relatedItemDoc(opts documentOptions) *string {
	if len(opts.related) == 0 {
		return nil
	}
	return str(opts.related[0].DocumentNumber)
}

// ---------------- FE (precio con IVA incluido) ----------------

type invoiceSpec struct {
	receiver     *structs.ReceiverRequest // nil: consumidor final bajo el monto legal
	items        []itemSpec
	global       globalDiscounts
	ivaRetention bool // 1% sobre la venta gravada sin IVA (VERIFICAR, guia 3.7)
	extraTax     *extraTax
	opts         documentOptions
}

func buildInvoice(spec invoiceSpec) *structs.CreateInvoiceRequest {
	var totals saleTotals
	var sumIVA decimal.Decimal
	items := make([]structs.InvoiceItemRequest, 0, len(spec.items))

	for i, it := range spec.items {
		amounts, sale := accumulate(it, &totals)
		req := structs.InvoiceItemRequest{
			ItemRequest: baseItem(it, i+1, "FE"),
			TaxedSale:   amounts.taxed, ExemptSale: amounts.exempt, NonSubjectSale: amounts.nonSubject,
			NonTaxed: amounts.nonTaxed,
		}
		req.RelatedDoc = relatedItemDoc(spec.opts)
		if it.kind == taxedSale {
			iva := sale.Div(ivaDivisor).Mul(ivaRate).Round(8) // ivaItem = (ventaGravada / 1.13) x 13%
			req.IVAItem = r8(iva)
			sumIVA = sumIVA.Add(iva)
			if spec.extraTax != nil {
				req.Taxes = []string{spec.extraTax.code}
			}
		}
		items = append(items, req)
	}

	subTotal, totalDiscount, percentage := discountSummary(totals, spec.global)
	// totalIva = suma de ivaItem - IVA del descuento global a ventas gravadas (campo 157)
	totalIVA := sumIVA.Sub(d(spec.global.taxed).Div(ivaDivisor).Mul(ivaRate).Round(8))

	var ivaRetention decimal.Decimal
	if spec.ivaRetention {
		taxedNet := totals.taxed.Sub(d(spec.global.taxed))
		ivaRetention = taxedNet.Div(ivaDivisor).Mul(onePercent).Round(2)
	}

	var taxes []structs.TaxRequest
	totalOperation := subTotal // montoTotalOperacion = subTotal + tributos (campo 150)
	if spec.extraTax != nil && totals.taxed.GreaterThan(decimal.Zero) {
		value := totals.taxed.Round(2).Mul(d(spec.extraTax.rate)).Round(2)
		taxes = []structs.TaxRequest{{Code: spec.extraTax.code, Description: spec.extraTax.description, Value: r2(value)}}
		totalOperation = totalOperation.Add(value)
	}

	// totalPagar = montoTotalOperacion - IVA retenido +/- cargos/abonos (campo 153)
	toPay := r2(totalOperation.Sub(ivaRetention).Add(totals.nonTaxed))

	summary := &structs.InvoiceSummaryRequest{
		SummaryRequest: structs.SummaryRequest{
			TotalNonSubject: r2(totals.nonSubject), TotalExempt: r2(totals.exempt), TotalTaxed: r2(totals.taxed),
			SubTotalSales: r2(totals.subTotalSales()), NonSubjectDiscount: spec.global.nonSubject,
			ExemptDiscount: spec.global.exempt, DiscountPercentage: r2(percentage),
			TotalDiscount: r2(totalDiscount), SubTotal: r2(subTotal), TotalOperation: r2(totalOperation),
			TotalNonTaxed: r2(totals.nonTaxed), TotalToPay: toPay, OperationCondition: operationCondition(spec.opts),
			Taxes: taxes, PaymentTypes: payments(toPay, spec.opts), TotalInWords: str(amountInWords(toPay)),
		},
		TaxedDiscount: spec.global.taxed,
		IVARetention:  r2(ivaRetention),
		TotalIVA:      r2(totalIVA),
	}
	if spec.opts.observation != "" {
		summary.Observations = str(spec.opts.observation)
	}

	return &structs.CreateInvoiceRequest{
		ModelType: 1, Receiver: spec.receiver, Items: items, Summary: summary,
		ThirdPartySale: spec.opts.thirdParty, RelatedDocs: spec.opts.related,
		OtherDocs: spec.opts.otherDocs, Appendixes: spec.opts.appendixes,
	}
}

// ---------------- CCF (precio sin IVA, IVA como tributo 20) ----------------

type ccfSpec struct {
	receiver      *structs.ReceiverRequest
	items         []itemSpec
	global        globalDiscounts
	ivaPerception bool // 1% sobre total gravado (como lo calcula el validador; VERIFICAR guia 3.7)
	ivaRetention  bool // 1% sobre gravado neto del descuento global
	opts          documentOptions
}

func buildCCF(spec ccfSpec) *structs.CreateCreditFiscalRequest {
	var totals saleTotals
	items := make([]structs.CreditItemRequest, 0, len(spec.items))

	for i, it := range spec.items {
		amounts, _ := accumulate(it, &totals)
		req := structs.CreditItemRequest{
			ItemRequest: baseItem(it, i+1, "CCF"),
			TaxedSale:   amounts.taxed, ExemptSale: amounts.exempt, NonSubjectSale: amounts.nonSubject,
			NonTaxed: amounts.nonTaxed,
		}
		req.RelatedDoc = relatedItemDoc(spec.opts)
		if it.kind == taxedSale {
			req.Taxes = []string{"20"}
		}
		items = append(items, req)
	}

	subTotal, totalDiscount, percentage := discountSummary(totals, spec.global)
	taxedNet := totals.taxed.Round(2).Sub(d(spec.global.taxed))

	var taxes []structs.TaxRequest
	var iva decimal.Decimal
	if totals.taxed.GreaterThan(decimal.Zero) {
		iva = taxedNet.Mul(ivaRate).Round(2) // tributo 20 = (totalGravada - descuGravada) x 13%
		taxes = []structs.TaxRequest{{Code: "20", Description: "Impuesto al Valor Agregado 13%", Value: r2(iva)}}
	}

	var perception, retention decimal.Decimal
	if spec.ivaPerception {
		perception = totals.taxed.Round(2).Mul(onePercent).Round(2)
	}
	if spec.ivaRetention {
		retention = taxedNet.Mul(onePercent).Round(2)
	}

	totalOperation := subTotal.Add(iva) // campo 150
	toPay := r2(totalOperation.Add(perception).Sub(retention).Add(totals.nonTaxed))

	summary := &structs.CreditSummaryRequest{
		SummaryRequest: structs.SummaryRequest{
			TotalNonSubject: r2(totals.nonSubject), TotalExempt: r2(totals.exempt), TotalTaxed: r2(totals.taxed),
			SubTotalSales: r2(totals.subTotalSales()), NonSubjectDiscount: spec.global.nonSubject,
			ExemptDiscount: spec.global.exempt, DiscountPercentage: r2(percentage),
			TotalDiscount: r2(totalDiscount), SubTotal: r2(subTotal), TotalOperation: r2(totalOperation),
			TotalNonTaxed: r2(totals.nonTaxed), TotalToPay: toPay, OperationCondition: operationCondition(spec.opts),
			Taxes: taxes, PaymentTypes: payments(toPay, spec.opts), TotalInWords: str(amountInWords(toPay)),
		},
		TaxedDiscount: spec.global.taxed,
		IVAPerception: r2(perception),
		IVARetention:  r2(retention),
	}
	if spec.opts.observation != "" {
		summary.Observations = str(spec.opts.observation)
	}

	return &structs.CreateCreditFiscalRequest{
		ModelType: 1, Receiver: spec.receiver, Items: items, Summary: summary,
		ThirdPartySale: spec.opts.thirdParty, RelatedDocs: spec.opts.related,
		OtherDocs: spec.opts.otherDocs, Appendixes: spec.opts.appendixes,
	}
}

// ---------------- NC / ND (ajustes sobre un CCF o un CR) ----------------

type adjustmentSpec struct {
	receiver      *structs.ReceiverRequest
	related       *emittedDTE
	relatedType   string // "03" CCF (por defecto) o "07" CR
	items         []itemSpec
	ivaPerception bool
	ivaRetention  bool
}

type adjustmentAmounts struct {
	items        []structs.CreditItemRequest
	taxed        float64
	exempt       float64
	nonSubject   float64
	nonTaxed     float64
	subTotal     float64
	taxes        []structs.TaxRequest
	operation    float64
	toPay        float64
	perception   float64
	retention    float64
	itemDiscount float64
}

// adjustmentTotals aplica los campos 150 y 153 de NCE/NDE: montoTotal = subTotalVentas + tributos;
// totalPagar = montoTotal + IVA percibido - IVA retenido +/- cargos
func adjustmentTotals(spec adjustmentSpec, prefix string) adjustmentAmounts {
	var totals saleTotals
	out := adjustmentAmounts{}
	for i, it := range spec.items {
		amounts, _ := accumulate(it, &totals)
		req := structs.CreditItemRequest{
			ItemRequest: baseItem(it, i+1, prefix),
			TaxedSale:   amounts.taxed, ExemptSale: amounts.exempt, NonSubjectSale: amounts.nonSubject,
			NonTaxed: amounts.nonTaxed,
		}
		req.RelatedDoc = str(spec.related.GenerationCode)
		if it.kind == taxedSale {
			req.Taxes = []string{"20"}
		}
		out.items = append(out.items, req)
	}

	subTotalSales := totals.subTotalSales().Round(2)
	var iva decimal.Decimal
	if totals.taxed.GreaterThan(decimal.Zero) {
		iva = totals.taxed.Round(2).Mul(ivaRate).Round(2)
		out.taxes = []structs.TaxRequest{{Code: "20", Description: "Impuesto al Valor Agregado 13%", Value: r2(iva)}}
	}
	var perception, retention decimal.Decimal
	if spec.ivaPerception {
		perception = totals.taxed.Round(2).Mul(onePercent).Round(2)
	}
	if spec.ivaRetention {
		retention = totals.taxed.Round(2).Mul(onePercent).Round(2)
	}
	operation := subTotalSales.Add(iva)

	out.taxed, out.exempt, out.nonSubject = r2(totals.taxed), r2(totals.exempt), r2(totals.nonSubject)
	out.nonTaxed = r2(totals.nonTaxed)
	out.subTotal = r2(subTotalSales)
	out.operation = r2(operation)
	out.toPay = r2(operation.Add(perception).Sub(retention).Add(totals.nonTaxed))
	out.perception, out.retention = r2(perception), r2(retention)
	out.itemDiscount = r2(totals.itemDiscounts)
	return out
}

func relatedDoc(related *emittedDTE, documentType string) []structs.RelatedDocRequest {
	if documentType == "" {
		documentType = "03"
	}
	return []structs.RelatedDocRequest{{
		DocumentType: documentType, GenerationType: 2,
		DocumentNumber: related.GenerationCode, EmissionDate: related.EmissionDate,
	}}
}

func adjustmentSummary(a adjustmentAmounts) structs.SummaryRequest {
	return structs.SummaryRequest{
		TotalNonSubject: a.nonSubject, TotalExempt: a.exempt, TotalTaxed: a.taxed,
		SubTotalSales: a.subTotal, SubTotal: a.subTotal, TotalDiscount: a.itemDiscount,
		TotalNonTaxed: a.nonTaxed, TotalOperation: a.operation, TotalToPay: a.toPay, OperationCondition: 1,
		Taxes: a.taxes, PaymentTypes: payments(a.toPay, documentOptions{}), TotalInWords: str(amountInWords(a.toPay)),
	}
}

func buildCreditNote(spec adjustmentSpec) *structs.CreateCreditNoteRequest {
	a := adjustmentTotals(spec, "NC")
	items := make([]structs.CreditNoteItemRequest, 0, len(a.items))
	for _, it := range a.items {
		items = append(items, structs.CreditNoteItemRequest{
			ItemRequest: it.ItemRequest, NonSubjectSale: it.NonSubjectSale, ExemptSale: it.ExemptSale, TaxedSale: it.TaxedSale,
		})
	}
	return &structs.CreateCreditNoteRequest{
		ModelType: 1, Receiver: spec.receiver, Items: items, RelatedDocs: relatedDoc(spec.related, spec.relatedType),
		Summary: &structs.CreditNoteSummaryRequest{
			SummaryRequest: adjustmentSummary(a), IVAPerception: a.perception, IVARetention: a.retention,
		},
	}
}

func buildDebitNote(spec adjustmentSpec) *structs.CreateDebitNoteRequest {
	a := adjustmentTotals(spec, "ND")
	items := make([]structs.DebitNoteItemRequest, 0, len(a.items))
	for _, it := range a.items {
		items = append(items, structs.DebitNoteItemRequest{
			ItemRequest: it.ItemRequest, NonSubjectSale: it.NonSubjectSale, ExemptSale: it.ExemptSale, TaxedSale: it.TaxedSale,
		})
	}
	return &structs.CreateDebitNoteRequest{
		ModelType: 1, Receiver: spec.receiver, Items: items, RelatedDocs: relatedDoc(spec.related, spec.relatedType),
		Summary: &structs.DebitNoteSummaryRequest{
			SummaryRequest: adjustmentSummary(a), IVAPerception: a.perception, IVARetention: a.retention,
		},
	}
}

// ---------------- FSE ----------------

type fseSpec struct {
	receiver        *structs.FSEReceiverRequest
	items           []itemSpec
	globalDiscount  float64 // descu
	incomeRetention float64 // tasa, por ejemplo 0.10 (VERIFICAR con el contador, guia 3.11)
	paymentCodes    []string
}

func buildFSE(spec fseSpec) *structs.CreateFSERequest {
	var totalPurchase, itemDiscounts decimal.Decimal
	items := make([]structs.FSEItemRequest, 0, len(spec.items))
	for i, it := range spec.items {
		purchase := itemSale(it)
		items = append(items, structs.FSEItemRequest{ItemRequest: baseItem(it, i+1, "FSE"), Purchase: r8(purchase)})
		totalPurchase = totalPurchase.Add(purchase)
		itemDiscounts = itemDiscounts.Add(d(it.discount))
	}

	subTotal := totalPurchase.Round(2).Sub(d(spec.globalDiscount)) // subTotal = totalCompra - descu
	retention := subTotal.Mul(d(spec.incomeRetention)).Round(2)
	toPay := r2(subTotal.Sub(retention)) // totalPagar = subTotal - reteRenta

	return &structs.CreateFSERequest{
		Receiver: spec.receiver, Items: items,
		Summary: &structs.FSESummaryRequest{
			SummaryRequest: structs.SummaryRequest{
				NonSubjectDiscount: spec.globalDiscount, // el mapper de FSE usa este campo como "descu"
				TotalDiscount:      r2(itemDiscounts.Add(d(spec.globalDiscount))),
				SubTotal:           r2(subTotal), TotalToPay: toPay, OperationCondition: 1,
				PaymentTypes: payments(toPay, documentOptions{paymentCodes: spec.paymentCodes}),
				TotalInWords: str(amountInWords(toPay)),
			},
			TotalPurchase:   r2(totalPurchase),
			IncomeRetention: r2(retention),
		},
	}
}

// ---------------- NR ----------------

func buildRemissionNote(receiver *structs.RemissionNoteReceiverRequest, specs []itemSpec) *structs.CreateRemissionNoteRequest {
	var taxed, itemDiscounts decimal.Decimal
	items := make([]*structs.RemissionNoteItemRequest, 0, len(specs))
	zero := 0.0
	for i, it := range specs {
		sale := r8(itemSale(it))
		number, itemType := i+1, it.itemType
		quantity, price, discount := it.quantity, it.unitPrice, it.discount
		items = append(items, &structs.RemissionNoteItemRequest{
			ItemNumber: &number, ItemType: &itemType, Code: code("NR", i+1), Description: str(it.description),
			Quantity: &quantity, UnitMeasure: intPtr(59), UnitPrice: &price, DiscountAmount: &discount,
			NonSubjectSale: &zero, ExemptSale: &zero, TaxedSale: &sale, Tributes: []string{"20"},
		})
		taxed = taxed.Add(d(sale))
		itemDiscounts = itemDiscounts.Add(d(it.discount))
	}

	taxedTotal := r2(taxed)
	iva := r2(d(taxedTotal).Mul(ivaRate))
	total := r2(d(taxedTotal).Add(d(iva))) // montoTotalOperacion = subTotal + tributos
	totalDiscount := r2(itemDiscounts)
	words := amountInWords(total)

	return &structs.CreateRemissionNoteRequest{
		Receiver: receiver, Items: items,
		Summary: &structs.RemissionNoteSummaryRequest{
			NonSubjectTotal: &zero, ExemptTotal: &zero, TaxedTotal: &taxedTotal, SubtotalSales: &taxedTotal,
			NonSubjectDiscount: &zero, ExemptDiscount: &zero, TaxedDiscount: &zero, TotalDiscount: &totalDiscount,
			Tributes: []*structs.TaxRequest{{Code: "20", Description: "Impuesto al Valor Agregado 13%", Value: iva}},
			Subtotal: &taxedTotal, TotalAmount: &total, AmountInWords: &words,
		},
	}
}

// ---------------- CR (Comprobante de Retencion con documentos fisicos) ----------------

type retentionLine struct {
	documentNumber string  // numero del documento fisico (FE o CCF del proveedor)
	documentType   string  // "01" FE o "03" CCF
	taxedAmount    float64 // monto sujeto a retencion (sin IVA)
	code           string  // CAT-006: 22 = 1%, C4 = 13%
	description    string
}

var retentionRates = map[string]decimal.Decimal{"22": onePercent, "C4": ivaRate}

// buildRetention: IVA retenido por item = monto sujeto x tasa del codigo; totales = suma de items
func buildRetention(receiver *structs.ReceiverRequest, lines []retentionLine, emissionDate string) *structs.CreateRetentionRequest {
	var totalSubject, totalRetained decimal.Decimal
	items := make([]structs.RetentionItem, 0, len(lines))
	for _, line := range lines {
		subject := d(line.taxedAmount).Round(2)
		retained := subject.Mul(retentionRates[line.code]).Round(2)
		subjectValue, retainedValue := r2(subject), r2(retained)
		items = append(items, structs.RetentionItem{
			DocumentType: 1, DocumentNumber: line.documentNumber, Description: line.description,
			RetentionCode: line.code, TaxedAmount: &subjectValue, IvaAmount: &retainedValue,
			EmissionDate: str(emissionDate), DTEType: str(line.documentType),
		})
		totalSubject = totalSubject.Add(subject)
		totalRetained = totalRetained.Add(retained)
	}
	return &structs.CreateRetentionRequest{
		Receiver: receiver, Items: items,
		Summary: &structs.RetentionSummary{
			TotalRetentionAmount: r2(totalSubject),
			TotalRetentionIVA:    r2(totalRetained),
		},
	}
}
