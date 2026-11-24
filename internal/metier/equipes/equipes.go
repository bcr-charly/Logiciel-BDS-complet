// Package equipes gère la composition des équipes par sport (responsable du
// sport ou admin), à partir des adhérents à jour de paiement.
package equipes

import (
	"sort"
	"strings"
	"time"

	"bds/internal/kernel"
	"bds/internal/metier/comptes"
)

type Equipe struct {
	ID         string    `json:"id"`
	Sport      string    `json:"sport"`
	Nom        string    `json:"nom"`
	Adversaire string    `json:"adversaire,omitempty"`
	Date       time.Time `json:"date"`
	Lieu       string    `json:"lieu,omitempty"`
	Membres    []string  `json:"membres"` // IDs de comptes
	CreatedAt  time.Time `json:"created_at"`
}

type Service struct {
	col *kernel.Collection[Equipe]
}

func New(store *kernel.Store) (*Service, error) {
	col, err := kernel.NewCollection[Equipe](store, "equipes")
	if err != nil {
		return nil, err
	}
	return &Service{col: col}, nil
}

func (s *Service) Creer(e Equipe) (*Equipe, error) {
	e.ID = kernel.RandomID()
	e.CreatedAt = time.Now()
	if err := s.col.Append(e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Service) ParSport(sport string) []Equipe {
	var out []Equipe
	for _, e := range s.col.All() {
		if e.Sport == sport {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out
}

func (s *Service) Supprimer(id string) error {
	_, err := s.col.DeleteWhere(func(e Equipe) bool { return e.ID == id })
	return err
}

// ExportCSV exporte les feuilles d'équipe d'un sport avec les noms des membres.
func (s *Service) ExportCSV(sport string, comptesSvc *comptes.Service) string {
	var b strings.Builder
	b.WriteString("equipe;date;adversaire;lieu;membres\n")
	for _, e := range s.ParSport(sport) {
		var noms []string
		for _, id := range e.Membres {
			if c, ok := comptesSvc.ParID(id); ok {
				noms = append(noms, c.Prenom+" "+c.Nom+" <"+c.Email+">")
			}
		}
		b.WriteString(strings.Join([]string{
			e.Nom, e.Date.Format("2006-01-02 15:04"), e.Adversaire, e.Lieu, strings.Join(noms, ", "),
		}, ";") + "\n")
	}
	return b.String()
}
