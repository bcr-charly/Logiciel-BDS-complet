// Package auth : inscription (avec vérification d'e-mail), connexion,
// déconnexion, vérification de compte et profil adhérent.
package auth

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"bds/internal/app"
	"bds/internal/kernel"
	"bds/internal/metier/paiements"
)

func Register(a *app.App) {
	a.Router.Handle("GET", "/inscription", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		a.Render(w, r, "inscription", map[string]any{}, http.StatusOK)
	})

	a.Router.Handle("POST", "/inscription", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		f := r.PostForm
		pass, pass2 := f.Get("password"), f.Get("password2")
		if pass != pass2 {
			a.Redirect(w, r, "/inscription", "flash_erreur_champs")
			return
		}
		c, err := a.Comptes.Creer(f.Get("email"), f.Get("login"), pass, f.Get("nom"), f.Get("prenom"), f.Get("telephone"), f.Get("annee"), f.Get("sport"))
		if err != nil {
			log.Printf("inscription refusée: %v", err)
			a.Redirect(w, r, "/inscription", "flash_erreur_champs")
			return
		}
		// Envoi du lien de vérification : SMTP si configuré, sinon mode démo (lien journalisé)
		lien := a.Conf.Get("base_url", "") + "/verifier-email?token=" + c.VerifToken
		if a.Conf.Has("smtp_host") {
			log.Printf("TODO SMTP: envoyer le lien de vérification à %s : %s", c.Email, lien)
		} else {
			log.Printf("mode démo (SMTP non configuré) — lien de vérification de %s : %s", c.Email, lien)
		}
		a.Redirect(w, r, "/connexion", "flash_compte_cree")
	})

	a.Router.Handle("GET", "/verifier-email", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.Comptes.VerifierEmail(r.URL.Query().Get("token")) {
			a.Redirect(w, r, "/connexion", "flash_compte_cree")
			return
		}
		a.Redirect(w, r, "/connexion", "flash_erreur_champs")
	})

	a.Router.Handle("GET", "/connexion", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		data := map[string]any{"next": r.URL.Query().Get("next")}
		a.Render(w, r, "connexion", data, http.StatusOK)
	})

	a.Router.Handle("POST", "/connexion", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		c, err := a.Comptes.Login(r.PostFormValue("ident"), r.PostFormValue("password"))
		if err != nil {
			a.Redirect(w, r, "/connexion", "flash_mauvais_ident")
			return
		}
		sess := a.Session(r)
		sess.CompteID = c.ID
		if err := a.Sessions.Remplacer(w, sess); err != nil {
			http.Error(w, "erreur session", http.StatusInternalServerError)
			return
		}
		next := r.PostFormValue("next")
		if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
			next = "/profil"
		}
		a.Redirect(w, r, next, "")
	})

	a.Router.Handle("POST", "/deconnexion", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		a.Sessions.Detruire(w, a.Session(r).ID)
		a.Redirect(w, r, "/", "")
	})

	a.Router.Handle("GET", "/profil", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		u := a.RequireRole(w, r, "adherent", "responsable", "admin")
		if u == nil {
			return
		}
		c, _ := a.Comptes.ParID(u.ID)
		pub := c.Public()

		var inscriptions []map[string]any
		// inscriptions de l'utilisateur
		for _, e := range a.Evenements.Tous() {
			if insc, ok := a.Evenements.InscriptionCompte(c.ID, e.ID); ok {
				inscriptions = append(inscriptions, map[string]any{
					"id": insc.ID, "statut": insc.Statut, "qr": insc.QRToken,
					"evenement": e.Nom, "date": e.Date,
				})
			}
		}
		data := map[string]any{
			"compte":       pub,
			"inscriptions": inscriptions,
			"commandes":    a.Boutique.ParCompte(c.ID),
			"paiements":    a.Paiements.ParCompte(c.ID),
			"cotiz_prix":   a.Conf.Get("cotiz_montant", "15"),
			"licence_prix": a.Conf.Get("licence_montant", "10"),
		}
		a.Render(w, r, "profil", data, http.StatusOK)
	})

	// Paiement de la cotisation ou de la licence via HelloAsso.
	a.Router.Handle("POST", "/profil/payer", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		u := a.RequireRole(w, r, "adherent", "responsable", "admin")
		if u == nil {
			return
		}
		objet := r.PostFormValue("objet")
		if objet != paiements.ObjetCotiz && objet != paiements.ObjetLicence {
			a.Redirect(w, r, "/profil", "flash_erreur_champs")
			return
		}
		montant := 15.0
		item := "Cotisation BDS"
		if objet == paiements.ObjetLicence {
			montant = 10.0
			item = "Licence universitaire BDS"
		}
		if v := a.Conf.Get("cotiz_montant", ""); objet == paiements.ObjetCotiz && v != "" {
			montant = parseFloat(v, montant)
		}
		if v := a.Conf.Get("licence_montant", ""); objet == paiements.ObjetLicence && v != "" {
			montant = parseFloat(v, montant)
		}
		retour := a.Conf.Get("base_url", "") + "/profil"
		pay, err := a.Paiements.CreerPaiement(u.ID, objet, "", montant, item, retour)
		if err != nil {
			log.Printf("paiement: %v", err)
			a.Redirect(w, r, "/profil", "flash_erreur_champs")
			return
		}
		if pay.CheckoutURL != "" {
			http.Redirect(w, r, pay.CheckoutURL, http.StatusSeeOther)
			return
		}
		a.Redirect(w, r, "/profil", "flash_commande_ok")
	})
}

func parseFloat(s string, def float64) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return def
}
