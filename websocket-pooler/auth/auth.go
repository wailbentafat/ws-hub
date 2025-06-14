package auth

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)
var JwtSecretKey = []byte("3jzt ndiro f environement variable")





func GenerateToken(w http.ResponseWriter, r *http.Request) {
	userID := "user123"

	tokenString, err := createSignedToken(userID)
	if err != nil {
		log.Printf("Error creating signed token: %v", err)
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(tokenString))
}

func createSignedToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,                                 
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iat": time.Now().Unix(),                       
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString(JwtSecretKey)
	if err != nil {
		return "", fmt.Errorf("could not sign token: %w", err)
	}

	return signedToken, nil
}