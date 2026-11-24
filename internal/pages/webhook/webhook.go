// Package webhook : webhook HelloAsso (secret d'URL) et vérification des QR
// d'inscription (pointage par un responsable ou l'admin).
package webhook

import (
	"io"
	"net/http"

	"bds/internal/app"
	"bds/internal/kernel"
	"bds/internal/metier/comptes"
	"bds/internal/metier/qrcode"
)

func Register(a *app.App) {
	// Webhook HelloAsso : /webhook/helloasso/<secret> (le secret d'URL est la
	// seule barrière, les notifications des comptes association ne sont pas
	// signées ; le paiement est re-lu côté API avant confirmation).
	a.Router.Handle("POST", "/webhook/helloasso/{secret}", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "lecture impossible", http.StatusBadRequest)
			return
		}
		if err := a.Paiements.TraiterWebhook(p["secret"], body); err != nil {
			http.Error(w, "webhook refusé", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})

	// Vérification de QR (phase 2) : saisie du token signé.
	a.Router.Handle("GET", "/qr/verifier", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin, comptes.RoleResponsable) == nil {
			return
		}
		a.Render(w, r, "qr", map[string]any{"resultat": ""}, http.StatusOK)
	})

	a.Router.Handle("POST", "/qr/verifier", func(w http.ResponseWriter, r *http.Request, p kernel.Params) {
		if a.RequireRole(w, r, comptes.RoleAdmin, comptes.RoleResponsable) == nil {
			return
		}
		resultat := "qr_invalide"
		payload, err := a.QR.Verifier(r.PostFormValue("token"))
		if err == nil {
			if _, ok := a.Evenements.InscriptionCompte(payload.CompteID, payload.EvenementID); ok && payload.InscriptionID != "" {
				ok, err := a.Evenements.Pointer(payload.InscriptionID)
				switch {
				case err != nil:
					resultat = "qr_invalide"
				case !ok:
					resultat = "qr_deja_pointe"
				default:
					resultat = "qr_valide"
				}
			} else {
				resultat = "qr_invalide"
			}
		} else if err == qrcode.ErrTokenExpire {
			resultat = "qr_invalide"
		}
		a.Render(w, r, "qr", map[string]any{"resultat": resultat, "token": r.PostFormValue("token")}, http.StatusOK)
	})
}
