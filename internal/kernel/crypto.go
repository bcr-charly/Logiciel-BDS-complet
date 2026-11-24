package kernel

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// HashPassword dérive un hash PBKDF2-SHA256 du mot de passe (stdlib uniquement).
func HashPassword(password string, salt []byte) ([]byte, error) {
	return pbkdf2.Key(sha256.New, password, salt, 100_000, 32)
}

// CreerComptePassword génère sel + hash, encodés en base64.
func CreerHashPassword(password string) (saltB64, hashB64 string, err error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}
	h, err := HashPassword(password, salt)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(salt), base64.StdEncoding.EncodeToString(h), nil
}

// VerifierPassword compare en temps constant.
func VerifierPassword(password, saltB64, hashB64 string) bool {
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return false
	}
	want, err := base64.StdEncoding.DecodeString(hashB64)
	if err != nil {
		return false
	}
	got, err := HashPassword(password, salt)
	if err != nil {
		return false
	}
	return hmac.Equal(got, want)
}

// Signe calcule HMAC-SHA256(secret, message) en hex.
func Signe(secret []byte, message string) string {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(message))
	return hex.EncodeToString(m.Sum(nil))
}

// VerifieSignature vérifie HMAC-SHA256 en temps constant.
func VerifieSignature(secret []byte, message, sig string) bool {
	want := Signe(secret, message)
	return hmac.Equal([]byte(want), []byte(sig))
}

// RandomID génère un identifiant aléatoire (hex 16 octets).
func RandomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// RandomToken génère un token aléatoire URL-safe (base64url).
func RandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
