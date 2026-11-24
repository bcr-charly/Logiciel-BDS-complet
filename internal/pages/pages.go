// Package pages enregistre toutes les pages auprès du noyau (une page = un
// module qui s'enregistre, le noyau ne connaît aucune page).
package pages

import (
	"bds/internal/app"
	"bds/internal/pages/accueil"
	"bds/internal/pages/admin"
	"bds/internal/pages/auth"
	"bds/internal/pages/boutique"
	"bds/internal/pages/evenements"
	"bds/internal/pages/resp"
	"bds/internal/pages/webhook"
)

func RegisterAll(a *app.App) {
	accueil.Register(a)
	auth.Register(a)
	evenements.Register(a)
	boutique.Register(a)
	admin.Register(a)
	resp.Register(a)
	webhook.Register(a)
}
