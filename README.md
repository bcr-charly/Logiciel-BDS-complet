# Logiciel-BDS-complet

Site web « tout en un » du **BDS de l'ENSIIE** : inscriptions aux compétitions, boutique,
base de données des adhérents et espace d'administration distant (annonces, suivi des
inscriptions et commandes).

## Stack

- **Next.js 15** (App Router, Server Actions, TypeScript)
- **Tailwind CSS v4** — palette injectée comme design tokens (`src/app/globals.css`)
- **Prisma** — SQLite en développement, PostgreSQL recommandé en production (changer
  `DATABASE_URL` et `provider` dans `prisma/schema.prisma`)
- Auth admin : mot de passe hashé (bcrypt) + cookie de session signé (JWT HS256 via `jose`),
  protégé par un middleware sur `/admin/*`

## Palette (direction artistique — maillot noir satin & or)

| Rôle | Hex | Usage |
| --- | --- | --- |
| Noir profond | `#191718` | fond du maillot, zones mates |
| Noir charbon | `#2A2629` | plis et zones intermédiaires |
| Reflet satin | `#555253` | brillances du tissu |
| Bronze sombre | `#6C6149` | ombres des bordures dorées |
| Or champagne | `#B5A681` | teinte centrale de l'or (col, manches, logo) |
| Or satin clair | `#DDCB95` | faces éclairées de l'or |
| Éclat métallique | `#F2E3B3` | points de brillance maximale |

## Modules

- **Accueil** : annonces publiées par l'admin (fil d'actualité)
- **Compétitions** : liste des compétitions, inscription en ligne (crée/met à jour le profil
  adhérent via l'e-mail, gère les places restantes et les clôtures)
- **Boutique** : catalogue produits, commande avec décrément de stock et calcul du total
- **Adhésion** : création du profil adhérent
- **Admin** (`/admin`, protégé) : connexion, statistiques, CRUD des annonces,
  inscriptions et commandes récentes. La base de données reste consultable à distance avec
  `npm run db:studio` (Prisma Studio, à n'exposer qu'en local/VPN)

## Démarrage

```bash
cp .env.example .env          # puis ajuster AUTH_SECRET / compte admin
npm install
npx prisma generate
npx prisma db push            # crée la base SQLite
npx prisma db seed            # compte admin + données de démo
npm run dev                   # http://localhost:3000
```

Compte admin de démo : `admin@bds-ensiie.fr` / `admin123` (à changer via `ADMIN_PASSWORD`).

## Structure

```
prisma/schema.prisma     # modèles : Admin, Adherent, Annonce, Competition,
                         # Inscription, Produit, Commande, LigneCommande
src/app/                 # pages (accueil, compétitions, boutique, adhésion, admin)
src/actions/             # server actions (auth, annonces, adhérents, compétitions, boutique)
src/components/          # Header / Footer (direction artistique)
src/lib/                 # client Prisma, session JWT, formatage
src/middleware.ts        # protection de /admin/*
```

## Production

- Renseigner `DATABASE_URL` (PostgreSQL) et un `AUTH_SECRET` fort dans les variables
  d'environnement ; ne jamais commiter `.env` (déjà ignoré).
- Faire `npx prisma migrate deploy` après passage à PostgreSQL.
- Le middleware vérifie la session sur toutes les routes `/admin/*` hormis `/admin/login`.
