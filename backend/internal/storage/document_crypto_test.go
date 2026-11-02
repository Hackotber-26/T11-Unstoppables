package storage

import (
	"bytes"
	"testing"
)

func TestDriverDocumentEncryptionRoundTripAndAuthentication(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	document := []byte("%PDF-1.7 private driver identity")
	encrypted, err := encryptDriverDocument(document, key)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, document) {
		t.Fatal("encrypted file contains the plaintext document")
	}
	decrypted, err := decryptDriverDocument(encrypted, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, document) {
		t.Fatalf("decrypted document = %q, want %q", decrypted, document)
	}

	encrypted[len(encrypted)-1] ^= 0xff
	if _, err := decryptDriverDocument(encrypted, key); err == nil {
		t.Fatal("expected modified ciphertext to fail authentication")
	}
}

func TestDriverDocumentEncryptionRejectsWrongKeyAndUnencryptedFiles(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	encrypted, err := encryptDriverDocument([]byte("document"), key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decryptDriverDocument(encrypted, bytes.Repeat([]byte{2}, 32)); err == nil {
		t.Fatal("expected wrong key to fail authentication")
	}
	if _, err := decryptDriverDocument([]byte("plaintext"), key); err == nil {
		t.Fatal("expected unencrypted legacy file to be rejected when encryption is enabled")
	}
}

func TestDriverDocumentStorageRemainsPlaintextWithoutDevelopmentKey(t *testing.T) {
	document := []byte("development document")
	stored, err := encryptDriverDocument(document, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, document) {
		t.Fatal("development storage without a key should preserve local demo behavior")
	}
}
