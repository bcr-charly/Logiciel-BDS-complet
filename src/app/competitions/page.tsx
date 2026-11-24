import { prisma } from "@/lib/db";
import { formatDate, formatPrix } from "@/lib/format";
import { inscrire } from "@/actions/competitions";

export const dynamic = "force-dynamic";

type SearchParams = Promise<Record<string, string | string[] | undefined>>;

const messages: Record<string, { texte: string; ok: boolean }> = {
  inscrit: { texte: "Inscription enregistrée. À très vite sur le terrain !", ok: true },
  erreur_champs: { texte: "Merci de remplir tous les champs obligatoires.", ok: false },
  erreur_cloturee: { texte: "Cette compétition est clôturée.", ok: false },
  erreur_complet: { texte: "Plus de places disponibles pour cette compétition.", ok: false },
  erreur_deja: { texte: "Tu es déjà inscrit à cette compétition.", ok: false },
};

export default async function CompetitionsPage({ searchParams }: { searchParams: SearchParams }) {
  const sp = await searchParams;
  const cle = sp.inscrit ? "inscrit" : `erreur_${String(sp.erreur ?? "")}`;
  const message = messages[cle];

  const competitions = await prisma.competition.findMany({
    orderBy: { date: "asc" },
    include: { _count: { select: { inscriptions: true } } },
  });

  return (
    <div className="mx-auto max-w-6xl px-4 py-14">
      <h1 className="texte-or text-4xl font-extrabold uppercase tracking-widest">Compétitions</h1>
      <p className="mt-3 max-w-2xl text-or-satin/90">
        Retrouve ici toutes les compétitions organisées par le BDS. Inscris-toi en renseignant
        ton e-mail ENSIIE — ton profil adhérent est créé automatiquement si besoin.
      </p>

      {message && (
        <p
          className={`mt-6 rounded-lg border px-4 py-3 text-sm ${
            message.ok
              ? "border-or-champagne/60 bg-or-champagne/10 text-eclat"
              : "border-red-900/70 bg-red-950/40 text-red-200"
          }`}
        >
          {message.texte}
        </p>
      )}

      <div className="mt-10 grid gap-6 lg:grid-cols-2">
        {competitions.map((c) => {
          const restantes =
            c.places == null ? null : Math.max(0, c.places - c._count.inscriptions);
          const complet = restantes === 0;
          const desactive = c.cloturee || complet;
          return (
            <article key={c.id} className="carte-satin flex flex-col rounded-xl p-6">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="badge-or rounded-full px-3 py-1 text-[11px]">{c.sport}</span>
                {c.cloturee ? (
                  <span className="rounded-full border border-red-900/70 px-3 py-1 text-[11px] uppercase tracking-widest text-red-300">
                    Clôturée
                  </span>
                ) : (
                  <span className="rounded-full border border-or-champagne/50 px-3 py-1 text-[11px] uppercase tracking-widest text-or-satin">
                    {c.tarif === 0 ? "Gratuit" : formatPrix(c.tarif)}
                  </span>
                )}
              </div>
              <h2 className="texte-or mt-4 text-2xl font-bold">{c.nom}</h2>
              <p className="mt-2 text-sm text-or-satin/90">
                {formatDate(c.date)} — {c.lieu}
              </p>
              {c.description && (
                <p className="mt-3 text-sm leading-relaxed text-or-champagne/90">{c.description}</p>
              )}
              <p className="mt-2 text-xs uppercase tracking-widest text-reflet-satin">
                {c.places == null
                  ? `${c._count.inscriptions} inscrit(s) — places illimitées`
                  : restantes != null
                    ? `${restantes} place(s) restante(s) sur ${c.places}`
                    : ""}
              </p>

              <form action={inscrire} className="mt-5 grid gap-3 border-t border-bronze-sombre/50 pt-5 sm:grid-cols-2">
                <input type="hidden" name="competitionId" value={c.id} />
                <input required name="prenom" placeholder="Prénom" className="champ rounded-lg px-3 py-2 text-sm" />
                <input required name="nom" placeholder="Nom" className="champ rounded-lg px-3 py-2 text-sm" />
                <input required type="email" name="email" placeholder="E-mail ENSIIE" className="champ rounded-lg px-3 py-2 text-sm sm:col-span-2" />
                <input name="telephone" type="tel" placeholder="Téléphone (facultatif)" className="champ rounded-lg px-3 py-2 text-sm sm:col-span-2" />
                <button
                  type="submit"
                  disabled={desactive}
                  className="bouton-or rounded-lg px-4 py-2 text-xs sm:col-span-2"
                >
                  {c.cloturee ? "Inscriptions clôturées" : complet ? "Complet" : "S'inscrire"}
                </button>
              </form>
            </article>
          );
        })}
      </div>
    </div>
  );
}
