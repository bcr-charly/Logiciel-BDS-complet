package metier_test

import (
	"os"
	"testing"
	"time"

	"bds/internal/kernel"
	"bds/internal/metier/annonces"
	"bds/internal/metier/boutique"
	"bds/internal/metier/comptes"
	"bds/internal/metier/evenements"
	"bds/internal/metier/paiements"
	"bds/internal/metier/qrcode"
)

func nouveauStore(t *testing.T) *kernel.Store {
	t.Helper()
	store, err := kernel.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

type qrFaux struct{}

func (qrFaux) Generer(inscriptionID, compteID, evenementID string, exp time.Time) string {
	return "qr-" + inscriptionID
}

func TestComptes(t *testing.T) {
	store := nouveauStore(t)
	s, err := comptes.New(store)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Creer("jeanne@ensiie.fr", "jeanne", "motdepasse123", "Dupont", "Jeanne", "", "2A", "basketball")
	if err != nil {
		t.Fatal(err)
	}
	if !c.AUnRole(comptes.RoleAdherent) {
		t.Fatal("rôle adhérent manquant")
	}
	if _, err := s.Creer("jeanne@ensiie.fr", "autre", "motdepasse123", "", "", "", "", ""); err != comptes.ErrEmailPris {
		t.Fatalf("email dupliqué: %v", err)
	}
	if _, err := s.Login("jeanne", "motdepasse123"); err != nil {
		t.Fatalf("login par identifiant: %v", err)
	}
	if _, err := s.Login("jeanne@ensiie.fr", "mauvais"); err != comptes.ErrMauvaisIdent {
		t.Fatalf("mauvais mdp: %v", err)
	}
	if !s.VerifierEmail(c.VerifToken) {
		t.Fatal("vérification email")
	}
	got, _ := s.ParID(c.ID)
	if !got.EmailVerifie {
		t.Fatal("email devrait être vérifié")
	}
	admin, err := s.CreerAdmin("admin@bds-ensiie.fr", "admin12345", "Admin")
	if err != nil {
		t.Fatal(err)
	}
	if !admin.AUnRole(comptes.RoleAdmin) {
		t.Fatal("rôle admin manquant")
	}
}

func TestEvenementsInscription(t *testing.T) {
	store := nouveauStore(t)
	cs, _ := comptes.New(store)
	es, err := evenements.New(store, qrFaux{})
	if err != nil {
		t.Fatal(err)
	}
	c, err := cs.Creer("a@ensiie.fr", "aaa", "motdepasse123", "A", "A", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	maj := func(id string) comptes.Compte { cv, _ := cs.ParID(id); return cv }

	e, err := es.Creer(evenements.Evenement{
		Nom: "Tournoi", Sport: "football", Date: time.Now().Add(7 * 24 * time.Hour),
		Lieu: "Stade", Tarif: 0, PlacesMax: 2, ExigeCotiz: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// cotiz non payée -> refus
	ca := maj(c.ID)
	if _, _, err := es.Inscrire(&ca, *e); err != evenements.ErrCotizRequise {
		t.Fatalf("attendu ErrCotizRequise, got %v", err)
	}
	cs.SetPaiement(c.ID, "cotiz", true)
	ca = maj(c.ID)
	insc, gratuit, err := es.Inscrire(&ca, *e)
	if err != nil || gratuit != "" {
		t.Fatalf("inscription gratuite: insc=%+v gratuit=%q err=%v", insc, gratuit, err)
	}
	if insc.Statut != evenements.StatutConfirmee || insc.QRToken == "" {
		t.Fatalf("inscription gratuite devrait être confirmée avec QR: %+v", insc)
	}
	// double inscription
	if _, _, err := es.Inscrire(&ca, *e); err != evenements.ErrDejaInscrit {
		t.Fatalf("attendu ErrDejaInscrit, got %v", err)
	}
	// capacité
	c2, _ := cs.Creer("b@ensiie.fr", "bbb", "motdepasse123", "B", "B", "", "", "")
	cs.SetPaiement(c2.ID, "cotiz", true)
	c2v := maj(c2.ID)
	if _, _, err := es.Inscrire(&c2v, *e); err != nil {
		t.Fatal(err)
	}
	c3, _ := cs.Creer("c@ensiie.fr", "ccc", "motdepasse123", "C", "C", "", "", "")
	cs.SetPaiement(c3.ID, "cotiz", true)
	c3v := maj(c3.ID)
	if _, _, err := es.Inscrire(&c3v, *e); err != evenements.ErrComplet {
		t.Fatalf("attendu ErrComplet, got %v", err)
	}
	// clôturé
	es.SetCloturee(e.ID, true)
	c4, _ := cs.Creer("d@ensiie.fr", "ddd", "motdepasse123", "D", "D", "", "", "")
	cs.SetPaiement(c4.ID, "cotiz", true)
	c4v := maj(c4.ID)
	if _, _, err := es.Inscrire(&c4v, *e); err != evenements.ErrCloture {
		t.Fatalf("attendu ErrCloture, got %v", err)
	}
	// pointage QR anti-double
	ok, err := es.Pointer(insc.ID)
	if err != nil || !ok {
		t.Fatalf("pointage: %v %v", ok, err)
	}
	ok, err = es.Pointer(insc.ID)
	if err != nil || ok {
		t.Fatalf("double pointage devrait être refusé: %v %v", ok, err)
	}
}

func TestBoutiqueCommande(t *testing.T) {
	store := nouveauStore(t)
	bs, _ := boutique.New(store)
	art, err := bs.CreerArticle(boutique.Article{Nom: "Sweat", Prix: 45, Stock: 2})
	if err != nil {
		t.Fatal(err)
	}
	cmd, err := bs.Commander("", "x@ensiie.fr", art.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Total != 45 {
		t.Fatalf("total = %v", cmd.Total)
	}
	if _, err := bs.Commander("", "y@ensiie.fr", art.ID, 5); err != boutique.ErrStockInsuffisant {
		t.Fatalf("attendu stock insuffisant, got %v", err)
	}
	a, _ := bs.ArticleParID(art.ID)
	if a.Stock != 1 {
		t.Fatalf("stock = %d", a.Stock)
	}
}

func TestPaiementsIdempotence(t *testing.T) {
	store := nouveauStore(t)
	dir := t.TempDir()
	confPath := dir + "/conf.txt"
	if err := os.WriteFile(confPath, []byte("session_secret=secret-de-test-32-caracteres\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	conf, err := kernel.LoadConf(confPath)
	if err != nil {
		t.Fatal(err)
	}
	cs, _ := comptes.New(store)
	es, _ := evenements.New(store, qrFaux{})
	bs, _ := boutique.New(store)
	ps, err := paiements.New(store, conf, cs, es, bs)
	if err != nil {
		t.Fatal(err)
	}
	c, err := cs.Creer("p@ensiie.fr", "payeur", "motdepasse123", "P", "P", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	pay, err := ps.CreerPaiement(c.ID, paiements.ObjetCotiz, "", 15, "Cotisation", "http://localhost/profil")
	if err != nil {
		t.Fatal(err)
	}
	if pay.Statut != paiements.PayEnAttente || pay.CheckoutURL != "" {
		t.Fatalf("mode démo attendu sans URL: %+v", pay)
	}
	if err := ps.MarquerPaye(pay.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := cs.ParID(c.ID)
	if !got.CotizPayee {
		t.Fatal("cotiz devrait être payée après paiement")
	}
	if err := ps.MarquerPaye(pay.ID); err != paiements.ErrDejaTraite {
		t.Fatalf("second marquage devrait être idempotent, got %v", err)
	}
	// webhook avec mauvais secret
	if err := ps.TraiterWebhook("mauvais", []byte(`{"data":{"id":1}}`)); err == nil {
		t.Fatal("webhook avec mauvais secret devrait échouer")
	}
}

func TestQRCode(t *testing.T) {
	s := qrcode.New([]byte("secret-de-test-32-caracteres-ok"))
	tok := s.Generer("ins1", "cpt1", "evt1", time.Now().Add(time.Hour))
	p, err := s.Verifier(tok)
	if err != nil {
		t.Fatal(err)
	}
	if p.InscriptionID != "ins1" || p.EvenementID != "evt1" {
		t.Fatalf("payload: %+v", p)
	}
	if _, err := s.Verifier(tok + "x"); err == nil {
		t.Fatal("token modifié devrait être refusé")
	}
	tokExp := s.Generer("ins1", "cpt1", "evt1", time.Now().Add(-time.Hour))
	if _, err := s.Verifier(tokExp); err != qrcode.ErrTokenExpire {
		t.Fatalf("attendu expiration, got %v", err)
	}
}

func TestAnnonces(t *testing.T) {
	store := nouveauStore(t)
	as, _ := annonces.New(store)
	a1, _ := as.Creer(annonces.TypeOuverture, "Inscriptions ouvertes", "contenu", "football", "x")
	as.Creer(annonces.TypeInfo, "Actu globale", "contenu", "", "x")
	if len(as.Recentes(10)) != 2 {
		t.Fatal("recentes")
	}
	if len(as.PourSport("football")) != 2 {
		t.Fatal("pour sport inclut globales")
	}
	if len(as.PourSport("basketball")) != 1 {
		t.Fatal("pour autre sport = globales seules")
	}
	as.Basculer(a1.ID)
	if len(as.Recentes(10)) != 1 {
		t.Fatal("basculer devrait masquer")
	}
}


