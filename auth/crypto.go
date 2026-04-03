package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// NewAESKeyFromHex parses a 32-byte (64 hex chars) key from env.
func NewAESKeyFromHex(hexKey string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("ENCRYPTION_SECRET is not valid hex: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_SECRET must be exactly 32 bytes (64 hex chars), got %d", len(key))
	}
	return key, nil
}

// DeriveKey derives a 32-byte AES key from a base key and a session ID.
func DeriveKey(baseKey []byte, sessionID string) []byte {
	// Using PBKDF2 with sessionID as salt.
	// 4096 iterations for balance between security and performance on every request?
	// Actually we might want just a single expensive derivation or many fast ones.
	// Since we might call this frequently, 1024 or even simple SHA256 might be enough
	// if the baseKey is already high entropy.
	return pbkdf2.Key(baseKey, []byte(sessionID), 1024, 32, sha256.New)
}

// Encrypt encrypts plaintext using AES-256-GCM. Returns ciphertext and nonce separately.
func Encrypt(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts AES-256-GCM ciphertext using the given key and nonce.
func Decrypt(key, ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (bad key or corrupted data): %w", err)
	}
	return plaintext, nil
}
