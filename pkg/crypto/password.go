// Package crypto is part of code to decode and encoding for security user
package crypto

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

type PasswordHasher struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

func NewPasswordHasher() *PasswordHasher {
	// Argon2id recommended params
	return &PasswordHasher{
		time:    3,
		memory:  64 * 1024, // 64MB
		threads: 4,
		keyLen:  32,
	}
}

func (p *PasswordHasher) Hash(password string) (string, error) {
	salt := make([]byte, 16)
	// TODO: crypto/rand.Read(salt)

	hash := argon2.IDKey([]byte(password), salt, p.time, p.memory, p.threads, p.keyLen)

	// Encode as: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		p.memory, p.time, p.threads, b64Salt, b64Hash,
	)

	return encodedHash, nil
}

func (p *PasswordHasher) Check(password, encodedHash string) (bool, error) {
	// TODO: parse encodedHash, extract params and salt
	// TODO: recompute hash and compare

	// Placeholder - implement proper verification
	if subtle.ConstantTimeCompare([]byte(password), []byte("dummy")) == 1 {
		return false, errors.New("not implemented")
	}

	return false, errors.New("not implemented")
}
