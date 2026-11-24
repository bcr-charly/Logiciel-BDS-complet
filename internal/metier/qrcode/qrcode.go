// Package qrcode gère les QR d'inscription (phase 2) : le contenu est signé
// HMAC-SHA256 avec le secret de conf.txt (inscription, adhérent, événement,
// expiration). La génération de l'image QR viendra en phase 2 ; le token signé
// et sa vérification sont déjà en place.
package qrcode

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

type Payload struct {
	InscriptionID string    `json:"ins"`
	CompteID      string    `json:"cpt"`
	EvenementID   string    `json:"evt"`
	Exp           time.Time `json:"exp"`
}

var (
	ErrTokenInvalide = errors.New("token invalide")
	ErrTokenExpire    = errors.New("token expiré")
)

type Service struct {
	secret []byte
}

func New(secret []byte) *Service {
	return &Service{secret: secret}
}

// Generer produit "base64url(payload).base64url(hmac)".
func (s *Service) Generer(inscriptionID, compteID, evenementID string, exp time.Time) string {
	p := Payload{InscriptionID: inscriptionID, CompteID: compteID, EvenementID: evenementID, Exp: exp}
	b, _ := json.Marshal(p)
	enc := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(enc))
	return enc + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Verifier valide la signature et l'expiration du token.
func (s *Service) Verifier(token string) (*Payload, error) {
	enc, sig, ok := cutDot(token)
	if !ok {
		return nil, ErrTokenInvalide
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(enc))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(want, got) {
		return nil, ErrTokenInvalide
	}
	b, err := base64.RawURLEncoding.DecodeString(enc)
	if err != nil {
		return nil, ErrTokenInvalide
	}
	var p Payload
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, ErrTokenInvalide
	}
	if time.Now().After(p.Exp) {
		return nil, ErrTokenExpire
	}
	return &p, nil
}

func cutDot(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}
