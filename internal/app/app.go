// Package app compose le noyau (kernel) et les services métier (metier),
// enregistre les routes de base et fournit les helpers de rendu / session.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	root "bds"
	"bds/internal/kernel"
	"bds/internal/metier/annonces"
	"bds/internal/metier/boutique"
	"bds/internal/metier/comptes"
	"bds/internal/metier/equipes"
	"bds/internal/metier/evenements"
	"bds/internal/metier/paiements"
	"bds/internal/metier/qrcode"
)

// Types des contenus éditoriaux (contenu/, éditables sans recompilation).
type Sport struct {
	Slug        string `json:"slug"`
	Nom         string `json:"nom"`
	Description string `json:"description"`
	Responsable string `json:"responsable"`
}

type MembreEquipe struct {
	Role   string `json:"role"`
	Nom    string `json:"nom"`
	Prenom string `json:"prenom"`
	Photo  string `json:"photo"`
}

type Partenaire struct {
	Nom string `json:"nom"`
	URL string `json:"url"`
}

type Accueil struct {
	Titre       string `json:"titre"`
	Accroche    string `json:"accroche"`
	AccrocheEn  string `json:"accroche_en"`
}

// App est l'application complète.
type App struct {
	Conf       *kernel.Conf
	Store      *kernel.Store
	Router     *kernel.Router
	Renderer   *kernel.Renderer
	Sessions   *kernel.Sessions
	Comptes    *comptes.Service
	Annonces   *annonces.Service
	Evenements *evenements.Service
	Equipes    *equipes.Service
	Boutique   *boutique.Service
	Paiements  *paiements.Service
	QR         *qrcode.Service

	authLimiter *kernel.RateLimiter
	StopRecon   chan struct{}
}

// A est l'instance globale (initialisée par Init).
var A *App

type ctxKey struct{}

// Init charge la conf, ouvre le stockage, instancie les services, enregistre
// les routes de base et amorce les données de démo si la base est vide.
func Init(confPath, portOverride, creerAdmin string) error {
	conf, err := kernel.LoadConf(confPath)
	if err != nil {
		return fmt.Errorf("conf: %w", err)
	}
	if portOverride != "" {
		conf = overridePort(conf, portOverride)
	}
	secret := conf.Get("session_secret", "")
	if len(secret) < 16 {
		return errors.New("session_secret obligatoire (16+ caractères) dans conf.txt")
	}
	store, err := kernel.NewStore("data")
	if err != nil {
		return err
	}
	sessions, err := kernel.NewSessions(store, []byte(secret), time.Duration(conf.Int("duree_session_jours", 7))*24*time.Hour)
	if err != nil {
		return err
	}
	renderer, err := kernel.NewRenderer()
	if err != nil {
		return err
	}
	cSvc, err := comptes.New(store)
	if err != nil {
		return err
	}
	aSvc, err := annonces.New(store)
	if err != nil {
		return err
	}
	qrSvc := qrcode.New([]byte(secret))
	eSvc, err := evenements.New(store, qrSvc)
	if err != nil {
		return err
	}
	eqSvc, err := equipes.New(store)
	if err != nil {
		return err
	}
	bSvc, err := boutique.New(store)
	if err != nil {
		return err
	}
	pSvc, err := paiements.New(store, conf, cSvc, eSvc, bSvc)
	if err != nil {
		return err
	}

	A = &App{
		Conf: conf, Store: store, Router: kernel.NewRouter(), Renderer: renderer,
		Sessions: sessions, Comptes: cSvc, Annonces: aSvc, Evenements: eSvc,
		Equipes: eqSvc, Boutique: bSvc, Paiements: pSvc, QR: qrSvc,
		authLimiter: kernel.NewRateLimiter(10, 10*time.Minute),
		StopRecon:   make(chan struct{}),
	}

	A.registerBaseRoutes()

	if err := seed(conf, cSvc, aSvc, eSvc, bSvc); err != nil {
		return err
	}
	if creerAdmin != "" {
		email, pass, ok := strings.Cut(creerAdmin, ":")
		if !ok || email == "" || pass == "" {
			return errors.New("-creer-admin attend email:motdepasse")
		}
		if _, err := cSvc.CreerAdmin(email, pass, "Administrateur BDS"); err != nil {
			return err
		}
		log.Printf("compte admin prêt : %s", email)
	}

	go pSvc.BoucleReconciliation(time.Duration(conf.Int("reconciliation_minutes", 15))*time.Minute, A.StopRecon)
	return nil
}

func overridePort(c *kernel.Conf, port string) *kernel.Conf {
	nc := &kernel.Conf{}
	*nc = *c
	nc.SetValue("port", port)
	return nc
}

func (a *App) registerBaseRoutes() {
	a.Router.Handle("GET", "/static/{path}", a.serveStatic)
	a.Router.Handle("GET", "/healthz", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	})
	a.Router.Handle("GET", "/lang/{lang}", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		lang := p["lang"]
		if lang != kernel.LangFR && lang != kernel.LangEN {
			lang = kernel.LangFR
		}
		http.SetCookie(w, &http.Cookie{Name: kernel.LangCookie, Value: lang, Path: "/", MaxAge: 365 * 24 * 3600, SameSite: http.SameSiteLaxMode})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

// Handler renvoie la chaîne de middlewares complète.
func (a *App) Handler() http.Handler {
	core := kernel.SecurityHeaders(a.Router)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := a.Sessions.Courante(w, r)
		if err != nil {
			http.Error(w, "erreur session", http.StatusInternalServerError)
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, sess))

		// rate limit sur connexion et inscription
		if r.Method == http.MethodPost && (r.URL.Path == "/connexion" || r.URL.Path == "/inscription") {
			if !a.authLimiter.Allow(kernel.ClientIP(r) + "|" + r.URL.Path) {
				http.Error(w, "trop de tentatives, réessaie plus tard", http.StatusTooManyRequests)
				return
			}
		}

		// CSRF sur les POST (webhook HelloAsso exclu : secret d'URL)
		if r.Method == http.MethodPost && !strings.HasPrefix(r.URL.Path, "/webhook/helloasso/") {
			if err := r.ParseForm(); err != nil || r.PostFormValue("csrf") != sess.CSRF {
				http.Error(w, "CSRF invalide", http.StatusForbidden)
				return
			}
		}
		core.ServeHTTP(w, r)
	})
}

// Run démarre le serveur HTTP (HTTP uniquement ; le HTTPS est assuré par le
// reverse proxy d'ARISE).
func (a *App) Run() error {
	addr := a.Conf.Get("host", "127.0.0.1") + ":" + a.Conf.Get("port", "8080")
	log.Printf("BDS ENSIIE à l'écoute sur %s (base_url=%s)", addr, a.Conf.Get("base_url", ""))
	srv := &http.Server{Addr: addr, Handler: a.Handler(), ReadHeaderTimeout: 10 * time.Second}
	return srv.ListenAndServe()
}

// ---- helpers de requête ----

func (a *App) Session(r *http.Request) kernel.Session {
	if v, ok := r.Context().Value(ctxKey{}).(kernel.Session); ok {
		return v
	}
	return kernel.Session{}
}

// User renvoie l'utilisateur connecté (nil si anonyme).
func (a *App) User(r *http.Request) *kernel.User {
	sess := a.Session(r)
	if sess.CompteID == "" {
		return nil
	}
	c, ok := a.Comptes.ParID(sess.CompteID)
	if !ok {
		return nil
	}
	pub := c.Public()
	return &kernel.User{
		ID: pub.ID, Nom: pub.Nom, Prenom: pub.Prenom, Email: pub.Email,
		Roles: pub.Roles,
	}
}

func (a *App) Lang(r *http.Request) string { return kernel.Lang(r) }

func (a *App) T(r *http.Request, key string, args ...any) string {
	return kernel.T(a.Lang(r), key, args...)
}

// View construit la vue template pour la requête.
func (a *App) View(r *http.Request, page string, data any) *kernel.View {
	v := &kernel.View{
		Page: page, Lang: a.Lang(r),
		Flash: r.URL.Query().Get("flash"),
		CSRF:  a.Session(r).CSRF,
		User:  a.User(r),
		Data:  data,
		Extra: map[string]any{},
	}
	v.T = func(key string, args ...any) string { return a.T(r, key, args...) }
	var partenaires []Partenaire
	if err := a.ContenuJSON("partenaires.json", &partenaires); err == nil {
		v.Extra["partenaires"] = partenaires
	}
	return v
}

// Render écrit la page HTML avec le statut donné.
func (a *App) Render(w http.ResponseWriter, r *http.Request, page string, data any, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := a.Renderer.Render(w, page, a.View(r, page, data)); err != nil {
		log.Printf("render %s: %v", page, err)
	}
}

// Redirect redirige avec un éventuel message flash (clé i18n).
func (a *App) Redirect(w http.ResponseWriter, r *http.Request, url, flash string) {
	if flash != "" {
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		url += sep + "flash=" + flash
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// RequireRole renvoie l'utilisateur s'il a un des rôles, sinon redirige vers
// la connexion. Renvoie nil si l'accès est refusé (la réponse est déjà écrite).
func (a *App) RequireRole(w http.ResponseWriter, r *http.Request, roles ...string) *kernel.User {
	u := a.User(r)
	if u == nil {
		a.Redirect(w, r, "/connexion?next="+r.URL.Path, "")
		return nil
	}
	for _, role := range roles {
		if u.HasRole(role) {
			return u
		}
	}
	a.Redirect(w, r, "/", "flash_acces_refuse")
	return nil
}

// ContenuJSON lit un contenu éditorial : disque (contenu/) d'abord, embarqué ensuite.
func (a *App) ContenuJSON(name string, v any) error {
	if b, err := os.ReadFile(path.Join("contenu", name)); err == nil {
		return json.Unmarshal(b, v)
	}
	b, err := root.StaticFS.ReadFile("contenu/" + name)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// serveStatic sert /static/{path} depuis assets/ ou web/ (disque d'abord,
// embarqué ensuite).
func (a *App) serveStatic(w http.ResponseWriter, r *http.Request, p kernel.Params) {
	name := p["path"]
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") {
		http.NotFound(w, r)
		return
	}
	for _, base := range []string{"assets", "web"} {
		fp := path.Join(base, name)
		if fi, err := os.Stat(fp); err == nil && !fi.IsDir() {
			http.ServeFile(w, r, fp)
			return
		}
	}
	for _, base := range []string{"assets", "web"} {
		f, err := root.StaticFS.Open(base + "/" + name)
		if err != nil {
			continue
		}
		st, err := f.Stat()
		if err != nil || st.IsDir() {
			f.Close()
			continue
		}
		if rs, ok := f.(io.ReadSeeker); ok {
			http.ServeContent(w, r, name, st.ModTime(), rs)
			f.Close()
			return
		}
		f.Close()
	}
	http.NotFound(w, r)
}
