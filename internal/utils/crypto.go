package utils

import (
	"crypto/rand"
	"math/big"
)

// RandomInt generates a cryptographically secure random integer in the range [0, max)
func RandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		// Fallback to less secure method if crypto/rand fails
		// This should rarely happen
		panic("failed to generate random number: " + err.Error())
	}
	
	return int(n.Int64())
}

// GenerateRandomBytes generates n random bytes
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}