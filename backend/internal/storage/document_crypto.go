package storage

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

var documentEncryptionHeader = []byte("NIMDOC1")

func encryptDriverDocument(data, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return data, nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("configure driver document encryption: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("configure driver document encryption: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate driver document encryption nonce: %w", err)
	}
	encrypted := append([]byte{}, documentEncryptionHeader...)
	encrypted = append(encrypted, nonce...)
	return gcm.Seal(encrypted, nonce, data, documentEncryptionHeader), nil
}

func decryptDriverDocument(data, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return data, nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("configure driver document decryption: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("configure driver document decryption: %w", err)
	}
	headerSize := len(documentEncryptionHeader)
	nonceEnd := headerSize + gcm.NonceSize()
	if len(data) < nonceEnd || !bytes.Equal(data[:headerSize], documentEncryptionHeader) {
		return nil, errors.New("stored driver document is not encrypted with the configured key")
	}
	return gcm.Open(nil, data[headerSize:nonceEnd], data[nonceEnd:], documentEncryptionHeader)
}
