// Package comptes gère les comptes (adhérents, responsables, admin),
// la vérification d'e-mail et les statuts de paiement cotiz/licence.
package comptes

import (
	"errors"
	"strings"
	"time"

	"bds/internal/kernel"
)

const (
	RoleAdherent    = "adherent"
	RoleResponsable = "responsable"
	RoleAdmin       = "admin"
)

var (
	ErrEmailPris   = errors.New("email déjà utilisé")
	ErrLoginPris   = errors.New("identifiant déjà utilisé")
	ErrIdentInvalide = errors.New("identifiant invalide")
	ErrMauvaisIdent = errors.New("identifiant ou mot de passe incorrect")
	ErrEmailInvalide = errors.New("e-mail invalide")
)

type Compte struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Login        string    `json:"login"`
	PassSalt     string    `json:"pass_salt"`
	PassHash     string    `json:"pass_hash"`
	Nom          string    `json:"nom"`
	Prenom       string    `json:"prenom"`
	Telephone    string    `json:"telephone,omitempty"`
	Annee        string    `json:"annee,omitempty"`
	SportPrefere string    `json:"sport_prefere,omitempty"`
	Roles        []string  `json:"roles"`
	Sports       []string  `json:"sports,omitempty"` // sports gérés (responsable)
	CotizPayee   bool      `json:"cotiz_payee"`
	LicencePayee bool      `json:"licence_payee"`
	EmailVerifie bool      `json:"email_verifie"`
	VerifToken   string    `json:"verif_token,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func (c *Compte) AUnRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// EstResponsableDe indique si le compte gère le sport (admin ou responsable du sport).
func (c *Compte) EstResponsableDe(sport string) bool {
	if c.AUnRole(RoleAdmin) {
		return true
	}
	if !c.AUnRole(RoleResponsable) {
		return false
	}
	for _, s := range c.Sports {
		if s == sport {
			return true
		}
	}
	return false
}

// Public est la vue sans secret du compte.
type Public struct {
	ID           string   `json:"id"`
	Email        string   `json:"email"`
	Login        string   `json:"login"`
	Nom          string   `json:"nom"`
	Prenom       string   `json:"prenom"`
	Roles        []string `json:"roles"`
	Sports       []string `json:"sports,omitempty"`
	CotizPayee   bool     `json:"cotiz_payee"`
	LicencePayee bool     `json:"licence_payee"`
	EmailVerifie bool     `json:"email_verifie"`
}

func (c *Compte) Public() Public {
	return Public{
		ID: c.ID, Email: c.Email, Login: c.Login, Nom: c.Nom, Prenom: c.Prenom,
		Roles: c.Roles, Sports: c.Sports, CotizPayee: c.CotizPayee,
		LicencePayee: c.LicencePayee, EmailVerifie: c.EmailVerifie,
	}
}

type Service struct {
	col *kernel.Collection[Comptes]
}

// Collection wrapper pour garder la généricité interne.
type Comptes = Compte

func New(store *kernel.Store) (*Service, error) {
	col, err := kernel.NewCollection[Comptes](store, "comptes")
	if err != nil {
		return nil, err
	}
	return &Service{col: col}, nil
}

func emailValide(email string) bool {
	at := strings.LastIndex(email, "@")
	return at > 0 && at < len(email)-1 && strings.Contains(email[at:], ".")
}

func loginValide(login string) bool {
	if len(login) < 3 || len(login) > 32 {
		return false
	}
	for _, r := range login {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.'
		if !ok {
			return false
		}
	}
	return true
}

// Creer enregistre un nouveau compte adhérent (e-mail à vérifier).
func (s *Service) Creer(email, login, password, nom, prenom, telephone, annee, sport string) (*Compte, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	login = strings.ToLower(strings.TrimSpace(login))
	if !emailValide(email) {
		return nil, ErrEmailInvalide
	}
	if !loginValide(login) {
		return nil, ErrIdentInvalide
	}
	if len(password) < 8 {
		return nil, errors.New("mot de passe trop court (8 caractères min)")
	}
	if _, ok := s.col.Find(func(c Compte) bool { return c.Email == email }); ok {
		return nil, ErrEmailPris
	}
	if _, ok := s.col.Find(func(c Compte) bool { return c.Login == login }); ok {
		return nil, ErrLoginPris
	}
	salt, hash, err := kernel.CreerHashPassword(password)
	if err != nil {
		return nil, err
	}
	c := Compte{
		ID: kernel.RandomID(), Email: email, Login: login,
		PassSalt: salt, PassHash: hash,
		Nom: strings.TrimSpace(nom), Prenom: strings.TrimSpace(prenom),
		Telephone: strings.TrimSpace(telephone), Annee: strings.TrimSpace(annee),
		SportPrefere: strings.TrimSpace(sport),
		Roles: []string{RoleAdherent}, VerifToken: kernel.RandomToken(),
		CreatedAt: time.Now(),
	}
	if err := s.col.Append(c); err != nil {
		return nil, err
	}
	return &c, nil
}

// CreerAdmin crée ou met à jour un compte admin (démarrage / flag -creer-admin).
func (s *Service) CreerAdmin(email, password, nom string) (*Compte, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if existing, ok := s.col.Find(func(c Compte) bool { return c.Email == email }); ok {
		salt, hash, err := kernel.CreerHashPassword(password)
		if err != nil {
			return nil, err
		}
		updated, _, err := s.col.UpdateWhere(func(c Compte) bool { return c.ID == existing.ID }, func(c *Compte) {
			c.PassSalt, c.PassHash = salt, hash
			if !c.AUnRole(RoleAdmin) {
				c.Roles = append(c.Roles, RoleAdmin)
			}
			if !c.AUnRole(RoleAdherent) {
				c.Roles = append(c.Roles, RoleAdherent)
			}
			c.EmailVerifie = true
		})
		return &updated, err
	}
	c, err := s.Creer(email, strings.Split(email, "@")[0], password, nom, "", "", "", "")
	if err != nil {
		return nil, err
	}
	updated, _, err := s.col.UpdateWhere(func(x Compte) bool { return x.ID == c.ID }, func(x *Compte) {
		x.Roles = []string{RoleAdherent, RoleAdmin}
		x.EmailVerifie = true
		x.VerifToken = ""
	})
	return &updated, err
}

// Login authentifie par e-mail ou identifiant + mot de passe.
func (s *Service) Login(ident, password string) (*Compte, error) {
	ident = strings.ToLower(strings.TrimSpace(ident))
	c, ok := s.col.Find(func(c Compte) bool { return c.Email == ident || c.Login == ident })
	if !ok || !kernel.VerifierPassword(password, c.PassSalt, c.PassHash) {
		return nil, ErrMauvaisIdent
	}
	return &c, nil
}

func (s *Service) ParID(id string) (Compte, bool) {
	return s.col.Find(func(c Compte) bool { return c.ID == id })
}

func (s *Service) Tous() []Compte { return s.col.All() }

// Maj remplace un compte (par ID).
func (s *Service) Maj(c Compte) error {
	_, ok, err := s.col.UpdateWhere(func(x Compte) bool { return x.ID == c.ID }, func(x *Compte) { *x = c })
	if err != nil || !ok {
		return errors.New("compte introuvable")
	}
	return nil
}

// VerifierEmail active le compte via le token envoyé par e-mail.
func (s *Service) VerifierEmail(token string) bool {
	_, ok, _ := s.col.UpdateWhere(func(c Compte) bool { return c.VerifToken != "" && c.VerifToken == token }, func(c *Compte) {
		c.EmailVerifie = true
		c.VerifToken = ""
	})
	return ok
}

// SetRole ajoute/retire un rôle.
func (s *Service) SetRole(id, role string, actif bool) error {
	_, ok, err := s.col.UpdateWhere(func(c Compte) bool { return c.ID == id }, func(c *Compte) {
		has := c.AUnRole(role)
		if actif && !has {
			c.Roles = append(c.Roles, role)
		}
		if !actif && has {
			var out []string
			for _, r := range c.Roles {
				if r != role {
					out = append(out, r)
				}
			}
			c.Roles = out
		}
	})
	if err != nil || !ok {
		return errors.New("compte introuvable")
	}
	return nil
}

// SetSportsResponsable définit les sports gérés par un responsable.
func (s *Service) SetSportsResponsable(id string, sports []string) error {
	_, ok, err := s.col.UpdateWhere(func(c Compte) bool { return c.ID == id }, func(c *Compte) { c.Sports = sports })
	if err != nil || !ok {
		return errors.New("compte introuvable")
	}
	return nil
}

// SetPaiement bascule cotiz/licence payée.
func (s *Service) SetPaiement(id, quoi string, paye bool) error {
	_, ok, err := s.col.UpdateWhere(func(c Compte) bool { return c.ID == id }, func(c *Compte) {
		switch quoi {
		case "cotiz":
			c.CotizPayee = paye
		case "licence":
			c.LicencePayee = paye
		}
	})
	if err != nil || !ok {
		return errors.New("compte introuvable")
	}
	return nil
}

// ExportCSV exporte les comptes (admin).
func (s *Service) ExportCSV() string {
	var b strings.Builder
	b.WriteString("id;email;login;nom;prenom;roles;sports;cotiz;licence;email_verifie;created_at\n")
	for _, c := range s.col.All() {
		b.WriteString(strings.Join([]string{
			c.ID, c.Email, c.Login, c.Nom, c.Prenom,
			strings.Join(c.Roles, "+"), strings.Join(c.Sports, "+"),
			boolStr(c.CotizPayee), boolStr(c.LicencePayee), boolStr(c.EmailVerifie),
			c.CreatedAt.Format(time.RFC3339),
		}, ";") + "\n")
	}
	return b.String()
}

func boolStr(b bool) string {
	if b {
		return "oui"
	}
	return "non"
}
