package utils_test

import (
	"encoding/json"
	"testing"

	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/location"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	responseCommon "github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDistrictValueObject validates District creation and format constraints
func TestDistrictValueObject(t *testing.T) {
	test.TestMain(t)
	validCodes := []string{"01", "02", "10", "15", "33", "99"}
	for _, code := range validCodes {
		dist, err := location.NewDistrict(code)
		require.NoError(t, err)
		assert.Equal(t, code, dist.GetValue())
		assert.Equal(t, code, dist.ToString())
		assert.True(t, dist.IsValid())

		other := location.NewValidatedDistrict(code)
		assert.True(t, dist.Equals(other))
	}

	invalidCodes := []string{"00", "0", "1", "123", "AA", "", " 1"}
	for _, code := range invalidCodes {
		_, err := location.NewDistrict(code)
		assert.Error(t, err)
	}
}

// TestVersionValueObject validates document Version constraints up to version 4
func TestVersionValueObject(t *testing.T) {
	validVersions := []int{1, 2, 3, 4}
	for _, v := range validVersions {
		ver, err := document.NewVersion(v)
		require.NoError(t, err)
		assert.Equal(t, v, ver.GetValue())
		assert.True(t, ver.IsValid())
	}

	invalidVersions := []int{0, -1, 5, 10}
	for _, v := range invalidVersions {
		_, err := document.NewVersion(v)
		assert.Error(t, err)
	}
}

// TestAddressModelWithDistrict validates Address model getter and setter with District
func TestAddressModelWithDistrict(t *testing.T) {
	addr := &models.Address{}
	require.NoError(t, addr.SetDepartment("06"))
	require.NoError(t, addr.SetMunicipality("20"))
	require.NoError(t, addr.SetDistrict("01"))
	require.NoError(t, addr.SetComplement("Colonia Escalon"))

	assert.Equal(t, "06", addr.GetDepartment())
	assert.Equal(t, "20", addr.GetMunicipality())
	assert.Equal(t, "01", addr.GetDistrict())
	assert.Equal(t, "Colonia Escalon", addr.GetComplement())

	assert.Error(t, addr.SetDistrict("00"))
	assert.Error(t, addr.SetDistrict("XYZ"))
}

// TestUserAddressValidationWithDistrict validates user.Address validation logic
func TestUserAddressValidationWithDistrict(t *testing.T) {
	validUserAddr := &user.Address{
		Department:   "06",
		Municipality: "20",
		District:     "01",
		Complement:   "Avenida Masferrer Norte",
	}
	assert.NoError(t, validUserAddr.Validate())

	missingDistrict := &user.Address{
		Department:   "06",
		Municipality: "20",
		District:     "",
		Complement:   "Avenida Masferrer Norte",
	}
	assert.Error(t, missingDistrict.Validate())
}

// TestAddressMappersIntegration validates request and response address mapping with district
func TestAddressMappersIntegration(t *testing.T) {
	reqAddr := structs.AddressRequest{
		Department:   "06",
		Municipality: "20",
		District:     "03",
		Complement:   "Calle Los Almendros",
	}

	domainAddr, err := common.MapCommonRequestAddress(reqAddr)
	require.NoError(t, err)
	assert.Equal(t, "03", domainAddr.GetDistrict())

	respAddr := responseCommon.MapCommonResponseAddress(domainAddr)
	assert.Equal(t, "06", respAddr.Departamento)
	assert.Equal(t, "20", respAddr.Municipio)
	assert.Equal(t, "03", respAddr.Distrito)
	assert.Equal(t, "Calle Los Almendros", respAddr.Complemento)

	jsonBytes, err := json.Marshal(respAddr)
	require.NoError(t, err)

	var jsonMap map[string]interface{}
	require.NoError(t, json.Unmarshal(jsonBytes, &jsonMap))
	assert.Equal(t, "03", jsonMap["distrito"])
}

// TestDBAddressModelFields validates GORM Address model column definition
func TestDBAddressModelFields(t *testing.T) {
	dbAddr := db_models.Address{
		BranchID:     1,
		Department:   "06",
		Municipality: "20",
		District:     "01",
		Complement:   "Barrio El Centro",
	}
	assert.Equal(t, "01", dbAddr.District)
	assert.Equal(t, "addresses", dbAddr.TableName())
}
