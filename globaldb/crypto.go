package globaldb

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	keyFileName = "encryption.key"
	keySize     = 32 // AES-256
	// encPrefix marks a string as encrypted (vs plaintext JSON for backward compat).
	encPrefix = "enc:"
)

// getOrCreateKey returns the 32-byte AES-256 key, creating it if it doesn't exist.
// The key is stored alongside the database at ~/.orchestra/db/encryption.key with
// 0600 permissions so only the current user can read it.
func getOrCreateKey() ([]byte, error) {
	dbDir := filepath.Dir(globalDBPath())
	keyPath := filepath.Join(dbDir, keyFileName)

	// Try to read existing key.
	data, err := os.ReadFile(keyPath)
	if err == nil && len(data) == keySize {
		return data, nil
	}

	// Generate a new key.
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate encryption key: %w", err)
	}

	// Ensure directory exists.
	if err := os.MkdirAll(dbDir, 0700); err != nil {
		return nil, fmt.Errorf("create key directory: %w", err)
	}

	// Write with restrictive permissions.
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, fmt.Errorf("write encryption key: %w", err)
	}

	return key, nil
}

// encryptConfig encrypts a config map to a base64 string prefixed with "enc:".
func encryptConfig(config map[string]string) (string, error) {
	if len(config) == 0 {
		return "{}", nil
	}

	plaintext, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	key, err := getOrCreateKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return encPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptConfig decrypts a config string. If the string is not encrypted
// (no "enc:" prefix), it is treated as plaintext JSON for backward compatibility.
func decryptConfig(data string) (map[string]string, error) {
	config := make(map[string]string)

	if data == "" || data == "{}" {
		return config, nil
	}

	// Backward compatibility: plaintext JSON (no prefix).
	if len(data) > 0 && data[0] == '{' {
		json.Unmarshal([]byte(data), &config)
		return config, nil
	}

	// Must be encrypted.
	if len(data) <= len(encPrefix) || data[:len(encPrefix)] != encPrefix {
		// Unknown format — treat as empty.
		return config, nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(data[len(encPrefix):])
	if err != nil {
		return config, fmt.Errorf("decode base64: %w", err)
	}

	key, err := getOrCreateKey()
	if err != nil {
		return config, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return config, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return config, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return config, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return config, fmt.Errorf("decrypt: %w", err)
	}

	json.Unmarshal(plaintext, &config)
	return config, nil
}

// encryptString encrypts a plaintext string and returns a base64 "enc:" prefixed string.
func encryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := getOrCreateKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptString decrypts an "enc:" prefixed string. If the string has no prefix,
// it is returned as-is for backward compatibility.
func decryptString(data string) (string, error) {
	if data == "" {
		return "", nil
	}

	// No prefix — treat as plaintext (backward compat).
	if len(data) <= len(encPrefix) || data[:len(encPrefix)] != encPrefix {
		return data, nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(data[len(encPrefix):])
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	key, err := getOrCreateKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}
