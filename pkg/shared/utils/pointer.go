package utils

// ToStringPointer converts a string to a string pointer
func ToStringPointer(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ToIntPointer converts an int to an int pointer
func ToIntPointer(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// ToFloat64Pointer converts a float64 to a float64 pointer
func ToFloat64Pointer(f float64) *float64 {
	return &f
}

// PointerToString converts a string pointer to a string
func PointerToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
