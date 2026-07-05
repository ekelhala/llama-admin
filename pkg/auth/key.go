package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateKey(prefix string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	return prefix + "-" + hex.EncodeToString(b), nil
}
