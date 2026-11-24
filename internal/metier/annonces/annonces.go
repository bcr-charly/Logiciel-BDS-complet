// Package annonces gère les actus : types ouverture d'inscriptions,
// résultats de matchs, informations générales. Globales (admin) ou par sport
// (responsable).
package annonces

import (
	"sort"
	"time"

	"bds/internal/kernel"
)

const (
	TypeOuverture = "ouverture"
	TypeResultats = "resultats"
	TypeInfo      = "info"
)

type Annonce struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // ouverture | resultats | info
	Titre     string    `json:"titre"`
	Contenu   string    `json:"contenu"`
	Sport     string    `json:"sport,omitempty"` // vide = annonce globale
	AuteurID  string    `json:"auteur_id"`
	Publiee   bool      `json:"publiee"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	col *kernel.Collection[Annonce]
}

func New(store *kernel.Store) (*Service, error) {
	col, err := kernel.NewCollection[Annonce](store, "annonces")
	if err != nil {
		return nil, err
	}
	return &Service{col: col}, nil
}

func (s *Service) Creer(typ, titre, contenu, sport, auteurID string) (*Annonce, error) {
	a := Annonce{
		ID: kernel.RandomID(), Type: typ, Titre: titre, Contenu: contenu,
		Sport: sport, AuteurID: auteurID, Publiee: true, CreatedAt: time.Now(),
	}
	if err := s.col.Append(a); err != nil {
		return nil, err
	}
	return &a, nil
}

func triDesc(items []Annonce) {
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
}

// Recentes renvoie les annonces publiées, plus récentes d'abord.
func (s *Service) Recentes(limit int) []Annonce {
	var out []Annonce
	for _, a := range s.col.All() {
		if a.Publiee {
			out = append(out, a)
		}
	}
	triDesc(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *Service) Toutes() []Annonce {
	out := s.col.All()
	triDesc(out)
	return out
}

// PourSport renvoie les annonces publiées d'un sport + les globales.
func (s *Service) PourSport(sport string) []Annonce {
	var out []Annonce
	for _, a := range s.col.All() {
		if a.Publiee && (a.Sport == "" || a.Sport == sport) {
			out = append(out, a)
		}
	}
	triDesc(out)
	return out
}

func (s *Service) Supprimer(id string) error {
	_, err := s.col.DeleteWhere(func(a Annonce) bool { return a.ID == id })
	return err
}

func (s *Service) Basculer(id string) error {
	_, ok, err := s.col.UpdateWhere(func(a Annonce) bool { return a.ID == id }, func(a *Annonce) { a.Publiee = !a.Publiee })
	if err != nil || !ok {
		return err
	}
	return nil
}
