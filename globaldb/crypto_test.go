package globaldb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// Use a temp dir so we don't pollute the real key store.
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Ensure globaldb dir exists for key storage.
	dbDir := filepath.Join(tmpDir, ".orchestra", "db")
	os.MkdirAll(dbDir, 0700)

	config := map[string]string{
		"ANTHROPIC_API_KEY": "sk-ant-test123",
		"OPENAI_API_KEY":    "sk-test456",
	}

	encrypted, err := encryptConfig(config)
	if err != nil {
		t.Fatalf("encryptConfig: %v", err)
	}

	// Must have "enc:" prefix.
	if len(encrypted) < 4 || encrypted[:4] != encPrefix {
		t.Fatalf("expected enc: prefix, got %q", encrypted[:10])
	}

	// Decrypt and verify.
	decrypted, err := decryptConfig(encrypted)
	if err != nil {
		t.Fatalf("decryptConfig: %v", err)
	}

	if decrypted["ANTHROPIC_API_KEY"] != "sk-ant-test123" {
		t.Errorf("ANTHROPIC_API_KEY = %q, want %q", decrypted["ANTHROPIC_API_KEY"], "sk-ant-test123")
	}
	if decrypted["OPENAI_API_KEY"] != "sk-test456" {
		t.Errorf("OPENAI_API_KEY = %q, want %q", decrypted["OPENAI_API_KEY"], "sk-test456")
	}
}

func TestDecryptConfig_BackwardCompat_PlaintextJSON(t *testing.T) {
	config, err := decryptConfig(`{"API_KEY":"abc123"}`)
	if err != nil {
		t.Fatalf("decryptConfig plaintext: %v", err)
	}
	if config["API_KEY"] != "abc123" {
		t.Errorf("API_KEY = %q, want %q", config["API_KEY"], "abc123")
	}
}

func TestDecryptConfig_EmptyInputs(t *testing.T) {
	for _, input := range []string{"", "{}"} {
		config, err := decryptConfig(input)
		if err != nil {
			t.Fatalf("decryptConfig(%q): %v", input, err)
		}
		if len(config) != 0 {
			t.Errorf("decryptConfig(%q) returned %d keys, want 0", input, len(config))
		}
	}
}

func TestDecryptConfig_UnknownFormat(t *testing.T) {
	// Non-JSON, non-enc: prefix — treated as empty.
	config, err := decryptConfig("garbage")
	if err != nil {
		t.Fatalf("decryptConfig(garbage): %v", err)
	}
	if len(config) != 0 {
		t.Errorf("expected empty config for unknown format, got %d keys", len(config))
	}
}

func TestEncryptConfig_EmptyMap(t *testing.T) {
	result, err := encryptConfig(map[string]string{})
	if err != nil {
		t.Fatalf("encryptConfig(empty): %v", err)
	}
	if result != "{}" {
		t.Errorf("empty config should return '{}', got %q", result)
	}
}

func TestEncryptConfig_DifferentCiphertexts(t *testing.T) {
	// Each encryption should produce different ciphertext (random nonce).
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, ".orchestra", "db"), 0700)

	config := map[string]string{"key": "value"}

	enc1, err := encryptConfig(config)
	if err != nil {
		t.Fatalf("encrypt 1: %v", err)
	}
	enc2, err := encryptConfig(config)
	if err != nil {
		t.Fatalf("encrypt 2: %v", err)
	}

	if enc1 == enc2 {
		t.Error("two encryptions of the same data should differ (random nonce)")
	}

	// Both should decrypt to the same value.
	d1, _ := decryptConfig(enc1)
	d2, _ := decryptConfig(enc2)
	if d1["key"] != d2["key"] {
		t.Error("decrypted values should match")
	}
}

func TestGetOrCreateKey_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, ".orchestra", "db"), 0700)

	key1, err := getOrCreateKey()
	if err != nil {
		t.Fatalf("getOrCreateKey 1: %v", err)
	}
	if len(key1) != keySize {
		t.Fatalf("key size = %d, want %d", len(key1), keySize)
	}

	// Second call should return the same key.
	key2, err := getOrCreateKey()
	if err != nil {
		t.Fatalf("getOrCreateKey 2: %v", err)
	}
	if string(key1) != string(key2) {
		t.Error("key should be persistent across calls")
	}
}

func TestGetOrCreateKey_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, ".orchestra", "db"), 0700)

	_, err := getOrCreateKey()
	if err != nil {
		t.Fatalf("getOrCreateKey: %v", err)
	}

	keyPath := filepath.Join(tmpDir, ".orchestra", "db", keyFileName)
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key file: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("key file permissions = %o, want 0600", perm)
	}
}
