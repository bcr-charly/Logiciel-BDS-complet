import Link from "next/link";
import { prisma } from "@/lib/db";
import { formatDateCourt } from "@/lib/format";

export const dynamic = "force-dynamic";

const cartes = [
  {
    href: "/competitions",
    titre: "Compétitions",
    texte: "Inscris-toi aux tournois et compétitions universitaires en quelques clics.",
  },
  {
    href: "/boutique",
    titre: "Boutique",
    texte: "Équipe-toi avec les produits officiels du BDS, aux couleurs du maillot.",
  },
  {
    href: "/adhesion",
    titre: "Adhésion",
    texte: "Crée ton profil adhérent et rejoins la communauté sportive de l'ENSIIE.",
  },
];

export default async function Home() {
  const annonces = await prisma.annonce.findMany({
    where: { publiee: true },
    orderBy: { createdAt: "desc" },
    take: 6,
  });
  const nbCompetitions = await prisma.competition.count({ where: { cloturee: false } });

  return (
    <div>
      <section className="mx-auto max-w-6xl px-4 pb-16 pt-20 text-center">
        <p className="badge-or inline-block rounded-full px-4 py-1 text-xs">
          Bureau des Sports — ENSIIE
        </p>
        <h1 className="texte-or mt-6 text-5xl font-extrabold uppercase leading-tight tracking-wide sm:text-6xl">
          Le sport, le style,
          <br />
          la communauté
        </h1>
        <p className="mx-auto mt-6 max-w-2xl text-lg text-or-satin/90">
          Inscriptions aux compétitions, boutique officielle et espace adhérents du BDS de
          l'ENSIIE, réunis sur un seul site.
        </p>
        <div className="mt-10 flex flex-wrap items-center justify-center gap-4">
          <Link
            href="/competitions"
            className="bouton-or rounded-lg px-6 py-3 text-sm shadow-lg shadow-black/40"
          >
            Voir les compétitions ({nbCompetitions})
          </Link>
          <Link
            href="/boutique"
            className="bouton-contour rounded-lg px-6 py-3 text-sm"
          >
            Découvrir la boutique
          </Link>
        </div>
      </section>

      <div className="ligne-or mx-auto max-w-6xl" />

      <section className="mx-auto max-w-6xl px-4 py-14">
        <div className="mb-8 flex items-end justify-between">
          <h2 className="texte-or text-3xl font-extrabold uppercase tracking-widest">Annonces</h2>
        </div>
        {annonces.length === 0 ? (
          <p className="text-or-champagne/80">
            Aucune annonce pour le moment. Reviens bientôt !
          </p>
        ) : (
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {annonces.map((a) => (
              <article key={a.id} className="carte-satin rounded-xl p-6">
                <p className="text-xs uppercase tracking-widest text-reflet-satin">
                  {formatDateCourt(a.createdAt)}
                </p>
                <h3 className="texte-or mt-2 text-xl font-bold">{a.titre}</h3>
                <p className="mt-3 text-sm leading-relaxed text-or-satin/90">{a.contenu}</p>
              </article>
            ))}
          </div>
        )}
      </section>

      <section className="mx-auto max-w-6xl px-4 pb-8">
        <div className="grid gap-5 md:grid-cols-3">
          {cartes.map((c) => (
            <Link
              key={c.href}
              href={c.href}
              className="carte-satin group rounded-xl p-6 transition hover:border-or-champagne/70"
            >
              <h3 className="texte-or text-xl font-extrabold uppercase tracking-widest">
                {c.titre}
              </h3>
              <p className="mt-3 text-sm leading-relaxed text-or-satin/90">{c.texte}</p>
              <p className="mt-4 text-xs uppercase tracking-widest text-or-champagne transition group-hover:text-eclat">
                Accéder →
              </p>
            </Link>
          ))}
        </div>
      </section>
    </div>
  );
}
