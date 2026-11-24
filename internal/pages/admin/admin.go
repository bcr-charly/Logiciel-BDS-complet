// Package admin : espace d'administration (rôle admin) — statistiques, CRUD
// annonces / événements / articles, gestion des rôles et des statuts de
// paiement, exports CSV, validation manuelle des paiements (mode démo) et
// purge annuelle.
package admin

import (
	"net/http"
	"strconv"
	"time"

	"bds/internal/app"
	"bds/internal/kernel"
	"bds/internal/metier/annonces"
	"bds/internal/metier/boutique"
	"bds/internal/metier/comptes"
	"bds/internal/metier/evenements"
)

func Register(a *app.App) {
	a.Router.Handle("GET", "/admin", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		var insc []map[string]any
		for _, e := range a.Evenements.Tous() {
			for _, i := range a.Evenements.Inscrits(e.ID) {
				c, _ := a.Comptes.ParID(i.CompteID)
				email := ""
				if ok := c.ID != ""; ok {
					email = c.Email
				}
				insc = append(insc, map[string]any{
					"id": i.ID, "statut": i.Statut, "email": email,
					"evenement": e.Nom, "date": i.CreatedAt,
				})
			}
		}
		data := map[string]any{
			"annonces":    a.Annonces.Toutes(),
			"evenements":  a.Evenements.Tous(),
			"articles":    a.Boutique.TousArticles(),
			"comptes":     a.Comptes.Tous(),
			"paiements":   a.Paiements.Tous(),
			"commandes":   a.Boutique.Commandes(),
			"inscriptions": insc,
			"stats": map[string]int{
				"comptes":     len(a.Comptes.Tous()),
				"evenements":  len(a.Evenements.Tous()),
				"inscriptions": len(insc),
				"commandes":   len(a.Boutique.Commandes()),
				"articles":    len(a.Boutique.TousArticles()),
				"paiements":   len(a.Paiements.Tous()),
			},
			"helloasso": a.Paiements.Configure(),
		}
		a.Render(w, r, "admin", data, http.StatusOK)
	})

	// ---- annonces ----
	a.Router.Handle("POST", "/admin/annonces", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		typ := r.PostFormValue("type")
		if typ != annonces.TypeOuverture && typ != annonces.TypeResultats && typ != annonces.TypeInfo {
			typ = annonces.TypeInfo
		}
		if _, err := a.Annonces.Creer(typ, r.PostFormValue("titre"), r.PostFormValue("contenu"), r.PostFormValue("sport"), a.User(r).ID); err != nil {
			a.Redirect(w, r, "/admin", "flash_erreur_champs")
			return
		}
		a.Redirect(w, r, "/admin", "flash_annonce_creee")
	})
	a.Router.Handle("POST", "/admin/annonces/{id}/supprimer", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		a.Annonces.Supprimer(p["id"])
		a.Redirect(w, r, "/admin", "flash_supprime")
	})
	a.Router.Handle("POST", "/admin/annonces/{id}/basculer", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		a.Annonces.Basculer(p["id"])
		a.Redirect(w, r, "/admin", "")
	})

	// ---- événements ----
	a.Router.Handle("POST", "/admin/evenements", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		date, err := time.Parse("2006-01-02T15:04", r.PostFormValue("date"))
		if err != nil {
			a.Redirect(w, r, "/admin", "flash_erreur_champs")
			return
		}
		tarif := parseFloat(r.PostFormValue("tarif"), 0)
		places := 0
		if v := r.PostFormValue("places"); v != "" {
			places = int(parseFloat(v, 0))
		}
		e := evenements.Evenement{
			Nom: r.PostFormValue("nom"), Sport: r.PostFormValue("sport"),
			Date: date, Lieu: r.PostFormValue("lieu"),
			Description: r.PostFormValue("description"),
			Tarif: tarif, PlacesMax: places,
			ExigeCotiz: r.PostFormValue("exige_cotiz") == "on",
			ExigeLicence: r.PostFormValue("exige_licence") == "on",
		}
		if _, err := a.Evenements.Creer(e); err != nil {
			a.Redirect(w, r, "/admin", "flash_erreur_champs")
			return
		}
		a.Redirect(w, r, "/admin", "")
	})
	a.Router.Handle("POST", "/admin/evenements/{id}/cloturer", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		e, ok := a.Evenements.ParID(p["id"])
		if ok {
			a.Evenements.SetCloturee(e.ID, !e.Cloturee)
		}
		a.Redirect(w, r, "/admin", "")
	})
	a.Router.Handle("POST", "/admin/evenements/{id}/supprimer", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		a.Evenements.Supprimer(p["id"])
		a.Redirect(w, r, "/admin", "flash_supprime")
	})

	// ---- articles boutique ----
	a.Router.Handle("POST", "/admin/articles", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		art := boutique.Article{
			Nom: r.PostFormValue("nom"), Description: r.PostFormValue("description"),
			Prix: parseFloat(r.PostFormValue("prix"), 0),
			Stock: int(parseFloat(r.PostFormValue("stock"), 0)),
		}
		if _, err := a.Boutique.CreerArticle(art); err != nil {
			a.Redirect(w, r, "/admin", "flash_erreur_champs")
			return
		}
		a.Redirect(w, r, "/admin", "")
	})
	a.Router.Handle("POST", "/admin/articles/{id}/supprimer", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		a.Boutique.SupprimerArticle(p["id"])
		a.Redirect(w, r, "/admin", "flash_supprime")
	})

	// ---- comptes : rôles et statuts de paiement ----
	a.Router.Handle("POST", "/admin/comptes/{id}/role-admin", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		c, ok := a.Comptes.ParID(p["id"])
		if ok {
			a.Comptes.SetRole(c.ID, comptes.RoleAdmin, !c.AUnRole(comptes.RoleAdmin))
		}
		a.Redirect(w, r, "/admin", "")
	})
	a.Router.Handle("POST", "/admin/comptes/{id}/cotiz", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		c, ok := a.Comptes.ParID(p["id"])
		if ok {
			a.Comptes.SetPaiement(c.ID, "cotiz", !c.CotizPayee)
		}
		a.Redirect(w, r, "/admin", "")
	})
	a.Router.Handle("POST", "/admin/comptes/{id}/licence", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		c, ok := a.Comptes.ParID(p["id"])
		if ok {
			a.Comptes.SetPaiement(c.ID, "licence", !c.LicencePayee)
		}
		a.Redirect(w, r, "/admin", "")
	})
	a.Router.Handle("GET", "/admin/comptes.csv", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="comptes.csv"`)
		w.Write([]byte(a.Comptes.ExportCSV()))
	})
	a.Router.Handle("GET", "/admin/evenements/{id}/inscrits.csv", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="inscrits.csv"`)
		w.Write([]byte(a.Evenements.ExportCSV(p["id"], a.Comptes)))
	})

	// ---- paiements : validation manuelle (mode démo) ----
	a.Router.Handle("POST", "/admin/paiements/{id}/payer", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		if err := a.Paiements.MarquerPaye(p["id"]); err != nil {
			a.Redirect(w, r, "/admin", "flash_erreur_champs")
			return
		}
		a.Redirect(w, r, "/admin", "flash_paye")
	})

	// ---- purge annuelle (fin d'année scolaire) ----
	a.Router.Handle("POST", "/admin/purge", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin) == nil {
			return
		}
		a.Sessions.PurgeExpired()
		a.Redirect(w, r, "/admin", "flash_supprime")
	})
}

func parseFloat(s string, def float64) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return def
}
