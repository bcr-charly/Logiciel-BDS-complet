package kernel

import (
	"net/http"
	"time"
)

// Session est une session persistée (data/sessions.json). Le cookie contient
// "id.signature" où signature = HMAC-SHA256(secret, id).
type Session struct {
	ID        string    `json:"id"`
	CompteID  string    `json:"compte_id,omitempty"`
	CSRF      string    `json:"csrf"`
	CreatedAt time.Time `json:"created_at"`
	ExpireAt  time.Time `json:"expire_at"`
}

const SessionCookie = "bds_session"

type Sessions struct {
	col    *Collection[Session]
	secret []byte
	duree  time.Duration
}

func NewSessions(store *Store, secret []byte, duree time.Duration) (*Sessions, error) {
	col, err := NewCollection[Session](store, "sessions")
	if err != nil {
		return nil, err
	}
	return &Sessions{col: col, secret: secret, duree: duree}, nil
}

// Creer crée une session (anonyme ou liée à un compte) et renvoie le cookie.
func (s *Sessions) Creer(w http.ResponseWriter, compteID string) (Session, error) {
	now := time.Now()
	sess := Session{
		ID:        RandomID(),
		CompteID:  compteID,
		CSRF:      RandomToken(),
		CreatedAt: now,
		ExpireAt:  now.Add(s.duree),
	}
	if err := s.col.Append(sess); err != nil {
		return sess, err
	}
	s.SetCookie(w, sess)
	return sess, nil
}

func (s *Sessions) SetCookie(w http.ResponseWriter, sess Session) {
	val := sess.ID + "." + Signe(s.secret, sess.ID)
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie,
		Value:    val,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpireAt,
	})
}

// Valide lit le cookie de requête et renvoie la session si valide.
func (s *Sessions) Valide(r *http.Request) (Session, bool) {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return Session{}, false
	}
	id, sig, ok := cut(c.Value, ".")
	if !ok || !VerifieSignature(s.secret, id, sig) {
		return Session{}, false
	}
	sess, found := s.col.Find(func(x Session) bool { return x.ID == id })
	if !found || time.Now().After(sess.ExpireAt) {
		return Session{}, false
	}
	return sess, true
}

// Courante renvoie la session de la requête, ou la crée (anonyme) si absente.
func (s *Sessions) Courante(w http.ResponseWriter, r *http.Request) (Session, error) {
	if sess, ok := s.Valide(r); ok {
		return sess, nil
	}
	return s.Creer(w, "")
}

// Remplacer met à jour la session (ex. après connexion) et réécrit le cookie.
func (s *Sessions) Remplacer(w http.ResponseWriter, sess Session) error {
	sess.CSRF = RandomToken()
	sess.ExpireAt = time.Now().Add(s.duree)
	if _, _, err := s.col.UpdateWhere(func(x Session) bool { return x.ID == sess.ID }, func(x *Session) {
		*x = sess
	}); err != nil {
		return err
	}
	s.SetCookie(w, sess)
	return nil
}

func (s *Sessions) Detruire(w http.ResponseWriter, id string) {
	s.col.DeleteWhere(func(x Session) bool { return x.ID == id })
	http.SetCookie(w, &http.Cookie{Name: SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
}

// PurgeExpired supprime les sessions expirées.
func (s *Sessions) PurgeExpired() {
	now := time.Now()
	s.col.DeleteWhere(func(x Session) bool { return now.After(x.ExpireAt) })
}

func cut(s, sep string) (string, string, bool) {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}
