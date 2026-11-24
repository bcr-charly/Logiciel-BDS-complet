import { PrismaClient } from "@prisma/client";
import bcrypt from "bcryptjs";

const prisma = new PrismaClient();

const dans = (jours: number) => new Date(Date.now() + jours * 24 * 60 * 60 * 1000);

async function main() {
  const email = process.env.ADMIN_EMAIL ?? "admin@bds-ensiie.fr";
  const password = process.env.ADMIN_PASSWORD ?? "admin123";
  const hash = await bcrypt.hash(password, 10);

  await prisma.admin.upsert({
    where: { email },
    update: {},
    create: { email, password: hash, nom: "Administrateur BDS" },
  });
  console.log(`Admin prêt : ${email} (mot de passe : ${password})`);

  if ((await prisma.annonce.count()) === 0) {
    await prisma.annonce.createMany({
      data: [
        {
          titre: "Bienvenue sur le nouveau site du BDS",
          contenu:
            "Inscriptions aux compétitions, boutique officielle et espace adhérents sont désormais réunis au même endroit. Connecte-toi à ton espace admin pour publier des annonces.",
        },
        {
          titre: "Tournoi inter-écoles de football",
          contenu:
            "Les inscriptions sont ouvertes dans la rubrique Compétitions. Places limitées, ne tardez pas !",
        },
        {
          titre: "Nouvelle collection boutique",
          contenu:
            "Sweats, t-shirts et accessoires aux couleurs du maillot : noir satin et or champagne, à retrouver dans la boutique.",
        },
      ],
    });
  }

  if ((await prisma.competition.count()) === 0) {
    await prisma.competition.createMany({
      data: [
        {
          nom: "Tournoi de football inter-écoles",
          sport: "Football",
          date: dans(14),
          lieu: "Stade municipal de Strasbourg",
          description: "Tournoi 7 vs 7, équipes de 8 joueurs. Inscriptions par équipe ou individuels.",
          tarif: 5,
          places: 64,
        },
        {
          nom: "Cross de l'ENSIIE",
          sport: "Course à pied",
          date: dans(21),
          lieu: "Campus de l'ENSIIE",
          description: "Course de 5 km ouverte à tous les niveaux.",
          tarif: 0,
          places: null,
        },
        {
          nom: "Tournoi de basketball 3x3",
          sport: "Basketball",
          date: dans(30),
          lieu: "Gymnase du lycée",
          description: "Tournoi 3 contre 3, équipes de 4 joueurs.",
          tarif: 3,
          places: 32,
        },
      ],
    });
  }

  if ((await prisma.produit.count()) === 0) {
    await prisma.produit.createMany({
      data: [
        {
          nom: "Sweat à capuche BDS",
          description: "Satin noir profond, col et manches en or champagne, logo brodé éclat métallique.",
          prix: 45,
          stock: 25,
        },
        {
          nom: "T-shirt BDS",
          description: "Coupe classique, maillot noir charbon avec logo or satin clair.",
          prix: 20,
          stock: 40,
        },
        {
          nom: "Gourde inox BDS",
          description: "Inox brossé, finition bronze sombre, gravure laser du logo.",
          prix: 12,
          stock: 50,
        },
        {
          nom: "Écharpe supporter",
          description: "Noir profond avec franges or champagne et reflets satin.",
          prix: 15,
          stock: 30,
        },
      ],
    });
  }

  if ((await prisma.adherent.count()) === 0) {
    await prisma.adherent.create({
      data: {
        prenom: "Jeanne",
        nom: "Dupont",
        email: "jeanne.dupont@ensiie.fr",
        telephone: "06 12 34 56 78",
        annee: "2A",
        sportPrefere: "Basketball",
      },
    });
  }
}

main()
  .catch((e) => {
    console.error(e);
    process.exit(1);
  })
  .finally(async () => {
    await prisma.$disconnect();
  });
