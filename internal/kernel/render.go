package kernel

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
	"sync"
	"time"
)

//go:embed templates
var templatesFS embed.FS

// User est la vue publique du compte connecté, exposée aux templates.
type User struct {
	ID     string
	Nom    string
	Prenom string
	Email  string
	Roles  []string
}

func (u *User) HasRole(role string) bool {
	if u == nil {
		return false
	}
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// View est le contexte passé à chaque template.
type View struct {
	Page   string
	Lang   string
	Flash  string // clé i18n du message flash (vide = aucun)
	CSRF   string
	User   *User
	Data   any
	T      func(key string, args ...any) string
	Extra  map[string]any
}

func (v *View) Get(key string) any {
	if v.Extra == nil {
		return nil
	}
	return v.Extra[key]
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"t": func(v *View, key string, args ...any) string {
			if v.T == nil {
				return key
			}
			return v.T(key, args...)
		},
		"dict": func(kv ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i+1 < len(kv); i += 2 {
				if k, ok := kv[i].(string); ok {
					m[k] = kv[i+1]
				}
			}
			return m
		},
		"prix": func(f float64) string {
			return strings.ReplaceAll(fmt.Sprintf("%.2f €", f), ".", ",")
		},
		"date": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("02/01/2006 à 15:04")
		},
		"datecourt": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("02/01/2006")
		},
		"asset": func(p string) string { return "/static/" + p },
		"csrf": func(v *View) template.HTML {
			return template.HTML(`<input type="hidden" name="csrf" value="` + template.HTMLEscapeString(v.CSRF) + `">`)
		},
		"hasrole": func(v *View, role string) bool { return v.User.HasRole(role) },
		"hasanyrole": func(v *View, roles ...string) bool {
			for _, r := range roles {
				if v.User.HasRole(r) {
					return true
				}
			}
			return false
		},
	}
}

// Renderer rendu les pages html/template : layout + composants + une page.
type Renderer struct {
	mu    sync.Mutex
	base  *template.Template
	pages map[string]*template.Template
}

func NewRenderer() (*Renderer, error) {
	base, err := template.New("layout").Funcs(funcMap()).ParseFS(templatesFS,
		"templates/layout.html", "templates/components.html")
	if err != nil {
		return nil, err
	}
	return &Renderer{base: base, pages: map[string]*template.Template{}}, nil
}

// Render exécute la page demandée avec la vue fournie (code 200).
func (r *Renderer) Render(w io.Writer, page string, v *View) error {
	r.mu.Lock()
	tmpl, ok := r.pages[page]
	if !ok {
		var err error
		tmpl, err = r.base.Clone()
		if err != nil {
			r.mu.Unlock()
			return err
		}
		if _, err := tmpl.ParseFS(templatesFS, "templates/pages/"+page+".html"); err != nil {
			r.mu.Unlock()
			return fmt.Errorf("template page %q: %w", page, err)
		}
		r.pages[page] = tmpl
	}
	r.mu.Unlock()
	v.Page = page
	return tmpl.ExecuteTemplate(w, "layout", v)
}
