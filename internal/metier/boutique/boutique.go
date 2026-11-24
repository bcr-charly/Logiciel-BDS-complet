// Package boutique gère les articles (édités par l'admin) et les commandes.
// Le paiement passe par HelloAsso (aucune donnée bancaire stockée localement).
package boutique

import (
	"errors"
	"time"

	"bds/internal/kernel"
)

const (
	CmdEnAttente = "EN_ATTENTE"
	CmdPayee     = "PAYEE"
	CmdAnnulee   = "ANNULEE"
)

var (
	ErrStockInsuffisant = errors.New("stock insuffisant")
	ErrArticleIntrouvable = errors.New("article introuvable ou inactif")
)

type Article struct {
	ID          string    `json:"id"`
	Nom         string    `json:"nom"`
	Description string    `json:"description,omitempty"`
	Prix        float64   `json:"prix"`
	Stock       int       `json:"stock"`
	Actif       bool      `json:"actif"`
	Image       string    `json:"image,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Ligne struct {
	ArticleID    string  `json:"article_id"`
	Nom          string  `json:"nom"`
	Quantite     int     `json:"quantite"`
	PrixUnitaire float64 `json:"prix_unitaire"`
}

type Commande struct {
	ID         string    `json:"id"`
	CompteID   string    `json:"compte_id,omitempty"`
	Email      string    `json:"email"`
	Statut     string    `json:"statut"`
	Total      float64   `json:"total"`
	Lignes     []Ligne   `json:"lignes"`
	PaiementID string    `json:"paiement_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Service struct {
	articles *kernel.Collection[Article]
	commandes *kernel.Collection[Commande]
}

func New(store *kernel.Store) (*Service, error) {
	articles, err := kernel.NewCollection[Article](store, "articles")
	if err != nil {
		return nil, err
	}
	commandes, err := kernel.NewCollection[Commande](store, "commandes")
	if err != nil {
		return nil, err
	}
	return &Service{articles: articles, commandes: commandes}, nil
}

func (s *Service) CreerArticle(a Article) (*Article, error) {
	a.ID = kernel.RandomID()
	a.Actif = true
	a.CreatedAt = time.Now()
	if err := s.articles.Append(a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) ArticlesActifs() []Article {
	var out []Article
	for _, a := range s.articles.All() {
		if a.Actif {
			out = append(out, a)
		}
	}
	return out
}

func (s *Service) TousArticles() []Article { return s.articles.All() }

func (s *Service) ArticleParID(id string) (Article, bool) {
	return s.articles.Find(func(a Article) bool { return a.ID == id })
}

func (s *Service) SupprimerArticle(id string) error {
	_, err := s.articles.DeleteWhere(func(a Article) bool { return a.ID == id })
	return err
}

// Commander crée une commande (décrément de stock immédiat) et renvoie le
// total à payer via HelloAsso.
func (s *Service) Commander(compteID, email, articleID string, quantite int) (*Commande, error) {
	if quantite < 1 {
		quantite = 1
	}
	a, ok := s.ArticleParID(articleID)
	if !ok || !a.Actif {
		return nil, ErrArticleIntrouvable
	}
	if a.Stock < quantite {
		return nil, ErrStockInsuffisant
	}
	if _, _, err := s.articles.UpdateWhere(func(x Article) bool { return x.ID == articleID }, func(x *Article) {
		x.Stock -= quantite
	}); err != nil {
		return nil, err
	}
	cmd := Commande{
		ID: kernel.RandomID(), CompteID: compteID, Email: email,
		Statut: CmdEnAttente, CreatedAt: time.Now(),
		Lignes: []Ligne{{ArticleID: a.ID, Nom: a.Nom, Quantite: quantite, PrixUnitaire: a.Prix}},
	}
	cmd.Total = a.Prix * float64(quantite)
	if err := s.commandes.Append(cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}

func (s *Service) Commandes() []Commande { return s.commandes.All() }

func (s *Service) CommandeParID(id string) (Commande, bool) {
	return s.commandes.Find(func(c Commande) bool { return c.ID == id })
}

func (s *Service) ParCompte(compteID string) []Commande {
	var out []Commande
	for _, c := range s.commandes.All() {
		if c.CompteID == compteID {
			out = append(out, c)
		}
	}
	return out
}

// SetStatut met à jour le statut (ex. PAYEE après paiement HelloAsso).
func (s *Service) SetStatut(id, statut string) error {
	_, ok, err := s.commandes.UpdateWhere(func(c Commande) bool { return c.ID == id }, func(c *Commande) { c.Statut = statut })
	if err != nil || !ok {
		return errors.New("commande introuvable")
	}
	return nil
}

func (s *Service) AttacherPaiement(commandeID, paiementID string) error {
	_, ok, err := s.commandes.UpdateWhere(func(c Commande) bool { return c.ID == commandeID }, func(c *Commande) { c.PaiementID = paiementID })
	if err != nil || !ok {
		return errors.New("commande introuvable")
	}
	return nil
}
