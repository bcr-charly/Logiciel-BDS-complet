import { prisma } from "@/lib/db";
import { formatPrix } from "@/lib/format";
import { commander } from "@/actions/boutique";

export const dynamic = "force-dynamic";

type SearchParams = Promise<Record<string, string | string[] | undefined>>;

const messages: Record<string, { texte: string; ok: boolean }> = {
  commande: { texte: "Commande enregistrée ! Le BDS te recontacte pour le retrait et le paiement.", ok: true },
  erreur_champs: { texte: "Merci de remplir tous les champs obligatoires.", ok: false },
  erreur_produit: { texte: "Ce produit n'est plus disponible.", ok: false },
  erreur_stock: { texte: "Stock insuffisant pour cette quantité.", ok: false },
};

export default async function BoutiquePage({ searchParams }: { searchParams: SearchParams }) {
  const sp = await searchParams;
  const cle = sp.commande ? "commande" : `erreur_${String(sp.erreur ?? "")}`;
  const message = messages[cle];

  const produits = await prisma.produit.findMany({
    where: { actif: true },
    orderBy: { createdAt: "asc" },
  });

  return (
    <div className="mx-auto max-w-6xl px-4 py-14">
      <h1 className="texte-or text-4xl font-extrabold uppercase tracking-widest">Boutique</h1>
      <p className="mt-3 max-w-2xl text-or-satin/90">
        Les produits officiels du BDS, aux couleurs du maillot : noir satin et or champagne.
        Passe commande ici, le paiement et le retrait se font auprès du BDS.
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

      <div className="mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {produits.map((p) => (
          <article key={p.id} className="carte-satin flex flex-col rounded-xl p-6">
            <div className="flex h-28 items-center justify-center rounded-lg bg-noir-profond/80 text-4xl">
              <span className="texte-or font-extrabold tracking-widest">BDS</span>
            </div>
            <h2 className="texte-or mt-4 text-xl font-bold">{p.nom}</h2>
            {p.description && (
              <p className="mt-2 flex-1 text-sm leading-relaxed text-or-champagne/90">{p.description}</p>
            )}
            <div className="mt-4 flex items-center justify-between">
              <span className="text-lg font-extrabold text-eclat">{formatPrix(p.prix)}</span>
              <span className={`text-xs uppercase tracking-widest ${p.stock > 0 ? "text-or-satin" : "text-red-300"}`}>
                {p.stock > 0 ? `En stock : ${p.stock}` : "Rupture de stock"}
              </span>
            </div>

            <form action={commander} className="mt-4 grid gap-2 border-t border-bronze-sombre/50 pt-4">
              <input type="hidden" name="produitId" value={p.id} />
              <div className="grid grid-cols-2 gap-2">
                <input required name="prenom" placeholder="Prénom" className="champ rounded-lg px-3 py-2 text-sm" />
                <input required name="nom" placeholder="Nom" className="champ rounded-lg px-3 py-2 text-sm" />
              </div>
              <input required type="email" name="email" placeholder="E-mail" className="champ rounded-lg px-3 py-2 text-sm" />
              <div className="flex items-center gap-2">
                <input
                  required
                  type="number"
                  name="quantite"
                  min={1}
                  max={Math.max(1, p.stock)}
                  defaultValue={1}
                  className="champ w-24 rounded-lg px-3 py-2 text-sm"
                />
                <button
                  type="submit"
                  disabled={p.stock === 0}
                  className="bouton-or flex-1 rounded-lg px-4 py-2 text-xs"
                >
                  {p.stock === 0 ? "Indisponible" : "Commander"}
                </button>
              </div>
            </form>
          </article>
        ))}
      </div>
    </div>
  );
}
