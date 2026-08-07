package test_endpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	authModels "github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/test_endpoint"
	"gorm.io/gorm"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	commonModels "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	identificationVO "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/location"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/test_endpoint/models"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type testService struct {
	db         *gorm.DB
	authRepo   auth.AuthRepositoryPort
	httpClient *http.Client
}

func NewTestService(db *gorm.DB, authRepo auth.AuthRepositoryPort) test_endpoint.TestManager {
	return &testService{
		db:       db,
		authRepo: authRepo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (s *testService) RunSystemTest(ctx context.Context) (*models.TestResult, error) {
	startTime := utils.TimeNow()
	tests := make([]models.ComponentTest, 0)

	claims := ctx.Value("claims").(*authModels.AuthClaims)

	dbTest := s.testDatabase()
	tests = append(tests, dbTest)
	if !dbTest.Success {
		return s.buildResult(tests, startTime), nil
	}

	mappingTest := s.testDTEMapping()
	tests = append(tests, mappingTest)
	if !mappingTest.Success {
		return s.buildResult(tests, startTime), nil
	}

	seqTest := s.testSequenceGeneration()
	tests = append(tests, seqTest)

	signerTest := s.testSignerService(ctx, claims.NIT)
	tests = append(tests, signerTest)

	testDTE := getTestDTE()
	haciendaTest := s.testHaciendaTransmission(testDTE)
	tests = append(tests, haciendaTest)

	return s.buildResult(tests, startTime), nil
}

func (s *testService) testDatabase() models.ComponentTest {
	start := utils.TimeNow()
	test := models.ComponentTest{
		Name: "database_connection",
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		logs.Error("Database connection test failed", map[string]interface{}{
			"error": err.Error(),
		})
		test.Success = false
	}

	if err := sqlDB.Ping(); err != nil {
		logs.Error("Database ping test failed", map[string]interface{}{
			"error": err.Error(),
		})
		test.Success = false
	} else {
		test.Success = true
	}

	test.Duration = time.Since(start).Milliseconds()
	return test
}

func (s *testService) testDTEMapping() models.ComponentTest {
	start := utils.TimeNow()
	test := models.ComponentTest{
		Name: "dte_mapping",
	}

	testDTE := getTestDTE()

	mh := response_mapper.ToMHInvoice(testDTE)
	if mh == nil {
		logs.Error("DTE mapping test failed")
		test.Success = false
	} else {
		test.Success = true
	}

	test.Duration = time.Since(start).Milliseconds()
	return test
}

func (s *testService) testSequenceGeneration() models.ComponentTest {
	start := utils.TimeNow()
	test := models.ComponentTest{
		Name: "sequence_generation",
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&db_models.ControlNumberSequence{}).
			Where("branch_id = ? AND dte_type = ?", 0, "01").
			UpdateColumn("last_number", gorm.Expr("last_number + ?", 1)).
			Error
	})

	if err != nil {
		logs.Error("Sequence generation test failed", map[string]interface{}{
			"error": err.Error(),
		})
		test.Success = false
	} else {
		test.Success = true
	}

	test.Duration = time.Since(start).Milliseconds()
	return test
}

func (s *testService) buildResult(tests []models.ComponentTest, startTime time.Time) *models.TestResult {
	success := true
	for _, test := range tests {
		if !test.Success {
			success = false
			break
		}
	}

	return &models.TestResult{
		Success:  success,
		Tests:    tests,
		Duration: time.Since(startTime).Milliseconds(),
	}
}

func (s *testService) testHaciendaTransmission(testDTE *invoice_models.ElectronicInvoice) models.ComponentTest {
	start := utils.TimeNow()
	test := models.ComponentTest{
		Name: "hacienda_transmission",
	}

	mhDTE := response_mapper.ToMHInvoice(testDTE)
	if mhDTE == nil {
		logs.Error("Failed to map test DTE")
		test.Success = false
		test.Duration = time.Since(start).Milliseconds()
		return test
	}

	jsonData, err := json.Marshal(mhDTE)
	if err != nil {
		logs.Error("Failed to marshal test DTE", map[string]interface{}{
			"error": err.Error(),
		})
		test.Success = false
		test.Duration = time.Since(start).Milliseconds()
		return test
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx,
		"POST",
		config.MHPaths.ReceptionURL,
		bytes.NewBuffer(jsonData))
	if err != nil {
		logs.Error("Failed to create request", map[string]interface{}{
			"error": err.Error(),
		})
		test.Success = false
		test.Duration = time.Since(start).Milliseconds()
		return test
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logs.Error("Failed to transmit to Hacienda", map[string]interface{}{
			"error": err.Error(),
		})
		test.Success = false
		test.Duration = time.Since(start).Milliseconds()
		return test
	}
	defer resp.Body.Close()

	test.Success = resp.StatusCode >= 400 && resp.StatusCode < 500

	logs.Info("Hacienda transmission test completed", map[string]interface{}{
		"statusCode": resp.StatusCode,
		"success":    test.Success,
	})

	test.Duration = time.Since(start).Milliseconds()
	return test
}

func (s *testService) testSignerService(ctx context.Context, nit string) models.ComponentTest {
	start := utils.TimeNow()
	test := models.ComponentTest{
		Name: "signer_service",
	}

	client, err := s.authRepo.GetByNIT(ctx, nit)
	if err != nil {
		logs.Error("Failed to get client by NIT for signer test", map[string]interface{}{
			"error": err.Error(),
			"nit":   nit,
		})
		test.Success = false
		test.Duration = time.Since(start).Milliseconds()
		return test
	}

	testDTE := getTestDTE()
	mhDTE := response_mapper.ToMHInvoice(testDTE)
	if mhDTE == nil {
		logs.Error("Failed to map test DTE for signer test")
		test.Success = false
		test.Duration = time.Since(start).Milliseconds()
		return test
	}

	jsonData, err := json.Marshal(mhDTE)
	if err != nil {
		logs.Error("Failed to marshal test DTE for signer", map[string]interface{}{
			"error": err.Error(),
		})
		test.Success = false
		test.Duration = time.Since(start).Milliseconds()
		return test
	}

	signRequest := map[string]interface{}{
		"nit":         nit,
		"activo":      true,
		"passwordPri": client.PasswordPri,
		"dteJson":     json.RawMessage(jsonData),
	}

	reqBody, err := json.Marshal(signRequest)
	if err != nil {
		logs.Error("Failed to marshal sign request", map[string]interface{}{
			"error": err.Error(),
		})
		test.Success = false
		test.Duration = time.Since(start).Milliseconds()
		return test
	}

	httpReq, err := http.NewRequestWithContext(ctx,
		"POST",
		config.Signer.Path,
		bytes.NewBuffer(reqBody))
	if err != nil {
		logs.Error("Failed to create signer request", map[string]interface{}{
			"error": err.Error(),
		})
		test.Success = false
		test.Duration = time.Since(start).Milliseconds()
		return test
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		logs.Error("Failed to call signer service", map[string]interface{}{
			"error": err.Error(),
		})
		test.Success = false
		test.Duration = time.Since(start).Milliseconds()
		return test
	}
	defer resp.Body.Close()

	test.Success = resp.StatusCode == http.StatusOK

	logs.Info("Signer service test completed", map[string]interface{}{
		"statusCode": resp.StatusCode,
		"success":    test.Success,
	})

	test.Duration = time.Since(start).Milliseconds()
	return test
}

// getTestDTE generates a DTE with the minimum required fields for testing
func getTestDTE() *invoice_models.ElectronicInvoice {
	identification, err := common.MapCommonRequestIdentification(1, 1, constants.FacturaElectronica)
	if err != nil {
		logs.Error("Failed to map identification", map[string]interface{}{
			"error": err.Error(),
		})
		return nil
	}

	issuer := &commonModels.Issuer{
		NIT:                 *identificationVO.NewValidatedNIT("TEST PARA PRUEBAS"),
		NRC:                 *identificationVO.NewValidatedNRC("TEST PARA PRUEBAS"),
		Name:                "EMPRESA DE PRUEBA",
		ActivityCode:        *identificationVO.NewValidatedActivityCode("01234"),
		ActivityDescription: "ACTIVIDAD DE PRUEBA",
		EstablishmentType:   *document.NewValidatedEstablishmentType("01"),
		Address: &commonModels.Address{
			Department:   *location.NewValidatedDepartment("01"),
			Municipality: *location.NewValidatedMunicipality("01", "01"),
			Complement:   *location.NewValidatedAddress("DIRECCION DE PRUEBA"),
		},
		Phone: *base.NewValidatedPhone("22222222"),
		Email: *base.NewValidatedEmail("test@test.com"),
	}

	name := "CLIENTE DE PRUEBA"
	receiver := &commonModels.Receiver{
		DocumentType:   document.NewValidatedDTEType("13"),
		DocumentNumber: identificationVO.NewValidatedDocumentNumber("00000000-0"),
		Name:           &name,
		Address: &commonModels.Address{
			Department:   *location.NewValidatedDepartment("09"),
			Municipality: *location.NewValidatedMunicipality("04", "09"),
			Complement:   *location.NewValidatedAddress("SIMON"),
		},
		Email: base.NewValidatedEmail("example@gmail.com"),
	}

	items := []invoice_models.InvoiceItem{
		{
			Item: &commonModels.Item{
				Number:      *item.NewValidatedItemNumber(1),
				Type:        *item.NewValidatedItemType(1),
				Description: "PRODUCTO DE PRUEBA",
				Quantity:    *item.NewValidatedQuantity(1),
				UnitMeasure: *item.NewValidatedUnitMeasure(59),
				UnitPrice:   *financial.NewValidatedAmount(7.50),
				Discount:    *financial.NewValidatedDiscount(0.525),
				Code:        item.NewValidatedItemCode("6609"),
				Taxes:       []string{constants.TaxIVA},
			},
			NonSubjectSale: *financial.NewValidatedAmount(0),
			ExemptSale:     *financial.NewValidatedAmount(0),
			TaxedSale:      *financial.NewValidatedAmount(6.98),
			SuggestedPrice: *financial.NewValidatedAmount(0),
			NonTaxed:       *financial.NewValidatedAmount(0),
			IVAItem:        *financial.NewValidatedAmount(0.80),
		},
	}

	summary := invoice_models.InvoiceSummary{
		Summary: &commonModels.Summary{
			TotalNonSubject:    *financial.NewValidatedAmount(0),
			TotalExempt:        *financial.NewValidatedAmount(0),
			TotalTaxed:         *financial.NewValidatedAmount(6.98),
			SubTotal:           *financial.NewValidatedAmount(6.98),
			SubTotalSales:      *financial.NewValidatedAmount(6.98),
			NonSubjectDiscount: *financial.NewValidatedAmount(0),
			ExemptDiscount:     *financial.NewValidatedAmount(0),
			DiscountPercentage: *financial.NewValidatedDiscount(0),
			TotalDiscount:      *financial.NewValidatedAmount(0.53),
			TotalOperation:     *financial.NewValidatedAmount(6.98),
			TotalNonTaxed:      *financial.NewValidatedAmount(0),
			OperationCondition: *financial.NewValidatedPaymentCondition(1),
			TotalToPay:         *financial.NewValidatedAmount(6.98),
			TotalTaxes: []interfaces.Tax{
				&commonModels.Tax{
					Code:        *financial.NewValidatedTaxType(constants.TaxIVA),
					Description: "IVA 13%",
					Value:       &commonModels.TaxAmount{TotalAmount: *financial.NewValidatedAmount(0.80)},
				},
			},
			PaymentTypes: []interfaces.PaymentType{
				&commonModels.PaymentType{
					Code:      *financial.NewValidatedPaymentType("01"),
					Amount:    *financial.NewValidatedAmount(6.98),
					Reference: "",
				},
			},
		},
		TotalIva: *financial.NewValidatedAmount(0.80),
	}

	var itemsInterface = make([]interfaces.Item, len(items))
	for i, item := range items {
		itemsInterface[i] = &item
	}

	return &invoice_models.ElectronicInvoice{
		DTEDocument: &commonModels.DTEDocument{
			Identification: identification,
			Issuer:         issuer,
			Receiver:       receiver,
			Items:          itemsInterface,
			Summary:        &summary,
		},
		InvoiceItems:   items,
		InvoiceSummary: summary,
	}
}
