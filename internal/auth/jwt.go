package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"mangosteen/config"
)

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type keyEntry struct {
	kid        string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	createdAt  time.Time
}

type JWTManager struct {
	mu        sync.RWMutex
	keys      []keyEntry
	activeKid string
	issuer    string
}

func NewJWTManager(cfg *config.JWTConfig) (*JWTManager, error) {
	mgr := &JWTManager{
		issuer: cfg.Issuer,
	}

	// Get PEM content — try file path first, then raw PEM
	privPEM := cfg.PrivateKeyPEM
	pubPEM := cfg.PublicKeyPEM

	if privPEM != "" && strings.HasSuffix(privPEM, ".pem") {
		data, err := os.ReadFile(privPEM)
		if err != nil {
			return nil, fmt.Errorf("read private key file: %w", err)
		}
		privPEM = string(data)
	}
	if pubPEM != "" && strings.HasSuffix(pubPEM, ".pem") {
		data, err := os.ReadFile(pubPEM)
		if err != nil {
			return nil, fmt.Errorf("read public key file: %w", err)
		}
		pubPEM = string(data)
	}

	// Parse initial keys
	if privPEM != "" && pubPEM != "" {
		privKey, err := parsePrivateKey(privPEM)
		if err != nil {
			return nil, err
		}
		pubKey, err := parsePublicKey(pubPEM)
		if err != nil {
			return nil, err
		}

		kid := uuid.New().String()[:8]
		mgr.keys = append(mgr.keys, keyEntry{
			kid:        kid,
			privateKey: privKey,
			publicKey:  pubKey,
			createdAt:  time.Now(),
		})
		mgr.activeKid = kid
	}

	return mgr, nil
}

func parsePrivateKey(pemKey string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, errors.New("invalid private key PEM")
	}
	// PKCS#1: "RSA PRIVATE KEY"
	if block.Type == "RSA PRIVATE KEY" {
		return jwt.ParseRSAPrivateKeyFromPEM([]byte(pemKey))
	}
	// PKCS#8: "PRIVATE KEY"
	if block.Type == "PRIVATE KEY" {
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}
		return rsaKey, nil
	}
	return nil, fmt.Errorf("unsupported key type: %s", block.Type)
}

func parsePublicKey(pemKey string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil {
		return nil, errors.New("invalid public key PEM")
	}
	// PKCS#1: "RSA PUBLIC KEY"
	if block.Type == "RSA PUBLIC KEY" {
		return jwt.ParseRSAPublicKeyFromPEM([]byte(pemKey))
	}
	// PKIX/SPKI: "PUBLIC KEY"
	if block.Type == "PUBLIC KEY" {
		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}
	return nil, fmt.Errorf("unsupported public key type: %s", block.Type)
}

func (j *JWTManager) IssueAccess(userID, email, role string) (string, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"aud":   "mangosteen",
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
		"jti":   uuid.New().String(),
		"iss":   j.issuer,
	}

	// Use active key
	if len(j.keys) > 0 {
		activeKey := j.keys[0]
		for _, k := range j.keys {
			if k.kid == j.activeKid {
				activeKey = k
				break
			}
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = activeKey.kid
		return token.SignedString(activeKey.privateKey)
	}

	// Fallback to HMAC if no RSA keys configured
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("secret"))
}

func (j *JWTManager) IssueRefresh() (string, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	now := time.Now()
	claims := jwt.MapClaims{
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  now.Add(7 * 24 * time.Hour).Unix(),
		"jti":  uuid.New().String(),
		"iss":  j.issuer,
	}

	if len(j.keys) > 0 {
		activeKey := j.keys[0]
		for _, k := range j.keys {
			if k.kid == j.activeKid {
				activeKey = k
				break
			}
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		token.Header["kid"] = activeKey.kid
		return token.SignedString(activeKey.privateKey)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("secret"))
}

func (j *JWTManager) Validate(tokenString string) (jwt.MapClaims, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		// Try RSA first
		if _, ok := token.Method.(*jwt.SigningMethodRSA); ok {
			// Look up key by kid
			kid, ok := token.Header["kid"].(string)
			if !ok {
				// Fallback to first key if no kid
				if len(j.keys) > 0 {
					return j.keys[0].publicKey, nil
				}
				return nil, fmt.Errorf("no RSA keys configured")
			}

			for _, k := range j.keys {
				if k.kid == kid {
					return k.publicKey, nil
				}
			}
			return nil, fmt.Errorf("key not found: %s", kid)
		}

		// HMAC fallback
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
			return []byte("secret"), nil
		}

		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}

	token, err := jwt.Parse(tokenString, keyFunc)
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func (j *JWTManager) GetJWKS() JWKS {
	j.mu.RLock()
	defer j.mu.RUnlock()

	keys := make([]JWK, 0, len(j.keys))
	for _, k := range j.keys {
		if k.publicKey != nil {
			keys = append(keys, j.toJWK(k))
		}
	}
	return JWKS{Keys: keys}
}

func (j *JWTManager) toJWK(entry keyEntry) JWK {
	return JWK{
		Kid: entry.kid,
		Kty: "RSA",
		Alg: "RS256",
		Use: "sig",
		N:   base64.RawURLEncoding.EncodeToString(entry.publicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(entry.publicKey.E)).Bytes()),
	}
}

// RotateKeys adds a new key pair and optionally removes old keys
func (j *JWTManager) RotateKeys(newPrivateKeyPEM, newPublicKeyPEM string) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	privKey, err := parsePrivateKey(newPrivateKeyPEM)
	if err != nil {
		return err
	}
	pubKey, err := parsePublicKey(newPublicKeyPEM)
	if err != nil {
		return err
	}

	kid := uuid.New().String()[:8]
	j.keys = append(j.keys, keyEntry{
		kid:        kid,
		privateKey: privKey,
		publicKey:  pubKey,
		createdAt:  time.Now(),
	})
	j.activeKid = kid

	// Keep only last 3 keys for rotation
	if len(j.keys) > 3 {
		j.keys = j.keys[len(j.keys)-3:]
	}

	return nil
}
