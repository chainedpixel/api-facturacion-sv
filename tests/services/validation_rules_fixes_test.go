package services

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/validator/strategy"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	fseStrategy "github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/validator/strategy"
	test "github.com/chainedpixel/ordo-factus/tests"
)

func flattenCodes(err *dte_errors.DTEError) []string {
	if err == nil {
		return nil
	}
	if len(err.BusinessErrors) == 0 {
		return []string{err.GetCode()}
	}
	var codes []string
	for _, nested := range err.BusinessErrors {
		codes = append(codes, flattenCodes(nested)...)
	}
	return codes
}

func validationErrorType(t *testing.T, err error) string {
	t.Helper()
	var validationErr *dte_errors.ValidationError
	require.True(t, errors.As(err, &validationErr))
	return validationErr.GetType()
}

func TestItemDiscountIsMonetaryAmount(t *testing.T) {
	test.TestMain(t)

	t.Run("discount above 100 is accepted", func(t *testing.T) {
		var baseItem models.Item
		require.NoError(t, baseItem.SetDiscount(150.00))
		assert.Equal(t, 150.00, baseItem.GetDiscount())
	})

	t.Run("negative discount is rejected", func(t *testing.T) {
		var baseItem models.Item
		err := baseItem.SetDiscount(-1)
		require.Error(t, err)
		assert.Equal(t, "InvalidAmount", validationErrorType(t, err))
	})

	t.Run("discount up to the line amount keeps sale amount consistent", func(t *testing.T) {
		assert.Nil(t, strategy.ValidateItemSaleAmount(1, 200, 1, 150, 50))
		assert.NotNil(t, strategy.ValidateItemSaleAmount(1, 100, 1, 150, 0))
	})

	t.Run("summary percentage above 100 is still rejected", func(t *testing.T) {
		_, err := financial.NewDiscount(101)
		require.Error(t, err)
		assert.Equal(t, "InvalidDiscount", validationErrorType(t, err))
	})
}

func newFSEDocument(itemType int) *fse_models.FSEModel {
	baseItem := &models.Item{
		Type:        *item.NewValidatedItemType(itemType),
		Description: "Item",
		Quantity:    *item.NewValidatedQuantity(1),
		UnitPrice:   *financial.NewValidatedAmount(10),
	}
	return &fse_models.FSEModel{
		FSEItems: []fse_models.FSEItem{{Item: baseItem, Purchase: *financial.NewValidatedAmount(10)}},
	}
}

func TestFSEItemStrategyRejectsItemTypeFour(t *testing.T) {
	test.TestMain(t)

	err := fseStrategy.NewFSEItemStrategy(newFSEDocument(constants.Impuesto)).Validate()
	require.NotNil(t, err)
	assert.Equal(t, []string{"INVALIDFSEITEMTYPE"}, flattenCodes(err))

	for _, allowed := range []int{constants.Producto, constants.Servicio, constants.ProductoYServicio} {
		assert.Nil(t, fseStrategy.NewFSEItemStrategy(newFSEDocument(allowed)).Validate())
	}
}

func newDocumentWithOtherDoc(dteType string, code int) *models.DTEDocument {
	return &models.DTEDocument{
		Identification: &models.Identification{DTEType: *document.NewValidatedDTEType(dteType)},
		OtherDocuments: []interfaces.OtherDocuments{&models.OtherDocument{
			AssociatedCode: *document.NewValidatedAssociatedDocumentCode(code),
			Description:    strPtr("Description"),
			Detail:         strPtr("Detail"),
			Doctor:         &models.DoctorInfo{},
		}},
	}
}

func strPtr(value string) *string { return &value }

func TestOtherDocumentsCodeFourOnlyForExport(t *testing.T) {
	test.TestMain(t)

	err := (&strategy.OtherDocumentsStrategy{Document: newDocumentWithOtherDoc(constants.FacturaElectronica, 4)}).Validate()
	require.NotNil(t, err)
	assert.Equal(t, "INVALIDASSOCIATEDDOCUMENTCODEFORTYPE", err.GetCode())

	err = (&strategy.OtherDocumentsStrategy{Document: newDocumentWithOtherDoc(constants.FacturaExportacionElectronica, 4)}).Validate()
	assert.Nil(t, err)

	err = (&strategy.OtherDocumentsStrategy{Document: newDocumentWithOtherDoc(constants.FacturaElectronica, 1)}).Validate()
	assert.Nil(t, err)
}

func TestUnsupportedFuelTaxCodes(t *testing.T) {
	test.TestMain(t)

	for _, code := range []string{constants.TaxFOVIAL, constants.TaxCOTRANS} {
		_, err := financial.NewTaxType(code)
		require.Error(t, err)
		assert.Equal(t, "UnsupportedTaxCode", validationErrorType(t, err))
	}

	_, err := financial.NewTaxType(constants.TaxIVA)
	assert.NoError(t, err)
}

type paymentCase struct {
	name      string
	condition int
	amounts   []float64
	wantCode  string
}

func newPaymentDocument(condition int, total float64, amounts []float64) *models.DTEDocument {
	payments := make([]interfaces.PaymentType, len(amounts))
	for index, amount := range amounts {
		payment := &models.PaymentType{
			Code:   *financial.NewValidatedPaymentType(constants.TarjetaCredito),
			Amount: *financial.NewValidatedAmount(amount),
		}
		if condition == constants.Credit {
			period := 30
			payment.Period = &period
			payment.Term = financial.NewValidatedPaymentTerm("01")
		}
		payments[index] = payment
	}
	return &models.DTEDocument{
		Identification: &models.Identification{DTEType: *document.NewValidatedDTEType(constants.FacturaElectronica)},
		Summary: &models.Summary{
			OperationCondition: *financial.NewValidatedPaymentCondition(condition),
			TotalToPay:         *financial.NewValidatedAmount(total),
			PaymentTypes:       payments,
		},
	}
}

func TestPaymentTotalCreditAcceptsZeroAmount(t *testing.T) {
	test.TestMain(t)

	cases := []paymentCase{
		{"credit with zero amount", constants.Credit, []float64{0}, ""},
		{"credit with total amount", constants.Credit, []float64{100}, ""},
		{"credit with partial amount", constants.Credit, []float64{50}, "INVALIDPAYMENTTOTAL"},
		{"credit with empty payments", constants.Credit, nil, "INVALIDPAYMENTTOTAL"},
		{"cash with zero amount", constants.Cash, []float64{0}, "INVALIDPAYMENTTOTAL"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := newPaymentDocument(tc.condition, 100, tc.amounts)
			err := (&strategy.PaymentTotalStrategy{Document: doc}).Validate()
			if tc.wantCode == "" {
				assert.Nil(t, err)
				return
			}
			require.NotNil(t, err)
			assert.Equal(t, tc.wantCode, err.GetCode())
		})
	}
}
