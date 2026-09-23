package mappers

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestCommonMappers(t *testing.T) {
	test.TestMain(t)

	t.Run("TestMapCommonRequestAddress", func(t *testing.T) {
		address := fixtures.CreateDefaultAddress()
		result, err := common.MapCommonRequestAddress(*address)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, address.Department, result.Department.GetValue())
		assert.Equal(t, address.Municipality, result.Municipality.GetValue())
		assert.Equal(t, address.District, result.District.GetValue())
		assert.Equal(t, address.Complement, result.Complement.GetValue())

		addressInvalid := fixtures.CreateAddressWithEmptyFields()
		result, err = common.MapCommonRequestAddress(*addressInvalid)
		assert.Error(t, err)
		assert.Nil(t, result)

		addressInvalidMunicipality := fixtures.CreateAddressWithInvalidMunicipality()
		result, err = common.MapCommonRequestAddress(*addressInvalidMunicipality)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("TestMapCommonRequestReceiver", func(t *testing.T) {
		receiver := fixtures.CreateDefaultReceiver()
		result, err := common.MapCommonRequestReceiver(receiver)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, *receiver.DocumentType, result.DocumentType.GetValue())
		assert.Equal(t, *receiver.Name, *result.Name)

		receiverInvalidEmail := fixtures.CreateReceiverWithInvalidEmail()
		result, err = common.MapCommonRequestReceiver(receiverInvalidEmail)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("TestMapCommonRequestExtension", func(t *testing.T) {
		extension := fixtures.CreateDefaultExtension()
		result, err := common.MapCommonRequestExtension(extension)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, extension.DeliveryName, result.DeliveryName.GetValue())
		assert.Equal(t, extension.ReceiverName, result.ReceiverName.GetValue())

		extensionInvalid := fixtures.CreateExtensionWithMissingFields()
		result, err = common.MapCommonRequestExtension(extensionInvalid)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("TestMapCommonRequestAppendix", func(t *testing.T) {
		appendixes := []structs.AppendixRequest{fixtures.CreateDefaultAppendix()}
		result, err := common.MapCommonRequestAppendix(appendixes)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, appendixes[0].Field, result[0].Field.GetValue())

		appendixesInvalid := []structs.AppendixRequest{fixtures.CreateAppendixWithInvalidField()}
		result, err = common.MapCommonRequestAppendix(appendixesInvalid)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("TestMapCommonRequestPaymentsType", func(t *testing.T) {
		payments := []structs.PaymentRequest{fixtures.CreateDefaultPayment()}
		result, err := common.MapCommonRequestPaymentsType(payments)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, payments[0].Code, result[0].GetCode())

		paymentsInvalid := []structs.PaymentRequest{fixtures.CreatePaymentWithInvalidCode()}
		result, err = common.MapCommonRequestPaymentsType(paymentsInvalid)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("TestMapCommonRequestThirdPartySale", func(t *testing.T) {
		thirdPartySale := fixtures.CreateDefaultThirdPartySale()
		result, err := common.MapCommonRequestThirdPartySale(thirdPartySale)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, thirdPartySale.Name, result.Name)

		thirdPartySaleInvalid := fixtures.CreateThirdPartySaleWithEmptyNIT()
		result, err = common.MapCommonRequestThirdPartySale(thirdPartySaleInvalid)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
