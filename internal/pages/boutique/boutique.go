// Package boutique : catalogue et commande (paiement HelloAsso, aucune
// donnée bancaire stockée localement).
package boutique

import (
	"log"
	"net/http"
	"strconv"

	"bds/internal/app"
	"bds/internal/kernel"
	"bds/internal/metier/boutique"
	"bds/internal/metier/paiements"
)

func Register(a *app.App) {
	a.Router.Handle("GET", "/boutique", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		data := map[string]any{"articles": a.Boutique.ArticlesActifs()}
		a.Render(w, r, "boutique", data, http.StatusOK)
	})

	a.Router.Handle("POST", "/boutique/commander", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		u := a.User(r)
		email := r.PostFormValue("email")
		compteID := ""
		if u != nil {
			email = u.Email
			compteID = u.ID
		}
		if email == "" {
			a.Redirect(w, r, "/boutique", "flash_erreur_champs")
			return
		}
		quantite := 1
		if q, err := strconv.Atoi(r.PostFormValue("quantite")); err == nil && q > 0 {
			quantite = q
		}
		cmd, err := a.Boutique.Commander(compteID, email, r.PostFormValue("article_id"), quantite)
		switch err {
		case nil:
		case boutique.ErrStockInsuffisant:
			a.Redirect(w, r, "/boutique", "flash_erreur_champs")
			return
		default:
			log.Printf("commande: %v", err)
			a.Redirect(w, r, "/boutique", "flash_erreur_champs")
			return
		}
		retour := a.Conf.Get("base_url", "") + "/profil"
		pay, err := a.Paiements.CreerPaiement(compteID, paiements.ObjetCommande, cmd.ID, cmd.Total, "Commande boutique BDS", retour)
		if err != nil {
			log.Printf("paiement commande: %v", err)
			a.Redirect(w, r, "/boutique", "flash_erreur_champs")
			return
		}
		_ = a.Boutique.AttacherPaiement(cmd.ID, pay.ID)
		if pay.CheckoutURL != "" {
			http.Redirect(w, r, pay.CheckoutURL, http.StatusSeeOther)
			return
		}
		a.Redirect(w, r, "/boutique", "flash_commande_ok")
	})
}
