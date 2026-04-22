package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
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

type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey *rsa.PublicKey
	issuer   string
	kid      string
}

func NewJWTManager(cfg *config.JWTConfig) (*JWTManager, error) {
	mgr := &JWTManager{
		issuer: cfg.Issuer,
		kid:   uuid.New().String()[:8],
	}

	if cfg.PrivateKeyPEM != "" {
		key, err := parsePrivateKey(cfg.PrivateKeyPEM)
		if err != nil {
			return nil, err
		}
		mgr.privateKey = key
	}

	if cfg.PublicKeyPEM != "" {
		key, err := parsePublicKey(cfg.PublicKeyPEM)
		if err != nil {
			return nil, err
		}
		mgr.publicKey = key
	}

	return mgr, nil
}

func parsePrivateKey(pemKey string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("invalid private key")
	}
	return jwt.ParseRSAPrivateKeyFromPEM([]byte(pemKey))
}

func parsePublicKey(pemKey string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("invalid public key")
	}
	return jwt.ParseRSAPublicKeyFromPEM([]byte(pemKey))
}

func (j *JWTManager) IssueAccess(userID, email, role string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
		"jti":  uuid.New().String(),
		"iss":  j.issuer,
	}

	var token *jwt.Token
	if j.privateKey != nil {
		token = jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		return token.SignedString(j.privateKey)
	}

	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("secret"))
}

func (j *JWTManager) IssueRefresh() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  now.Add(7 * 24 * time.Hour).Unix(),
		"jti":  uuid.New().String(),
		"iss":  j.issuer,
	}

	var token *jwt.Token
	if j.privateKey != nil {
		token = jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		return token.SignedString(j.privateKey)
	}

	token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("secret"))
}

func (j *JWTManager) Validate(tokenString string) (jwt.MapClaims, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); ok && j.publicKey != nil {
			return j.publicKey, nil
		}
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
	keys := make([]JWK, 0)
	if j.publicKey != nil {
		keys = append(keys, j.toJWK())
	}
	return JWKS{Keys: keys}
}

func (j *JWTManager) toJWK() JWK {
	return JWK{
		Kid: j.kid,
		Kty: "RSA",
		Alg: "RS256",
		Use: "sig",
		N:   base64.RawURLEncoding.EncodeToString(j.publicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(j.publicKey.E)).Bytes()),
	}
}