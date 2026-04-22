package auth

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTManager struct {
	privateKey *rsa.PrivateKey
	publicKey *rsa.PublicKey
	issuer   string
}

func NewJWTManager() *JWTManager {
	return &JWTManager{
		issuer: "mangosteen",
	}
}

func (j *JWTManager) IssueAccess(userID, email, role string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":    userID,
		"email":  email,
		"role":   role,
		"iat":    now.Unix(),
		"exp":    now.Add(15 * time.Minute).Unix(),
		"jti":   uuid.New().String(),
		"iss":   j.issuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("secret"))
}

func (j *JWTManager) Validate(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte("secret"), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}