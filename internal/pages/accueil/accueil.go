// Package accueil : page d'accueil (maquette : grand logo + boutons
// "Nos sports" / "Inscription" + bande d'annonces), actus, sports, page BDS.
package accueil

import (
	"net/http"

	"bds/internal/app"
	"bds/internal/kernel"
)

func Register(a *app.App) {
	a.Router.Handle("GET", "/", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		var acc app.Accueil
		_ = a.ContenuJSON("accueil.json", &acc)
		data := map[string]any{
			"annonces": a.Annonces.Recentes(6),
			"accueil":  acc,
		}
		a.Render(w, r, "accueil", data, http.StatusOK)
	})

	a.Router.Handle("GET", "/actus", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		a.Render(w, r, "actus", map[string]any{"annonces": a.Annonces.Recentes(0)}, http.StatusOK)
	})

	a.Router.Handle("GET", "/sports", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		var sports []app.Sport
		if err := a.ContenuJSON("sports.json", &sports); err != nil {
			sports = nil
		}
		a.Render(w, r, "sports", map[string]any{"sports": sports}, http.StatusOK)
	})

	a.Router.Handle("GET", "/sports/{slug}", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		var sports []app.Sport
		if err := a.ContenuJSON("sports.json", &sports); err != nil {
			http.NotFound(w, r)
			return
		}
		for _, s := range sports {
			if s.Slug == p["slug"] {
				data := map[string]any{
					"sport":     s,
					"annonces":  a.Annonces.PourSport(s.Slug),
					"evenements": a.Evenements.Tous(),
				}
				a.Render(w, r, "sport", data, http.StatusOK)
				return
			}
		}
		http.NotFound(w, r)
	})

	a.Router.Handle("GET", "/bds", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		var equipe []app.MembreEquipe
		_ = a.ContenuJSON("equipe.json", &equipe)
		a.Render(w, r, "bds", map[string]any{"equipe": equipe}, http.StatusOK)
	})
}
