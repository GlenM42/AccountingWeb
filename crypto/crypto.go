package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    = 2
	argonMemory  = 102400
	argonThreads = 8
	argonKeyLen  = 32 // 256-bit KEK
)

// DeriveKEK derives the Key Encryption Key from the user's password and salt.
// This must be called with the exact same parameters used during registration.
func DeriveKEK(password string, saltB64 string) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}

	kek := argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)
	return kek, nil
}

// UnwrapDEK decrypts the wrapped DEK using the KEK.
func UnwrapDEK(kek []byte, wrappedB64, ivB64 string) ([]byte, error) {
	wrapped, err := base64.StdEncoding.DecodeString(wrappedB64)
	if err != nil {
		return nil, fmt.Errorf("decode wrapped DEK: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(ivB64)
	if err != nil {
		return nil, fmt.Errorf("decode IV: %w", err)
	}

	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	dek, err := gcm.Open(nil, iv, wrapped, nil)
	if err != nil {
		// Don't leak whether it was a bad password or corrupted data
		return nil, fmt.Errorf("unwrap failed: %w", err)
	}

	return dek, nil
}

// EncryptField encrypts a plaintext string with AES-GCM using the provided DEK.
// A random 12-byte nonce is generated for each call — never reuse a nonce with the same key.
func EncryptField(dek []byte, plaintext string) (ct []byte, iv []byte, err error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes for GCM
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return ciphertext, nonce, nil
}

// DecryptField decrypts a single AES-GCM encrypted field.
// ct is ciphertext, iv is the initialization vector, both raw bytes.
func DecryptField(dek, ct, iv []byte) (string, error) {
	block, err := aes.NewCipher(dek)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, iv, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt field: %w", err)
	}

	return string(plaintext), nil
}
