package app_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"bds/internal/app"
	"bds/internal/kernel"
	"bds/internal/pages"
)

// setupApp démarre une app complète dans un répertoire temporaire.
func setupApp(t *testing.T) *app.App {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	conf := "session_secret=secret-de-test-32-caracteres-ok\nport=0\nadmin_email=admin@test.fr\nadmin_password=admin12345\n"
	if err := os.WriteFile(filepath.Join(dir, "conf.txt"), []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := app.Init("conf.txt", "", ""); err != nil {
		t.Fatal(err)
	}
	pages.RegisterAll(app.A)
	return app.A
}

func TestPagesPubliques(t *testing.T) {
	a := setupApp(t)
	h := a.Handler()

	for _, path := range []string{"/", "/actus", "/sports", "/bds", "/evenements", "/boutique", "/inscription", "/connexion", "/healthz", "/static/style.css", "/static/logo-bds.jpg", "/static/app.js"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s -> %d", path, rec.Code)
		}
	}
	// 404
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/nexiste-pas", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("404 attendu, got %d", rec.Code)
	}
	// CSP présent
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") {
		t.Fatalf("CSP manquante: %q", csp)
	}
	// /admin protégé : redirection vers /connexion
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/admin", nil))
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "/connexion") {
		t.Fatalf("/admin -> %d %q", rec.Code, rec.Header().Get("Location"))
	}
}

// TestFluxComplet : inscription -> connexion -> profil -> inscription à un
// événement gratuit -> admin valide.
func TestFluxComplet(t *testing.T) {
	a := setupApp(t)
	h := a.Handler()

	csrfRe := regexp.MustCompile(`name="csrf" value="([^"]+)"`)

	get := func(path string, cookies []*http.Cookie) (*httptest.ResponseRecorder, []*http.Cookie) {
		req := httptest.NewRequest("GET", path, nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec, append(cookies, rec.Result().Cookies()...)
	}
	post := func(path string, form url.Values, cookies []*http.Cookie) (*httptest.ResponseRecorder, []*http.Cookie) {
		req := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec, append(cookies, rec.Result().Cookies()...)
	}
	csrfOf := func(rec *httptest.ResponseRecorder) string {
		m := csrfRe.FindStringSubmatch(rec.Body.String())
		if m == nil {
			t.Fatal("csrf introuvable dans la page")
		}
		return m[1]
	}

	// 1. GET /inscription -> session anonyme + csrf
	rec, cookies := get("/inscription", nil)
	if rec.Code != 200 {
		t.Fatalf("GET /inscription -> %d", rec.Code)
	}
	csrf := csrfOf(rec)

	// 2. POST /inscription
	form := url.Values{"csrf": {csrf}, "email": {"test@ensiie.fr"}, "login": {"test"}, "password": {"motdepasse123"}, "password2": {"motdepasse123"}, "nom": {"Test"}, "prenom": {"Jean"}, "telephone": {""}, "annee": {"1A"}, "sport": {"football"}}
	rec, cookies = post("/inscription", form, cookies)
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "/connexion") {
		t.Fatalf("POST /inscription -> %d %q", rec.Code, rec.Header().Get("Location"))
	}

	// 3. POST /connexion (sans CSRF valide -> 403 d'abord)
	form = url.Values{"csrf": {"faux"}, "ident": {"test"}, "password": {"motdepasse123"}}
	rec, _ = post("/connexion", form, cookies)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("CSRF invalide devrait donner 403, got %d", rec.Code)
	}
	rec, cookies = get("/connexion", cookies)
	csrf = csrfOf(rec)
	form = url.Values{"csrf": {csrf}, "ident": {"test"}, "password": {"motdepasse123"}}
	rec, cookies = post("/connexion", form, cookies)
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "/profil") {
		t.Fatalf("POST /connexion -> %d %q", rec.Code, rec.Header().Get("Location"))
	}

	// 4. GET /profil connecté
	rec, _ = get("/profil", cookies)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "test@ensiie.fr") {
		t.Fatalf("GET /profil -> %d", rec.Code)
	}

	// 5. inscription à l'événement gratuit "Cross de l'ENSIIE" (seed)
	evt := ""
	for _, e := range a.Evenements.Tous() {
		if e.Tarif == 0 {
			evt = e.ID
		}
	}
	if evt == "" {
		t.Fatal("événement gratuit de seed introuvable")
	}
	rec, cookies = get("/evenements/"+evt, cookies)
	csrf = csrfOf(rec)
	form = url.Values{"csrf": {csrf}}
	rec, _ = post("/evenements/"+evt+"/inscrire", form, cookies)
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "flash_inscrit_ok") {
		t.Fatalf("inscription event -> %d %q", rec.Code, rec.Header().Get("Location"))
	}

	// 6. admin : connexion avec le compte seedé
	rec, cookies2 := get("/connexion", nil)
	csrf = csrfOf(rec)
	form = url.Values{"csrf": {csrf}, "ident": {"admin@test.fr"}, "password": {"admin12345"}}
	rec, cookies2 = post("/connexion", form, cookies2)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("connexion admin -> %d", rec.Code)
	}
	rec, _ = get("/admin", cookies2)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Administration") {
		t.Fatalf("GET /admin -> %d", rec.Code)
	}

	// 7. rate limit : 11 POST /connexion depuis la même IP -> 429
	for i := 0; i < 11; i++ {
		rec, _ = post("/connexion", url.Values{"csrf": {"x"}, "ident": {"x"}, "password": {"x"}}, nil)
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("rate limit attendu 429, got %d", rec.Code)
	}
}

var _ = kernel.LangFR
var _ = app.A
