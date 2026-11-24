// Package evenements gère les compétitions/tournois (créés par l'admin) et
// les inscriptions des adhérents, conditionnées aux statuts de paiement.
package evenements

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"bds/internal/kernel"
	"bds/internal/metier/comptes"
)

const (
	StatutEnAttente = "EN_ATTENTE"
	StatutConfirmee = "CONFIRMEE"
	StatutAnnulee   = "ANNULEE"
)

var (
	ErrCloture      = errors.New("inscriptions clôturées")
	ErrComplet      = errors.New("événement complet")
	ErrDejaInscrit  = errors.New("déjà inscrit")
	ErrCotizRequise = errors.New("cotisation requise")
	ErrLicenceRequise = errors.New("licence requise")
	ErrIntrouvable  = errors.New("événement introuvable")
)

type Evenement struct {
	ID           string    `json:"id"`
	Nom          string    `json:"nom"`
	Sport        string    `json:"sport"`
	Date         time.Time `json:"date"`
	Lieu         string    `json:"lieu"`
	Description  string    `json:"description,omitempty"`
	Tarif        float64   `json:"tarif"` // 0 = gratuit
	PlacesMax    int       `json:"places_max"` // 0 = illimité
	ExigeCotiz   bool      `json:"exige_cotiz"`
	ExigeLicence bool      `json:"exige_licence"`
	Cloturee     bool      `json:"cloturee"`
	CreatedAt    time.Time `json:"created_at"`
}

type Inscription struct {
	ID          string     `json:"id"`
	EvenementID string     `json:"evenement_id"`
	CompteID    string     `json:"compte_id"`
	Statut      string     `json:"statut"`
	PaiementID  string     `json:"paiement_id,omitempty"`
	QRToken     string     `json:"qr_token,omitempty"`
	PointageLe  *time.Time `json:"pointage_le,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Service struct {
	evts *kernel.Collection[Evenement]
	insc *kernel.Collection[Inscription]
	qr   QRGen
}

// QRGen génère les tokens QR signés (délégué au package qrcode pour éviter
// un import circulaire).
type QRGen interface {
	Generer(inscriptionID, compteID, evenementID string, exp time.Time) string
}

func New(store *kernel.Store, qr QRGen) (*Service, error) {
	evts, err := kernel.NewCollection[Evenement](store, "evenements")
	if err != nil {
		return nil, err
	}
	insc, err := kernel.NewCollection[Inscription](store, "inscriptions")
	if err != nil {
		return nil, err
	}
	return &Service{evts: evts, insc: insc, qr: qr}, nil
}

func (s *Service) Creer(e Evenement) (*Evenement, error) {
	e.ID = kernel.RandomID()
	e.CreatedAt = time.Now()
	if err := s.evts.Append(e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Service) Tous() []Evenement {
	out := s.evts.All()
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out
}

func (s *Service) ParID(id string) (Evenement, bool) {
	return s.evts.Find(func(e Evenement) bool { return e.ID == id })
}

func (s *Service) Supprimer(id string) error {
	s.insc.DeleteWhere(func(i Inscription) bool { return i.EvenementID == id })
	_, err := s.evts.DeleteWhere(func(e Evenement) bool { return e.ID == id })
	return err
}

func (s *Service) SetCloturee(id string, cloturee bool) error {
	_, ok, err := s.evts.UpdateWhere(func(e Evenement) bool { return e.ID == id }, func(e *Evenement) { e.Cloturee = cloturee })
	if err != nil || !ok {
		return ErrIntrouvable
	}
	return nil
}

func (s *Service) NbInscrits(evenementID string) int {
	n := 0
	for _, i := range s.insc.All() {
		if i.EvenementID == evenementID && i.Statut != StatutAnnulee {
			n++
		}
	}
	return n
}

func (s *Service) PlacesRestantes(e Evenement) int {
	if e.PlacesMax <= 0 {
		return math.MaxInt
	}
	return e.PlacesMax - s.NbInscrits(e.ID)
}

func (s *Service) Inscrits(evenementID string) []Inscription {
	var out []Inscription
	for _, i := range s.insc.All() {
		if i.EvenementID == evenementID {
			out = append(out, i)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *Service) InscriptionCompte(compteID, evenementID string) (Inscription, bool) {
	return s.insc.Find(func(i Inscription) bool {
		return i.CompteID == compteID && i.EvenementID == evenementID && i.Statut != StatutAnnulee
	})
}

// Inscrire enregistre l'inscription d'un adhérent. Si un paiement est requis
// (tarif > 0), l'inscription passe EN_ATTENTE et sera confirmée à la validation
// du paiement. Sinon, si les exigences cotiz/licence sont satisfaites, elle est
// directement CONFIRMEE avec son token QR.
func (s *Service) Inscrire(c *comptes.Compte, e Evenement) (*Inscription, string, error) {
	// relecture systématique pour travailler sur un état frais
	frais, ok := s.ParID(e.ID)
	if !ok {
		return nil, "", ErrIntrouvable
	}
	e = frais
	if e.Cloturee {
		return nil, "", ErrCloture
	}
	if s.PlacesRestantes(e) <= 0 {
		return nil, "", ErrComplet
	}
	if _, ok := s.InscriptionCompte(c.ID, e.ID); ok {
		return nil, "", ErrDejaInscrit
	}
	if e.ExigeCotiz && !c.CotizPayee {
		return nil, "", ErrCotizRequise
	}
	if e.ExigeLicence && !c.LicencePayee {
		return nil, "", ErrLicenceRequise
	}
	insc := Inscription{
		ID: kernel.RandomID(), EvenementID: e.ID, CompteID: c.ID,
		Statut: StatutEnAttente, CreatedAt: time.Now(),
	}
	gratuit := e.Tarif <= 0
	if gratuit {
		insc.Statut = StatutConfirmee
		insc.QRToken = s.qr.Generer(insc.ID, c.ID, e.ID, e.Date.Add(24*time.Hour))
	}
	if err := s.insc.Append(insc); err != nil {
		return nil, "", err
	}
	return &insc, map[bool]string{true: "", false: "paiement"}[gratuit], nil
}

// AttacherPaiement lie un paiement à une inscription en attente.
func (s *Service) AttacherPaiement(inscriptionID, paiementID string) error {
	_, ok, err := s.insc.UpdateWhere(func(i Inscription) bool { return i.ID == inscriptionID }, func(i *Inscription) {
		i.PaiementID = paiementID
	})
	if err != nil || !ok {
		return errors.New("inscription introuvable")
	}
	return nil
}

// ConfirmerInscription passe l'inscription à CONFIRMEE et génère le QR
// (appelé à la validation du paiement, ou directement si gratuit).
func (s *Service) ConfirmerInscription(inscriptionID string) error {
	insc, ok := s.insc.Find(func(i Inscription) bool { return i.ID == inscriptionID })
	if !ok {
		return errors.New("inscription introuvable")
	}
	e, ok := s.ParID(insc.EvenementID)
	if !ok {
		return ErrIntrouvable
	}
	_, _, err := s.insc.UpdateWhere(func(i Inscription) bool { return i.ID == inscriptionID }, func(i *Inscription) {
		i.Statut = StatutConfirmee
		if i.QRToken == "" {
			i.QRToken = s.qr.Generer(i.ID, i.CompteID, i.EvenementID, e.Date.Add(24*time.Hour))
		}
	})
	return err
}

func (s *Service) Annuler(inscriptionID, compteID string) error {
	_, ok, err := s.insc.UpdateWhere(func(i Inscription) bool {
		return i.ID == inscriptionID && (compteID == "" || i.CompteID == compteID)
	}, func(i *Inscription) { i.Statut = StatutAnnulee })
	if err != nil || !ok {
		return errors.New("inscription introuvable")
	}
	return nil
}

// Pointer marque le pointage d'un QR (anti-double-pointage).
func (s *Service) Pointer(inscriptionID string) (bool, error) {
	insc, ok := s.insc.Find(func(i Inscription) bool { return i.ID == inscriptionID })
	if !ok {
		return false, errors.New("inscription introuvable")
	}
	if insc.Statut != StatutConfirmee {
		return false, errors.New("inscription non confirmée")
	}
	if insc.PointageLe != nil {
		return false, nil // déjà pointé
	}
	now := time.Now()
	_, _, err := s.insc.UpdateWhere(func(i Inscription) bool { return i.ID == inscriptionID }, func(i *Inscription) {
		i.PointageLe = &now
	})
	return true, err
}

// ExportCSV exporte la liste des inscrits d'un événement.
func (s *Service) ExportCSV(evenementID string, comptesSvc *comptes.Service) string {
	var b strings.Builder
	b.WriteString("inscription_id;compte_id;email;nom;prenom;statut;paiement_id;created_at\n")
	for _, i := range s.Inscrits(evenementID) {
		c, ok := comptesSvc.ParID(i.CompteID)
		email, nom, prenom := "", "", ""
		if ok {
			email, nom, prenom = c.Email, c.Nom, c.Prenom
		}
		b.WriteString(strings.Join([]string{
			i.ID, i.CompteID, email, nom, prenom, i.Statut, i.PaiementID,
			i.CreatedAt.Format(time.RFC3339),
		}, ";") + "\n")
	}
	return b.String()
}
