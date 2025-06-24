package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// GenerateSignature generates a HMAC SHA256 signature for the given body using the provided secret key.
func GenerateSignature(body interface{}, secret string) (string, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(bodyBytes)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// ValidateSignature validates the HMAC SHA256 signature for the given body using the provided secret key.
func ValidateSignature(body interface{}, secret, signature string) (bool, error) {
	expectedSignature, err := GenerateSignature(body, secret)
	if err != nil {
		return false, err
	}
	return hmac.Equal([]byte(expectedSignature), []byte(signature)), nil
}
