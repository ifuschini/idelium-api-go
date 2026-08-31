// Package integrations contains compatibility rules for outbound integration endpoints.
package integrations

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrInvalidApplicationKey = errors.New("integration application key is invalid")
var ErrInvalidCiphertext = errors.New("integration secret ciphertext is invalid")

type laravelPayload struct {
	IV    string `json:"iv"`
	Value string `json:"value"`
	MAC   string `json:"mac"`
	Tag   string `json:"tag"`
}

// ParseApplicationKey validates the Laravel APP_KEY representation without
// including key material in diagnostics.
func ParseApplicationKey(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "base64:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "base64:"))
		if err != nil || len(decoded) != 32 {
			return nil, ErrInvalidApplicationKey
		}
		return decoded, nil
	}
	if len(value) != 32 {
		return nil, ErrInvalidApplicationKey
	}
	return []byte(value), nil
}

// ApplicationKeyFromEnvironment loads APP_KEY from a value or mounted secret
// file without exposing key material in diagnostics.
func ApplicationKeyFromEnvironment() ([]byte, error) {
	value := os.Getenv("APP_KEY")
	filePath := os.Getenv("IDELIUM_APP_KEY_FILE")
	if filePath == "" {
		filePath = os.Getenv("APP_KEY_FILE")
	}
	if filePath != "" {
		contents, err := os.ReadFile(filePath)
		if err != nil {
			return nil, errors.New("integration application key file is not readable")
		}
		value = strings.TrimRight(string(contents), "\r\n")
	}
	key, err := ParseApplicationKey(value)
	if err != nil {
		return nil, errors.New("integration application key is not configured")
	}
	return key, nil
}

// EncryptLaravelString produces the AES-256-CBC envelope used by Laravel's
// Crypt::encryptString so the coexistence owner can decrypt Go-created rows.
func EncryptLaravelString(key []byte, plaintext string) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidApplicationKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", ErrInvalidApplicationKey
	}
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("generate integration secret initialization vector: %w", err)
	}
	padded := pkcs7Pad([]byte(plaintext), aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)
	encodedIV := base64.StdEncoding.EncodeToString(iv)
	encodedValue := base64.StdEncoding.EncodeToString(ciphertext)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(encodedIV + encodedValue))
	payload, err := json.Marshal(laravelPayload{IV: encodedIV, Value: encodedValue, MAC: hex.EncodeToString(mac.Sum(nil)), Tag: ""})
	if err != nil {
		return "", fmt.Errorf("encode integration secret envelope: %w", err)
	}
	return base64.StdEncoding.EncodeToString(payload), nil
}

// DecryptLaravelString validates and decrypts an existing Laravel envelope.
func DecryptLaravelString(key []byte, envelope string) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidApplicationKey
	}
	payloadJSON, err := base64.StdEncoding.DecodeString(envelope)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	var payload laravelPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return "", ErrInvalidCiphertext
	}
	iv, err := base64.StdEncoding.DecodeString(payload.IV)
	if err != nil || len(iv) != aes.BlockSize {
		return "", ErrInvalidCiphertext
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Value)
	if err != nil || len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return "", ErrInvalidCiphertext
	}
	expectedMAC, err := hex.DecodeString(payload.MAC)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(payload.IV + payload.Value))
	if !hmac.Equal(expectedMAC, mac.Sum(nil)) {
		return "", ErrInvalidCiphertext
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", ErrInvalidApplicationKey
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)
	plaintext, err = pkcs7Unpad(plaintext, aes.BlockSize)
	if err != nil {
		return "", ErrInvalidCiphertext
	}
	return string(plaintext), nil
}

func pkcs7Pad(value []byte, blockSize int) []byte {
	padding := blockSize - len(value)%blockSize
	result := make([]byte, len(value)+padding)
	copy(result, value)
	for index := len(value); index < len(result); index++ {
		result[index] = byte(padding)
	}
	return result
}

func pkcs7Unpad(value []byte, blockSize int) ([]byte, error) {
	if len(value) == 0 || len(value)%blockSize != 0 {
		return nil, ErrInvalidCiphertext
	}
	padding := int(value[len(value)-1])
	if padding < 1 || padding > blockSize || padding > len(value) {
		return nil, ErrInvalidCiphertext
	}
	for _, item := range value[len(value)-padding:] {
		if int(item) != padding {
			return nil, ErrInvalidCiphertext
		}
	}
	return value[:len(value)-padding], nil
}
