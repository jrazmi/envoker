package cryptids

import (
	"crypto/rand"
	"fmt"
)

var (
	IDAlphabet = "bcdfghjklmnpqrstvwxyZBCDFGHJKLMNPQRSTVWXYZ0123456789"
	IDLength   = 18
)

// GenerateID creates a random string from defaults
func GenerateID() (string, error) {
	return generateID(IDAlphabet, IDLength)
}

// GenerateID creates a random string from defaults
func GenerateCustomID(alphabet string, size int) (string, error) {
	return generateID(alphabet, size)
}

// GenerateNanoID creates a random string ID with the given alphabet and length
func generateID(alphabet string, size int) (string, error) {
	// Basic validation
	if len(alphabet) < 2 {
		return "", fmt.Errorf("alphabet must contain at least 2 characters")
	}
	if size < 1 {
		return "", fmt.Errorf("size must be at least 1")
	}

	// Calculate the mask based on the closest power of 2 that's less than the alphabet length
	mask := 1
	for mask < len(alphabet) {
		mask = (mask << 1) | 1
	}
	mask = mask >> 0 // This is the max value we should allow

	// Create a buffer with the necessary size
	step := int(float64(size) * 1.6) // Using a larger buffer to avoid multiple RNG calls
	if step < size {
		step = size
	}

	id := make([]byte, size)
	bytes := make([]byte, step)

	idIndex := 0
	for idIndex < size {
		// Generate random bytes
		_, err := rand.Read(bytes)
		if err != nil {
			return "", err
		}

		// Map random bytes to ID
		for i := 0; i < len(bytes) && idIndex < size; i++ {
			// Applying mask ensures uniform distribution
			alphabetIndex := int(bytes[i]) & mask

			// Skip if the index is out of range
			if alphabetIndex >= len(alphabet) {
				continue
			}

			// Add the character to the ID
			id[idIndex] = alphabet[alphabetIndex]
			idIndex++
		}
	}

	return string(id), nil
}
