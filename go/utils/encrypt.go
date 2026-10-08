package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Decrypt the given base64 encrypted data using the given key
func Decrypt(dataBase64 string, key []byte) ([]byte, error) {
	// Base64 decode the ciphertext first
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: must be 16, 24, or 32 bytes")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Since we know the ciphertext is actually nonce+ciphertext
	// And len(nonce) == NonceSize(). We can separate the two.
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	decoded, err := gcm.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	if err != nil {
		return nil, fmt.Errorf("message authentication failed: %v", err)
	}

	return decoded, nil
}

// Encrypt the given data using the given key
// key must be a 16, 24 or 32 byte key.
// This returns the encrypted data in base64 encoded format.
func Encrypt(data []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return "", err
	}
	encData := gcm.Seal(nonce, nonce, data, nil)
	encDataBase64 := base64.StdEncoding.EncodeToString(encData)
	return encDataBase64, nil
}
