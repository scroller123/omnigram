package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var errInvalidToken = errors.New("invalid or expired token")

type Claims struct {
	jwt.RegisteredClaims
}

func jwtSecret() []byte {
	return []byte(os.Getenv("JWT_SECRET"))
}

// IssueToken creates a signed JWT valid for 7 days.
func IssueToken() (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "omnigram",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtSecret())
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT string.
func ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtSecret(), nil
	})
	if err != nil || !token.Valid {
		return nil, errInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errInvalidToken
	}
	return claims, nil
}
