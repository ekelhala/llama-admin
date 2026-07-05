package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	time    = 1
	memory  = 64 * 1024
	threads = 4
	keyLen  = 32
	saltLen = 16
)

func HashKey(key string) (string, error) {
	salt, err := generateSalt(saltLen)
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(key), salt, time, memory, threads, keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memory, time, threads,
		base64Encode(salt), base64Encode(hash)), nil
}

func VerifyKey(providedKey, hashedKey string) error {
	salt, hash, err := parseArgon2(hashedKey)
	if err != nil {
		return fmt.Errorf("parse hash: %w", err)
	}

	otherHash := argon2.IDKey([]byte(providedKey), salt, time, memory, threads, keyLen)
	if subtle.ConstantTimeCompare(hash, otherHash) != 1 {
		return fmt.Errorf("invalid API key")
	}
	return nil
}

func generateSalt(length int) ([]byte, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func parseArgon2(hashedKey string) ([]byte, []byte, error) {
	parts := strings.Split(hashedKey, "$")
	if len(parts) != 6 {
		return nil, nil, fmt.Errorf("invalid argon2 hash format")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, fmt.Errorf("decode salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, fmt.Errorf("decode hash: %w", err)
	}

	return salt, hash, nil
}

func base64Encode(b []byte) string {
	return base64.RawStdEncoding.EncodeToString(b)
}
