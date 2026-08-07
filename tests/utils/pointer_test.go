package utils

import (
	"testing"

	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToStringPointer verifies that a non-empty string is correctly converted to *string.
func TestToStringPointer(t *testing.T) {
	s := "hello"
	ptr := utils.ToStringPointer(s)

	require.NotNil(t, ptr)
	assert.Equal(t, s, *ptr)
}

// TestToStringPointer_EmptyString verifies empty string returns nil (by design).
func TestToStringPointer_EmptyString(t *testing.T) {
	ptr := utils.ToStringPointer("")
	assert.Nil(t, ptr)
}

// TestToIntPointer verifies non-zero ints are correctly converted to *int.
func TestToIntPointer(t *testing.T) {
	cases := []int{1, -1, 42, 99999}
	for _, v := range cases {
		ptr := utils.ToIntPointer(v)
		require.NotNil(t, ptr)
		assert.Equal(t, v, *ptr)
	}
}

// TestToIntPointer_Zero verifies that 0 returns nil (by design).
func TestToIntPointer_Zero(t *testing.T) {
	ptr := utils.ToIntPointer(0)
	assert.Nil(t, ptr)
}

// TestToFloat64Pointer verifies that a float64 value is correctly converted to *float64.
func TestToFloat64Pointer(t *testing.T) {
	cases := []float64{0.0, 1.5, -3.14, 100.0, 0.001}
	for _, v := range cases {
		ptr := utils.ToFloat64Pointer(v)
		require.NotNil(t, ptr)
		assert.InDelta(t, v, *ptr, 1e-9)
	}
}

// TestPointerToString_NonNil verifies dereferencing a valid *string.
func TestPointerToString_NonNil(t *testing.T) {
	s := "world"
	result := utils.PointerToString(&s)
	assert.Equal(t, "world", result)
}

// TestPointerToString_Nil verifies that a nil *string returns an empty string.
func TestPointerToString_Nil(t *testing.T) {
	result := utils.PointerToString(nil)
	assert.Equal(t, "", result)
}
