package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

// generateTestRSAKey generates a 2048-bit RSA key pair for testing.
// Returns PEM-encoded private and public keys as strings.
func generateTestRSAKey(t *testing.T) (privPEM, pubPEM string) {
	t.Helper()

	// Generate 2048-bit RSA key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Encode private key to PEM
	privBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	privPEM = string(pem.EncodeToMemory(privBlock))

	// Encode public key to PEM
	pubBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pubPEM = string(pem.EncodeToMemory(pubBlock))

	return privPEM, pubPEM
}
