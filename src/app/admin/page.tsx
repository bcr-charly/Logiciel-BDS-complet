import { redirect } from "next/navigation";
import { prisma } from "@/lib/db";
import { getSession } from "@/lib/auth";
import { formatDate, formatPrix } from "@/lib/format";
import { logout } from "@/actions/auth";
import { basculerPublication, creerAnnonce, supprimerAnnonce } from "@/actions/annonces";

export const dynamic = "force-dynamic";

type SearchParams = Promise<Record<string, string | string[] | undefined>>;

export default async function AdminPage({ searchParams }: { searchParams: SearchParams }) {
  const session = await getSession();
  if (!session) redirect("/admin/login");
  const sp = await searchParams;

  const [annonces, inscriptions, commandes, stats] = await Promise.all([
    prisma.annonce.findMany({ orderBy: { createdAt: "desc" } }),
    prisma.inscription.findMany({
      orderBy: { createdAt: "desc" },
      take: 20,
      include: { adherent: true, competition: true },
    }),
    prisma.commande.findMany({
      orderBy: { createdAt: "desc" },
      take: 20,
      include: { _count: { select: { lignes: true } } },
    }),
    Promise.all([
      prisma.adherent.count(),
      prisma.competition.count(),
      prisma.inscription.count(),
      prisma.commande.count(),
      prisma.produit.count(),
    ]).then(([adherents, competitions, insc, cmd, produits]) => ({
      adherents,
      competitions,
      insc,
      cmd,
      produits,
    })),
  ]);

  return (
    <div className="mx-auto max-w-6xl px-4 py-14">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="texte-or text-4xl font-extrabold uppercase tracking-widest">
            Administration
          </h1>
          <p className="mt-2 text-sm text-or-champagne/80">
            Connecté en tant que {session.nom} ({session.email})
          </p>
        </div>
        <form action={logout}>
          <button type="submit" className="bouton-contour rounded-lg px-4 py-2 text-xs">
            Se déconnecter
          </button>
        </form>
      </div>

      {sp.annonce === "1" && (
        <p className="mt-6 rounded-lg border border-or-champagne/60 bg-or-champagne/10 px-4 py-3 text-sm text-eclat">
          Annonce publiée.
        </p>
      )}
      {sp.supprime === "1" && (
        <p className="mt-6 rounded-lg border border-or-champagne/60 bg-or-champagne/10 px-4 py-3 text-sm text-eclat">
          Annonce supprimée.
        </p>
      )}
      {sp.erreur === "champs" && (
        <p className="mt-6 rounded-lg border border-red-900/70 bg-red-950/40 px-4 py-3 text-sm text-red-200">
          Titre et contenu sont obligatoires.
        </p>
      )}

      <section className="mt-10 grid grid-cols-2 gap-4 sm:grid-cols-5">
        {[
          { label: "Adhérents", valeur: stats.adherents },
          { label: "Compétitions", valeur: stats.competitions },
          { label: "Inscriptions", valeur: stats.insc },
          { label: "Commandes", valeur: stats.cmd },
          { label: "Produits", valeur: stats.produits },
        ].map((s) => (
          <div key={s.label} className="carte-satin rounded-xl p-4 text-center">
            <p className="texte-or text-3xl font-extrabold">{s.valeur}</p>
            <p className="mt-1 text-xs uppercase tracking-widest text-or-champagne/80">{s.label}</p>
          </div>
        ))}
      </section>

      <section className="mt-12">
        <h2 className="texte-or text-2xl font-extrabold uppercase tracking-widest">Annonces</h2>
        <form action={creerAnnonce} className="carte-satin mt-4 grid gap-3 rounded-xl p-5">
          <input required name="titre" placeholder="Titre de l'annonce" className="champ rounded-lg px-3 py-2 text-sm" />
          <textarea
            required
            name="contenu"
            placeholder="Contenu de l'annonce"
            rows={3}
            className="champ rounded-lg px-3 py-2 text-sm"
          />
          <button type="submit" className="bouton-or w-fit rounded-lg px-4 py-2 text-xs">
            Publier l'annonce
          </button>
        </form>

        <div className="mt-6 grid gap-4">
          {annonces.map((a) => (
            <div key={a.id} className="carte-satin flex flex-wrap items-center justify-between gap-3 rounded-xl p-4">
              <div>
                <p className="font-bold text-eclat">
                  {a.titre}{" "}
                  <span className={`ml-2 rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-widest ${a.publiee ? "border-or-champagne/50 text-or-satin" : "border-reflet-satin/60 text-reflet-satin"}`}>
                    {a.publiee ? "Publiée" : "Masquée"}
                  </span>
                </p>
                <p className="mt-1 text-sm text-or-champagne/90">{a.contenu}</p>
                <p className="mt-1 text-xs text-reflet-satin">{formatDate(a.createdAt)}</p>
              </div>
              <div className="flex gap-2">
                <form action={basculerPublication}>
                  <input type="hidden" name="id" value={a.id} />
                  <button type="submit" className="bouton-contour rounded-lg px-3 py-1.5 text-[11px]">
                    {a.publiee ? "Masquer" : "Publier"}
                  </button>
                </form>
                <form action={supprimerAnnonce}>
                  <input type="hidden" name="id" value={a.id} />
                  <button type="submit" className="rounded-lg border border-red-900/70 px-3 py-1.5 text-[11px] uppercase tracking-widest text-red-300 transition hover:bg-red-950/50">
                    Supprimer
                  </button>
                </form>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="mt-12">
        <h2 className="texte-or text-2xl font-extrabold uppercase tracking-widest">
          Inscriptions récentes
        </h2>
        <div className="carte-satin mt-4 overflow-x-auto rounded-xl">
          <table className="w-full min-w-[640px] text-left text-sm">
            <thead>
              <tr className="border-b border-bronze-sombre/60 text-xs uppercase tracking-widest text-or-champagne/80">
                <th className="px-4 py-3">Adhérent</th>
                <th className="px-4 py-3">Compétition</th>
                <th className="px-4 py-3">Statut</th>
                <th className="px-4 py-3">Date</th>
              </tr>
            </thead>
            <tbody>
              {inscriptions.map((i) => (
                <tr key={i.id} className="border-b border-noir-charbon/60 last:border-0">
                  <td className="px-4 py-3">
                    {i.adherent.prenom} {i.adherent.nom}
                    <span className="block text-xs text-reflet-satin">{i.adherent.email}</span>
                  </td>
                  <td className="px-4 py-3">{i.competition.nom}</td>
                  <td className="px-4 py-3">{i.statut}</td>
                  <td className="px-4 py-3 text-or-champagne/90">{formatDate(i.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="mt-12">
        <h2 className="texte-or text-2xl font-extrabold uppercase tracking-widest">
          Commandes récentes
        </h2>
        <div className="carte-satin mt-4 overflow-x-auto rounded-xl">
          <table className="w-full min-w-[640px] text-left text-sm">
            <thead>
              <tr className="border-b border-bronze-sombre/60 text-xs uppercase tracking-widest text-or-champagne/80">
                <th className="px-4 py-3">E-mail</th>
                <th className="px-4 py-3">Articles</th>
                <th className="px-4 py-3">Total</th>
                <th className="px-4 py-3">Statut</th>
                <th className="px-4 py-3">Date</th>
              </tr>
            </thead>
            <tbody>
              {commandes.map((c) => (
                <tr key={c.id} className="border-b border-noir-charbon/60 last:border-0">
                  <td className="px-4 py-3">{c.email}</td>
                  <td className="px-4 py-3">{c._count.lignes}</td>
                  <td className="px-4 py-3 font-bold text-eclat">{formatPrix(c.total)}</td>
                  <td className="px-4 py-3">{c.statut}</td>
                  <td className="px-4 py-3 text-or-champagne/90">{formatDate(c.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <p className="mt-12 text-center text-xs text-reflet-satin">
        Base de données consultable en local/distant avec <code className="text-or-champagne">npm run db:studio</code> (Prisma Studio).
      </p>
    </div>
  );
}
