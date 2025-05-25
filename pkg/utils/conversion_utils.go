package utils

import "strconv"

// Int64ToStr converts an int64 to its string representation.
func Int64ToStr(num int64) string {
	return strconv.FormatInt(num, 10)
}

// StrToInt64 converts a string to an int64.
// Returns 0 and an error if the conversion fails.
// Consider adding more robust error handling or specific error return if needed by callers.
func StrToInt64(s string) (int64, error) {
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		// Example of wrapping error for more context:
		// return 0, fmt.Errorf("failed to parse '%s' as int64: %w", s, err)
		return 0, err 
	}
	return num, nil
}
