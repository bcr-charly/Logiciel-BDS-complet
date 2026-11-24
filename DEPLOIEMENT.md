# Déploiement — côté ARISE

Le serveur de production appartient à ARISE et tourne sous Docker, **sans accès
administrateur** : on ne livre qu'une image autonome. Le HTTPS, le nom de domaine et
le routage sont assurés par le reverse proxy d'ARISE ; notre bloc n'écoute qu'en HTTP
sur un port interne.

## 1. Construire l'image (CI ou poste du bureau)

```bash
docker build -t bds-ensiie:latest .
# CI : cross-compile linux/amd64 et linux/arm64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bds-amd64 ./cmd/bds
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bds-arm64 ./cmd/bds
```

L'image est multi-stage : compilation dans `golang:1.27-alpine`, exécution sur
`gcr.io/distroless/static-debian12:nonroot` (binaire statique `CGO_ENABLED=0`,
certificats CA présents pour l'appel HTTPS HelloAsso). Rien n'est installé sur
l'hôte.

## 2. Lancer le conteneur

```bash
docker run -d --name bds \
  -p 127.0.0.1:8080:8080 \
  -v bds-data:/app/data \
  -v /etc/bds/conf.txt:/app/conf.txt:ro \
  bds-ensiie:latest
```

- `conf.txt` monté depuis l'hôte : changer `session_secret`, `admin_password`,
  `base_url=https://bds.ensiie.fr`, les clés HelloAsso et SMTP **avant** le premier
  démarrage, puis `docker restart bds`.
- Volume `bds-data` : persistance des fichiers JSON de `data/` (comptes, sessions,
  paiements, inscriptions, équipes, annonces).
- Premier démarrage : créer/mettre à jour l'admin :
  `docker exec bds /app/bds -conf /app/conf.txt -creer-admin admin@bds-ensiie.fr:motdepasse`
  (le flag n'est effectif que pour cette exécution ; pour le compte initial, les
  valeurs de `conf.txt` sont utilisées automatiquement).

## 3. Reverse proxy (ARISE)

Faire router `https://bds.ensiie.fr` vers `http://<hôte-docker>:8080`, par exemple :

```nginx
location / {
  proxy_pass http://127.0.0.1:8080;
  proxy_set_header Host $host;
  proxy_set_header X-Forwarded-Proto https;
}
```

## 4. Webhook HelloAsso (obligatoire pour les paiements)

Le webhook exige une URL publique en HTTPS. À faire router par ARISE **avant** la
mise en production des paiements :

```
https://bds.ensiie.fr/webhook/helloasso/<webhook_secret>
```

Le `<webhook_secret>` est celui de `conf.txt` (le secret d'URL est la seule barrière :
les notifications des comptes association HelloAsso ne sont pas signées, c'est pourquoi
le serveur **re-lit le paiement côté API** avant de le confirmer).

## 5. Surveillance

- `GET /healthz` → `{"ok":true}` (sonde Docker / proxy).
- Logs : `docker logs -f bds` (en mode démo SMTP, les liens de vérification y
  apparaissent).
- Données : contenues dans le volume `bds-data` (`/app/data/*.json`), sauvegardables
  à froid ; ne jamais commiter `data/`.
