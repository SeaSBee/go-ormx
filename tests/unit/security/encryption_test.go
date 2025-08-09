package security_test

import (
	"crypto/rand"
	"testing"

	"go-ormx/ormx/security"
)

func generateKey(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	return b
}

func setupDefaultEncryptor(t *testing.T) *security.FieldEncryptor {
	t.Helper()
	cfg := &security.EncryptionConfig{
		MasterKey:               generateKey(t, 32),
		KeySalt:                 generateKey(t, 16),
		Algorithm:               "AES-256-GCM",
		KeyRotationEnabled:      false,
		UseHardwareAcceleration: true,
	}
	encryptor, err := security.NewFieldEncryptor(cfg)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}
	security.SetDefaultEncryptor(encryptor)
	return encryptor
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	encryptor := setupDefaultEncryptor(t)

	plaintext := "super-secret"
	enc, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if enc == nil || enc.Data == "" || enc.Nonce == "" {
		t.Fatalf("expected non-empty encrypted field")
	}

	dec, err := encryptor.Decrypt(enc)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if dec != plaintext {
		t.Fatalf("round-trip mismatch: got %q want %q", dec, plaintext)
	}
}

func TestEncryptedString_ErrorWithoutDefaultEncryptor(t *testing.T) {
	// Ensure no default set
	security.SetDefaultEncryptor(nil)

	es := security.NewEncryptedString(nil)
	if err := es.Set("secret"); err == nil {
		t.Fatalf("expected error when no encryptor configured")
	}
}

func TestEncryptedString_UsesDefaultEncryptor(t *testing.T) {
	setupDefaultEncryptor(t)
	es := security.NewEncryptedString(nil)
	if err := es.Set("abc123"); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	val, err := es.Get()
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if val != "abc123" {
		t.Fatalf("got %q want %q", val, "abc123")
	}
}

func TestEncryptSensitiveData_Detection(t *testing.T) {
	encryptor := setupDefaultEncryptor(t)
	input := map[string]interface{}{
		"username": "alice",
		"password": "p@ss",
		"token":    "tkn",
		"age":      30,
	}
	out, err := security.EncryptSensitiveData(encryptor, input)
	if err != nil {
		t.Fatalf("EncryptSensitiveData failed: %v", err)
	}
	if _, ok := out["password"].(*security.EncryptedField); !ok {
		t.Fatalf("expected password to be encrypted field")
	}
	if _, ok := out["token"].(*security.EncryptedField); !ok {
		t.Fatalf("expected token to be encrypted field")
	}
	if out["username"].(string) != "alice" {
		t.Fatalf("username changed unexpectedly")
	}
}
