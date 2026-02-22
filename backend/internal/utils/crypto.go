package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// Encrypt encrypts data using AES-256-GCM
// keyHex must be a 32-byte (64 char) hex string, or we fall back to using it as raw bytes if len=32
func Encrypt(plaintext []byte, keyHex string) ([]byte, error) {
	key, err := decodeKey(keyHex)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM
func Decrypt(ciphertext []byte, keyHex string) ([]byte, error) {
	key, err := decodeKey(keyHex)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func decodeKey(keyHex string) ([]byte, error) {
	if len(keyHex) == 32 {
		// Assume raw bytes (for dev convenience with short keys) - though strictly this is 128-bit security if ASCII
		// Or pads to 32 bytes?
		// AES-256 requires 32 bytes.
		// If string is 32 chars, that's 32 bytes.
		return []byte(keyHex), nil
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, err
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: got %d bytes, want 32", len(key))
	}

	return key, nil
}
