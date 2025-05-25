package utils

import "os"

// Getenv retrieves the value of the environment variable named by the key.
// If the variable is not present or its value is empty, Getenv returns the fallback string.
func Getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}
