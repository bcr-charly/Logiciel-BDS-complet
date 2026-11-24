package kernel

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRouter(t *testing.T) {
	rt := NewRouter()
	rt.Handle("GET", "/evenements/{id}", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("evt=" + p["id"]))
	})
	rt.Handle("GET", "/", func(w http.ResponseWriter, r *http.Request, p Params) {
		w.Write([]byte("home"))
	})

	req := httptest.NewRequest("GET", "/evenements/abc123", nil)
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)
	if rec.Body.String() != "evt=abc123" {
		t.Fatalf("got %q", rec.Body.String())
	}

	req = httptest.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()
	rt.ServeHTTP(rec, req)
	if rec.Body.String() != "home" {
		t.Fatalf("got %q", rec.Body.String())
	}

	req = httptest.NewRequest("POST", "/evenements/abc123", nil)
	rec = httptest.NewRecorder()
	rt.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong method, got %d", rec.Code)
	}
}

func TestStoreCollection(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	type item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	col, err := NewCollection[item](store, "items")
	if err != nil {
		t.Fatal(err)
	}
	if err := col.Append(item{ID: "1", Name: "un"}); err != nil {
		t.Fatal(err)
	}
	if err := col.Append(item{ID: "2", Name: "deux"}); err != nil {
		t.Fatal(err)
	}
	if col.Len() != 2 {
		t.Fatalf("len = %d", col.Len())
	}
	got, ok := col.Find(func(i item) bool { return i.ID == "2" })
	if !ok || got.Name != "deux" {
		t.Fatalf("find failed: %+v %v", got, ok)
	}
	// persistance : recharger depuis le disque
	col2, err := NewCollection[item](store, "items")
	if err != nil {
		t.Fatal(err)
	}
	if col2.Len() != 2 {
		t.Fatalf("reload len = %d", col2.Len())
	}
	// pas de fichier .tmp qui traîne (réécriture atomique)
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("fichier temporaire non renommé: %s", e.Name())
		}
	}
	// update + delete
	if _, ok, err := col.UpdateWhere(func(i item) bool { return i.ID == "1" }, func(i *item) { i.Name = "modifié" }); err != nil || !ok {
		t.Fatal(err)
	}
	got, _ = col.Find(func(i item) bool { return i.ID == "1" })
	if got.Name != "modifié" {
		t.Fatalf("update failed: %+v", got)
	}
	if ok, err := col.DeleteWhere(func(i item) bool { return i.ID == "2" }); err != nil || !ok {
		t.Fatal(err)
	}
	if col.Len() != 1 {
		t.Fatalf("delete failed, len = %d", col.Len())
	}
}

func TestCrypto(t *testing.T) {
	salt, hash, err := CreerHashPassword("motdepasse123")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifierPassword("motdepasse123", salt, hash) {
		t.Fatal("bon mot de passe refusé")
	}
	if VerifierPassword("mauvais", salt, hash) {
		t.Fatal("mauvais mot de passe accepté")
	}
	sig := Signe([]byte("secret"), "message")
	if !VerifieSignature([]byte("secret"), "message", sig) {
		t.Fatal("signature valide refusée")
	}
	if VerifieSignature([]byte("autre"), "message", sig) {
		t.Fatal("signature avec mauvais secret acceptée")
	}
}

func TestSessions(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewStore(dir)
	s, err := NewSessions(store, []byte("secret-de-test-32-caracteres-ok"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	sess, err := s.Creer(rec, "compte1")
	if err != nil {
		t.Fatal(err)
	}
	cookie := rec.Result().Cookies()[0]
	if cookie.Name != SessionCookie || cookie.HttpOnly != true {
		t.Fatalf("cookie inattendu: %+v", cookie)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)
	got, ok := s.Valide(req)
	if !ok || got.CompteID != "compte1" {
		t.Fatalf("session non retrouvée: %+v %v", got, ok)
	}
	// cookie forgé
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(&http.Cookie{Name: SessionCookie, Value: sess.ID + "." + "faux"})
	if _, ok := s.Valide(req2); ok {
		t.Fatal("cookie forgé accepté")
	}
}

func TestI18n(t *testing.T) {
	if T("fr", "nav_sports") != "Sports" {
		t.Fatal("fr nav_sports")
	}
	if T("en", "nav_sports") != "Sports" {
		t.Fatal("en nav_sports")
	}
	if T("fr", "cle_inexistante") != "cle_inexistante" {
		t.Fatal("clé inconnue devrait être renvoyée telle quelle")
	}
	if T("en", "accueil_inscription") != "Sign up" {
		t.Fatal("en accueil_inscription")
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	for i := 0; i < 3; i++ {
		if !rl.Allow("k") {
			t.Fatalf("requête %d refusée", i)
		}
	}
	if rl.Allow("k") {
		t.Fatal("4e requête devrait être refusée")
	}
	if !rl.Allow("autre") {
		t.Fatal("autre clé devrait passer")
	}
}

func TestConf(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "conf.txt")
	os.WriteFile(p, []byte("# commentaire\nport=9090\nsecret = abc \n\n"), 0o600)
	c, err := LoadConf(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Get("port", "") != "9090" {
		t.Fatal("port")
	}
	if c.Get("secret", "") != "abc" {
		t.Fatal("secret trim")
	}
	if c.Get("absent", "def") != "def" {
		t.Fatal("defaut")
	}
}
