package hash

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// File computes the SHA-256 hash of the file at the given path.
func File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file for hash: %w", err)
	}
	return Bytes(data), nil
}

// Bytes computes the SHA-256 hash of the given data.
func Bytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
