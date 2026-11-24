// Package evenements : liste des événements, fiche événement et inscription
// en ligne (conditionnée aux statuts de paiement cotiz/licence et au tarif).
package evenements

import (
	"log"
	"net/http"

	"bds/internal/app"
	"bds/internal/kernel"
	"bds/internal/metier/evenements"
	"bds/internal/metier/paiements"
)

func Register(a *app.App) {
	a.Router.Handle("GET", "/evenements", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		var liste []map[string]any
		for _, e := range a.Evenements.Tous() {
			rest := a.Evenements.PlacesRestantes(e)
			liste = append(liste, map[string]any{
				"evt": e, "restantes": rest, "inscrits": a.Evenements.NbInscrits(e.ID),
				"complet": rest == 0, "illimite": e.PlacesMax <= 0,
			})
		}
		a.Render(w, r, "evenements", map[string]any{"evenements": liste}, http.StatusOK)
	})

	a.Router.Handle("GET", "/evenements/{id}", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		e, ok := a.Evenements.ParID(p["id"])
		if !ok {
			http.NotFound(w, r)
			return
		}
		u := a.User(r)
		deja := false
		if u != nil {
			if _, ok := a.Evenements.InscriptionCompte(u.ID, e.ID); ok {
				deja = true
			}
		}
		data := map[string]any{
			"evt":         e,
			"inscrits":    a.Evenements.Inscrits(e.ID),
			"restantes":   a.Evenements.PlacesRestantes(e),
			"deja_inscrit": deja,
			"user":        u,
		}
		a.Render(w, r, "evenement", data, http.StatusOK)
	})

	a.Router.Handle("POST", "/evenements/{id}/inscrire", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		u := a.User(r)
		if u == nil {
			a.Redirect(w, r, "/connexion?next="+r.URL.Path, "")
			return
		}
		c, ok := a.Comptes.ParID(u.ID)
		if !ok {
			a.Redirect(w, r, "/connexion", "")
			return
		}
		e, ok := a.Evenements.ParID(p["id"])
		if !ok {
			http.NotFound(w, r)
			return
		}
		insc, gratuit, err := a.Evenements.Inscrire(&c, e)
		switch err {
		case nil:
		case evenements.ErrDejaInscrit:
			a.Redirect(w, r, "/evenements/"+e.ID, "flash_deja_inscrit")
			return
		case evenements.ErrComplet:
			a.Redirect(w, r, "/evenements/"+e.ID, "flash_complet")
			return
		case evenements.ErrCloture:
			a.Redirect(w, r, "/evenements/"+e.ID, "flash_cloture")
			return
		case evenements.ErrCotizRequise, evenements.ErrLicenceRequise:
			a.Redirect(w, r, "/profil", "flash_paiement_requis")
			return
		default:
			log.Printf("inscription: %v", err)
			a.Redirect(w, r, "/evenements/"+e.ID, "flash_erreur_champs")
			return
		}
		if gratuit != "" {
			// tarif > 0 : paiement HelloAsso requis avant confirmation
			retour := a.Conf.Get("base_url", "") + "/evenements/" + e.ID
			pay, err := a.Paiements.CreerPaiement(c.ID, paiements.ObjetEvenement, insc.ID, e.Tarif, "Inscription — "+e.Nom, retour)
			if err != nil {
				log.Printf("paiement evenement: %v", err)
				a.Redirect(w, r, "/profil", "flash_erreur_champs")
				return
			}
			_ = a.Evenements.AttacherPaiement(insc.ID, pay.ID)
			if pay.CheckoutURL != "" {
				http.Redirect(w, r, pay.CheckoutURL, http.StatusSeeOther)
				return
			}
			a.Redirect(w, r, "/profil", "flash_paiement_requis")
			return
		}
		a.Redirect(w, r, "/evenements/"+e.ID, "flash_inscrit_ok")
	})
}
