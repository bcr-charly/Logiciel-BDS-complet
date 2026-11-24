# Mise en service — côté bureau du BDS

## 1. Lancer en local

```bash
go run ./cmd/bds                       # lit conf.txt, écoute sur host:port
go run ./cmd/bds -- -port 9000         # override du port
```

URL : `http://127.0.0.1:8080` (voir `base_url` dans `conf.txt`).

## 2. Premier démarrage

- La base `data/` est créée automatiquement (fichiers JSON, un par collection).
- Si elle est vide : compte admin créé depuis `admin_email` / `admin_password`
  (défaut `admin@bds-ensiie.fr` / `admin123`) + données de démo.
- Changer le mot de passe admin immédiatement :
  `go run ./cmd/bds -- -creer-admin admin@bds-ensiie.fr:nouveau-mot-de-passe`

## 3. conf.txt (redémarrage requis)

| Clé | Rôle |
| --- | --- |
| `host`, `port`, `base_url` | écoute HTTP (le HTTPS est assuré par le reverse proxy ARISE) |
| `session_secret` | **obligatoire**, 16+ caractères aléatoires (signatures HMAC des sessions) |
| `admin_email`, `admin_password` | compte admin initial |
| `helloasso_slug`, `helloasso_token`, `helloasso_api` | paiements HelloAsso (vide = mode démo, validation manuelle admin) |
| `webhook_secret` | secret d'URL du webhook `/webhook/helloasso/<webhook_secret>` |
| `smtp_host`, `smtp_port`, `smtp_user`, `smtp_password`, `smtp_from` | envoi des e-mails de vérification (vide = mode démo, lien journalisé dans les logs) |
| `duree_session_jours`, `reconciliation_minutes` | session et tâche de fond des paiements |
| `cotiz_montant`, `licence_montant` | montants (en €, optionnels, défaut 15 / 10) |

Générer un secret : `openssl rand -hex 32`.

## 4. Comptes et rôles

- Inscription libre à `/inscription` (e-mail + identifiant + mot de passe ≥ 8 caractères,
  vérification par lien envoyé par e-mail).
- Connexion à `/connexion` par e-mail **ou** identifiant.
- Rôles gérés par l'admin dans `/admin` (bouton « admin » sur chaque compte) ;
  un responsable est désigné par sport (champ `sports` du compte, modifiable via la
  collection `data/comptes.json` ou à terme via l'interface).

## 5. Contenus éditoriaux (sans recompilation)

Modifier `contenu/sports.json`, `contenu/equipe.json`, `contenu/partenaires.json`,
`contenu/accueil.json` puis redémarrer. Les répertoires disque priment sur les
valeurs embarquées dans le binaire.

## 6. Paiements HelloAsso

1. Créer l'accès API (back-office HelloAsso) et récupérer `slug` (organisation) + token.
2. Renseigner `helloasso_slug` / `helloasso_token` dans `conf.txt`, définir un
   `webhook_secret` fort.
3. Demander à ARISE le routage public `https://bds.ensiie.fr/webhook/helloasso/<webhook_secret>`
   (voir DEPLOIEMENT.md).
4. Tester avec le bac à sable HelloAsso (carte `4242 4242 4242 4242`).

Cotisation / licence : boutons sur `/profil`. Événements payants et boutique :
redirection automatique vers le checkout HelloAsso.

## 7. Tests

```bash
go test ./...
```

Couvre : routage, stockage atomique, PBKDF2/HMAC, sessions signées, i18n, rate-limit,
comptes, inscriptions (capacité, double, clôtures, exigences), boutique (stock),
paiements (idempotence, webhook), QR signés, et un flux HTTP complet
(inscription → connexion → profil → inscription événement → admin → annonce).
