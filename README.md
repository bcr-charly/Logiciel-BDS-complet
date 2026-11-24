# Logiciel-BDS-complet

Site web « tout en un » du **BDS de l'ENSIIE** : comptes et adhésion, événements
(compétitions inter-écoles, FFSU) avec inscriptions, boutique, annonces, paiements
HelloAsso et espace admin / responsables de sport.

Implémenté conformément à la **SPEC V2** : serveur **Go, bibliothèque standard
exclusivement**, rendu serveur `html/template` + JS léger écrit à la main, stockage
**fichiers JSON plats** (un par collection, réécriture atomique), livrable **binaire
unique / image Docker autonome** déployable sans dépendances sur la machine de l'asso
ARISE.

## Identité visuelle

Palette officielle (maillot noir satin & or), appliquée dans `web/style.css` :

| Rôle | Hex | Usage |
| --- | --- | --- |
| Noir profond | `#191718` | fond, zones mates |
| Noir charbon | `#2A2629` | cartes, plis |
| Reflet satin | `#555253` | brillances du tissu |
| Bronze sombre | `#6C6149` | ombres des bordures dorées |
| Or champagne | `#B5A681` | teinte centrale de l'or (logo, bordures) |
| Or satin clair | `#DDCB95` | faces éclairées de l'or |
| Éclat métallique | `#F2E3B3` | points de brillance maximale |

Logo officiel (bouclier hyène) : `assets/logo-bds.jpg`, servi en header et en grand
sur l'accueil (maquette page 1 : header logo + boutons Sports / BDS / Boutique /
Connexion + drapeau de langue ; corps = grand logo + boutons « Nos sports » /
« Inscription » ; dessous = bande d'annonces).

## Architecture

```
cmd/bds/            point d'entrée (flags : -conf, -port, -creer-admin email:mdp)
internal/kernel/    serveur HTTP, routage, sessions signées HMAC, i18n fr/en,
                    rendu html/template, stockage JSON atomique, CSP, rate-limit, CSRF
internal/metier/    un paquet par domaine (appelé par les pages, jamais le stockage direct)
  comptes/          comptes, rôles (adherent/responsable/admin), PBKDF2, cotiz/licence
  annonces/         actus (ouverture / résultats / info), globales ou par sport
  evenements/       compétitions + inscriptions (capacité, exigences de paiement)
  equipes/          composition des équipes par sport, export CSV
  boutique/         articles + commandes (décrément de stock)
  paiements/        HelloAsso Checkout v5 : checkout, webhook (secret d'URL + relecture
                    API), réconciliation 45 min, idempotence ; aucune donnée bancaire
  qrcode/           tokens QR signés HMAC (vérification + anti-double-pointage)
internal/pages/     un module par page, enregistré auprès du noyau à l'init
internal/app/       composition noyau + métier, middlewares, static embarqué
contenu/            contenus éditoriaux éditables sans recompilation (sports, équipe, partenaires)
data/               données persistantes (JSON) — ne jamais commiter (gitignored)
assets/ web/        logo/photos, CSS, JS (embarqués dans le binaire, disque prioritaire)
conf.txt            configuration (redémarrage requis)
```

Règles de la spec respectées : le noyau ne connaît aucune page (elles s'enregistrent) ;
aucun appel distant dans les pages livrées (seul le module paiements appelle HelloAsso,
exception prévue) ; CSP stricte `default-src 'self'` ; mots de passe PBKDF2 ; cookies
de session signés HMAC ; rate-limit sur connexion/inscription ; CSRF sur les POST.

## Développement

```bash
go build ./...          # compile
go test ./...           # tests (noyau, métier, flux HTTP complet)
go run ./cmd/bds        # lance avec conf.txt (http://127.0.0.1:8080)
```

Premier démarrage : compte admin créé depuis `conf.txt`
(`admin_email` / `admin_password`, par défaut `admin@bds-ensiie.fr` / `admin123`),
ou via `go run ./cmd/bds -- -creer-admin admin@bds-ensiie.fr:motdepasse`. Données de
démo (annonces, événements, articles) seedées si la base est vide.

## Rôles

| Rôle | Droits |
| --- | --- |
| adhérent | profil, inscriptions aux événements, boutique, historique, QR |
| responsable | son sport uniquement : adhérents (cotiz/licence), équipes, inscriptions, actus |
| admin | tout : rôles, événements, articles, actus globales, exports CSV, purge, paiements |

## Paiements

HelloAsso Checkout API v5 : la page crée un checkout-intent (métadonnées = adhérent +
objet), redirige vers HelloAsso, le webhook (`/webhook/helloasso/<secret>`) est vérifié
par secret d'URL puis relecture du paiement côté API, et une tâche de fond marque
abandonnés les paiements sans notification après 45 min. **Idempotence** : un paiement
déjà traité ne déclenche jamais deux fois son effet. Sans configuration HelloAsso
(mode démo), les paiements restent `EN_ATTENTE` et sont validés manuellement par
l'admin. Aucune donnée bancaire n'est stockée localement.

## Docs

- [MISE-EN-SERVICE.md](MISE-EN-SERVICE.md) — côté bureau (conf, comptes, HelloAsso, SMTP)
- [DEPLOIEMENT.md](DEPLOIEMENT.md) — côté ARISE (image Docker, reverse proxy, webhook)

## Phase 2 prévue

Génération de l'image QR (le token signé HMAC + vérification sont déjà en place) ;
envoi d'e-mails SMTP réel (le lien de vérification est journalisé en mode démo).
