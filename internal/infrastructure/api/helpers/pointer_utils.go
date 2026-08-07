package helpers

// ToFloat64Pointer converts a float64 to a pointer to float64
func ToFloat64Pointer(value float64) *float64 {
	return &value
}

// ToIntPointer converts an int to a pointer to int
func ToIntPointer(value int) *int {
	return &value
}

// ToStringPointer converts a string to a pointer to string
func ToStringPointer(value string) *string {
	return &value
}
