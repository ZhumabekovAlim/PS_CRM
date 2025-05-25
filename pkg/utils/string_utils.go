package utils

// NewNullString is a helper for string pointers, returning nil if string is empty.
// Useful for fields that are optional and should be NULL in DB if not provided.
func NewNullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
