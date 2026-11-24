package kernel

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SecurityHeaders applique une CSP stricte (aucune ressource externe) et les
// en-têtes de sécurité usuels.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// RateLimiter est un limiteur simple par clé (IP + chemin) : n requêtes par
// fenêtre glissante.
type RateLimiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time
	max     int
	window  time.Duration
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{hits: map[string][]time.Time{}, max: max, window: window}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-rl.window)
	var kept []time.Time
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rl.max {
		rl.hits[key] = kept
		return false
	}
	rl.hits[key] = append(kept, now)
	return true
}

// ClientIP extrait l'IP cliente (directe, le reverse proxy d'ARISE n'en ajoute
// pas nécessairement).
func ClientIP(r *http.Request) string {
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}
	return r.RemoteAddr
}

// VerifCSRF vérifie le champ "csrf" des POST contre la session, sauf pour les
// préfixes exemptés (webhook HelloAsso, signé par son secret d'URL).
func VerifCSRF(sessions *Sessions, exemptPrefixes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			for _, p := range exemptPrefixes {
				if strings.HasPrefix(r.URL.Path, p) {
					next.ServeHTTP(w, r)
					return
				}
			}
			sess, ok := sessions.Valide(r)
			if !ok {
				http.Error(w, "session invalide", http.StatusForbidden)
				return
			}
			if err := r.ParseForm(); err != nil || r.PostFormValue("csrf") != sess.CSRF {
				http.Error(w, "CSRF invalide", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
