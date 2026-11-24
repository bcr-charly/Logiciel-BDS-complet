// Package resp : espace responsable de sport — adhérents de son sport (statut
// cotiz/licence), équipes, annonces de son sport. Un responsable ne voit que
// son sport.
package resp

import (
	"net/http"
	"time"

	"bds/internal/app"
	"bds/internal/kernel"
	"bds/internal/metier/annonces"
	"bds/internal/metier/comptes"
	"bds/internal/metier/equipes"
)

func peutVoir(a *app.App, r *http.Request, sport string) bool {
	u := a.User(r)
	if u == nil {
		return false
	}
	c, ok := a.Comptes.ParID(u.ID)
	return ok && c.EstResponsableDe(sport)
}

func Register(a *app.App) {
	a.Router.Handle("GET", "/resp", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		u := a.RequireRole(w, r, comptes.RoleResponsable, comptes.RoleAdmin)
		if u == nil {
			return
		}
		c, _ := a.Comptes.ParID(u.ID)
		if c.AUnRole(comptes.RoleAdmin) || len(c.Sports) == 0 {
			a.Redirect(w, r, "/admin", "")
			return
		}
		a.Redirect(w, r, "/resp/"+c.Sports[0], "")
	})

	a.Router.Handle("GET", "/resp/{sport}", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		sport := p["sport"]
		if !peutVoir(a, r, sport) {
			a.Redirect(w, r, "/", "flash_acces_refuse")
			return
		}
		var adherents []comptes.Compte
		for _, c := range a.Comptes.Tous() {
			if c.SportPrefere == sport {
				adherents = append(adherents, c)
			}
		}
		var sports []app.Sport
		_ = a.ContenuJSON("sports.json", &sports)
		nomSport := sport
		for _, s := range sports {
			if s.Slug == sport {
				nomSport = s.Nom
			}
		}
		data := map[string]any{
			"sport":     sport,
			"nom_sport": nomSport,
			"adherents": adherents,
			"equipes":   a.Equipes.ParSport(sport),
			"annonces":  a.Annonces.PourSport(sport),
		}
		a.Render(w, r, "resp", data, http.StatusOK)
	})

	a.Router.Handle("POST", "/resp/{sport}/annonces", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		sport := p["sport"]
		if !peutVoir(a, r, sport) {
			a.Redirect(w, r, "/", "flash_acces_refuse")
			return
		}
		typ := r.PostFormValue("type")
		if typ != annonces.TypeOuverture && typ != annonces.TypeResultats && typ != annonces.TypeInfo {
			typ = annonces.TypeInfo
		}
		a.Annonces.Creer(typ, r.PostFormValue("titre"), r.PostFormValue("contenu"), sport, a.User(r).ID)
		a.Redirect(w, r, "/resp/"+sport, "flash_annonce_creee")
	})

	a.Router.Handle("POST", "/resp/{sport}/equipes", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		sport := p["sport"]
		if !peutVoir(a, r, sport) {
			a.Redirect(w, r, "/", "flash_acces_refuse")
			return
		}
		date, err := time.Parse("2006-01-02T15:04", r.PostFormValue("date"))
		if err != nil {
			a.Redirect(w, r, "/resp/"+sport, "flash_erreur_champs")
			return
		}
		e := equipes.Equipe{
			Sport: sport, Nom: r.PostFormValue("nom"), Adversaire: r.PostFormValue("adversaire"),
			Date: date, Lieu: r.PostFormValue("lieu"),
		}
		a.Equipes.Creer(e)
		a.Redirect(w, r, "/resp/"+sport, "")
	})

	a.Router.Handle("POST", "/resp/{sport}/equipes/{id}/supprimer", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		sport := p["sport"]
		if !peutVoir(a, r, sport) {
			a.Redirect(w, r, "/", "flash_acces_refuse")
			return
		}
		a.Equipes.Supprimer(p["id"])
		a.Redirect(w, r, "/resp/"+sport, "flash_supprime")
	})

	a.Router.Handle("GET", "/resp/{sport}/equipes.csv", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		sport := p["sport"]
		if !peutVoir(a, r, sport) {
			a.Redirect(w, r, "/", "flash_acces_refuse")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="equipes-`+sport+`.csv"`)
		w.Write([]byte(a.Equipes.ExportCSV(sport, a.Comptes)))
	})
}
