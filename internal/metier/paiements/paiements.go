// Package paiements gère les paiements HelloAsso : création de checkout,
// webhook (secret d'URL + relecture API), réconciliation périodique et
// application des effets (cotiz/licence à jour, commande payée, QR généré).
// Aucune donnée bancaire n'est stockée localement.
package paiements

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"bds/internal/kernel"
	"bds/internal/metier/boutique"
	"bds/internal/metier/comptes"
	"bds/internal/metier/evenements"
)

const (
	PayEnAttente = "EN_ATTENTE"
	PayPaye      = "PAYE"
	PayAbandonne = "ABANDONNE"

	ObjetCotiz     = "cotiz"
	ObjetLicence   = "licence"
	ObjetCommande  = "commande"
	ObjetEvenement = "evenement"
)

var ErrDejaTraite = errors.New("paiement déjà traité")

type Paiement struct {
	ID          string     `json:"id"`
	CompteID    string     `json:"compte_id,omitempty"`
	Objet       string     `json:"objet"` // cotiz | licence | commande | evenement
	ObjetID     string     `json:"objet_id,omitempty"`
	Montant     float64    `json:"montant"`
	Statut      string     `json:"statut"`
	CheckoutURL string     `json:"checkout_url,omitempty"`
	HelloAssoID int        `json:"helloasso_id,omitempty"`
	TraiteLe    *time.Time `json:"traite_le,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Service struct {
	col       *kernel.Collection[Paiement]
	conf      *kernel.Conf
	ha        *HelloAsso
	comptes   *comptes.Service
	evenements *evenements.Service
	boutique  *boutique.Service
}

func New(store *kernel.Store, conf *kernel.Conf, c *comptes.Service, e *evenements.Service, b *boutique.Service) (*Service, error) {
	col, err := kernel.NewCollection[Paiement](store, "paiements")
	if err != nil {
		return nil, err
	}
	ha := NewHelloAsso(conf.Get("helloasso_api", "https://api.helloasso.com"), conf.Get("helloasso_slug", ""), conf.Get("helloasso_token", ""))
	return &Service{col: col, conf: conf, ha: ha, comptes: c, evenements: e, boutique: b}, nil
}

// Configure indique si HelloAsso est configuré (sinon mode démo : paiements
// validés manuellement par l'admin).
func (s *Service) Configure() bool { return s.ha.configure() }

// CreerPaiement enregistre un paiement EN_ATTENTE et, si HelloAsso est
// configuré, crée le checkout-intent associé.
func (s *Service) CreerPaiement(compteID, objet, objetID string, montant float64, itemName, returnURL string) (*Paiement, error) {
	p := Paiement{
		ID: kernel.RandomID(), CompteID: compteID, Objet: objet, ObjetID: objetID,
		Montant: montant, Statut: PayEnAttente, CreatedAt: time.Now(),
	}
	if s.ha.configure() {
		url, err := s.ha.CreerCheckoutIntent(itemName, montant, p.ID, returnURL, map[string]string{
			"compte_id": compteID, "objet": objet, "objet_id": objetID,
		})
		if err != nil {
			return nil, err
		}
		p.CheckoutURL = url
	}
	if err := s.col.Append(p); err != nil {
		return nil, err
	}
	return &p, nil
}

// MarquerPaye valide un paiement (idempotent) et déclenche l'effet associé.
func (s *Service) MarquerPaye(id string) error {
	p, ok := s.col.Find(func(p Paiement) bool { return p.ID == id })
	if !ok {
		return errors.New("paiement introuvable")
	}
	if p.TraiteLe != nil {
		return ErrDejaTraite // idempotence : jamais d'effet deux fois
	}
	now := time.Now()
	_, _, err := s.col.UpdateWhere(func(x Paiement) bool { return x.ID == id }, func(x *Paiement) {
		x.Statut = PayPaye
		x.TraiteLe = &now
	})
	if err != nil {
		return err
	}
	return s.appliqueEffet(&p)
}

func (s *Service) appliqueEffet(p *Paiement) error {
	switch p.Objet {
	case ObjetCotiz:
		return s.comptes.SetPaiement(p.CompteID, "cotiz", true)
	case ObjetLicence:
		return s.comptes.SetPaiement(p.CompteID, "licence", true)
	case ObjetCommande:
		if err := s.boutique.SetStatut(p.ObjetID, boutique.CmdPayee); err != nil {
			return err
		}
		return nil
	case ObjetEvenement:
		return s.evenements.ConfirmerInscription(p.ObjetID)
	}
	return nil
}

// TraiterWebhook traite une notification HelloAsso : vérification du secret
// d'URL (passé dans le chemin), puis relecture du paiement côté API pour
// confirmation (les notifications des comptes association ne sont pas signées).
func (s *Service) TraiterWebhook(secret string, body []byte) error {
	if secret == "" || secret != s.conf.Get("webhook_secret", "") {
		return errors.New("secret de webhook invalide")
	}
	var notif WebhookNotification
	if err := json.Unmarshal(body, &notif); err != nil {
		return err
	}
	if notif.Data.ID == 0 {
		return errors.New("notification sans id de paiement")
	}
	if !s.ha.configure() {
		return nil // mode démo : rien à relire
	}
	status, err := s.ha.GetPayment(notif.Data.ID)
	if err != nil {
		return err
	}
	if status != "Authorized" && status != "Completed" {
		return nil
	}
	p, ok := s.col.Find(func(p Paiement) bool { return p.HelloAssoID == notif.Data.ID })
	if !ok {
		return errors.New("paiement inconnu localement")
	}
	return s.MarquerPaye(p.ID)
}

// Reconciliation marque ABANDONNE les paiements sans notification après
// 45 minutes (relecture API d'abord si HelloAsso est configuré).
func (s *Service) Reconciliation() {
	seuil := time.Now().Add(-45 * time.Minute)
	for _, p := range s.col.All() {
		if p.Statut != PayEnAttente || p.CreatedAt.After(seuil) {
			continue
		}
		if s.ha.configure() && p.HelloAssoID != 0 {
			if status, err := s.ha.GetPayment(p.HelloAssoID); err == nil && (status == "Authorized" || status == "Completed") {
				if err := s.MarquerPaye(p.ID); err != nil {
					log.Printf("reconciliation: %v", err)
				}
				continue
			}
		}
		if !s.ha.configure() {
			continue // mode démo : validation manuelle par l'admin
		}
		s.col.UpdateWhere(func(x Paiement) bool { return x.ID == p.ID }, func(x *Paiement) { x.Statut = PayAbandonne })
	}
}

// BoucleReconciliation lance la tâche de fond périodique.
func (s *Service) BoucleReconciliation(interval time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			s.Reconciliation()
		}
	}
}

func (s *Service) Tous() []Paiement { return s.col.All() }

func (s *Service) ParID(id string) (Paiement, bool) {
	return s.col.Find(func(p Paiement) bool { return p.ID == id })
}

func (s *Service) ParCompte(compteID string) []Paiement {
	var out []Paiement
	for _, p := range s.col.All() {
		if p.CompteID == compteID {
			out = append(out, p)
		}
	}
	return out
}

// ParseHelloAssoID extrait l'id numérique d'un paiement depuis une URL ou un texte.
func ParseHelloAssoID(s string) (int, error) { return strconv.Atoi(s) }
