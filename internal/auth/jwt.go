package auth

import (
	"fmt"
	"os"
	
	"github.com/golang-jwt/jwt/v5"
)

func ValidateJWT(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected singing method")
		}

		return []byte(os.Getenv("SECRET_KEY")), nil
	})

	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)
	userID, ok := claims["id"].(string)
	if !ok {
		return "", fmt.Errorf("no userId in token")
	}

	return userID, nil
}