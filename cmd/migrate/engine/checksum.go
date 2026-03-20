package engine

import (
	"crypto/sha256"
	"fmt"
	"os"
)

// FileChecksum computes the SHA-256 hex digest of a file's contents.
func FileChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file for checksum: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}
