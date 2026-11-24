package app

import (
	"log"
	"time"

	"bds/internal/kernel"
	"bds/internal/metier/annonces"
	"bds/internal/metier/boutique"
	"bds/internal/metier/comptes"
	"bds/internal/metier/evenements"
)

// seed amorce la base si elle est vide : compte admin (conf.txt), annonces,
// événements et articles de démo.
func seed(conf *kernel.Conf, c *comptes.Service, a *annonces.Service, e *evenements.Service, b *boutique.Service) error {
	if c.Tous() == nil || len(c.Tous()) == 0 {
		admin, err := c.CreerAdmin(conf.Get("admin_email", "admin@bds-ensiie.fr"), conf.Get("admin_password", "admin123"), "Administrateur BDS")
		if err != nil {
			return err
		}
		log.Printf("compte admin créé : %s (mot de passe : conf.txt admin_password)", admin.Email)
	}
	if len(a.Toutes()) == 0 {
		a.Creer(annonces.TypeInfo, "Bienvenue sur le nouveau site du BDS",
			"Inscriptions aux compétitions, boutique et actus du BDS sont réunis ici. Connecte-toi à l'espace admin pour publier des annonces.", "", "seed")
		a.Creer(annonces.TypeOuverture, "Tournoi de football inter-écoles",
			"Les inscriptions sont ouvertes dans la rubrique Événements. Places limitées !", "football", "seed")
	}
	if len(e.Tous()) == 0 {
		dans := func(j int) time.Time { return time.Now().Add(time.Duration(j) * 24 * time.Hour) }
		e.Creer(evenements.Evenement{Nom: "Tournoi de football inter-écoles", Sport: "football", Date: dans(14), Lieu: "Stade municipal", Description: "Tournoi 7 vs 7, équipes de 8 joueurs.", Tarif: 5, PlacesMax: 64})
		e.Creer(evenements.Evenement{Nom: "Cross de l'ENSIIE", Sport: "course-a-pied", Date: dans(21), Lieu: "Campus de l'ENSIIE", Description: "Course de 5 km ouverte à tous les niveaux.", Tarif: 0, PlacesMax: 0})
		e.Creer(evenements.Evenement{Nom: "Tournoi de basketball 3x3", Sport: "basketball", Date: dans(30), Lieu: "Gymnase du lycée", Description: "Tournoi 3 contre 3, équipes de 4 joueurs.", Tarif: 3, PlacesMax: 32, ExigeCotiz: true})
	}
	if len(b.TousArticles()) == 0 {
		b.CreerArticle(boutique.Article{Nom: "Sweat à capuche BDS", Description: "Satin noir profond, col et manches or champagne, logo brodé.", Prix: 45, Stock: 25})
		b.CreerArticle(boutique.Article{Nom: "T-shirt BDS", Description: "Maillot noir charbon, logo or satin clair.", Prix: 20, Stock: 40})
		b.CreerArticle(boutique.Article{Nom: "Gourde inox BDS", Description: "Inox brossé, finition bronze sombre, gravure laser.", Prix: 12, Stock: 50})
		b.CreerArticle(boutique.Article{Nom: "Écharpe supporter", Description: "Noir profond, franges or champagne.", Prix: 15, Stock: 30})
	}
	return nil
}
