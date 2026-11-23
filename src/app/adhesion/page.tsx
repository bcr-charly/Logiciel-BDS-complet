import { creerAdherent } from "@/actions/adherents";

export const dynamic = "force-dynamic";

type SearchParams = Promise<Record<string, string | string[] | undefined>>;

const messages: Record<string, { texte: string; ok: boolean }> = {
  ok: { texte: "Bienvenue au BDS ! Ton profil adhérent a été créé.", ok: true },
  existe: { texte: "Un profil existe déjà avec cet e-mail.", ok: false },
  erreur_champs: { texte: "Merci de renseigner prénom, nom et un e-mail valide.", ok: false },
};

export default async function AdhesionPage({ searchParams }: { searchParams: SearchParams }) {
  const sp = await searchParams;
  const cle = sp.ok ? "ok" : sp.existe ? "existe" : `erreur_${String(sp.erreur ?? "")}`;
  const message = messages[cle];

  return (
    <div className="mx-auto max-w-2xl px-4 py-14">
      <h1 className="texte-or text-4xl font-extrabold uppercase tracking-widest">Adhésion</h1>
      <p className="mt-3 text-or-satin/90">
        Crée ton profil adhérent du BDS de l'ENSIIE. Quelques informations simples suffisent —
        rien de plus.
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

      <form action={creerAdherent} className="carte-satin mt-8 grid gap-4 rounded-xl p-6 sm:grid-cols-2">
        <input required name="prenom" placeholder="Prénom" className="champ rounded-lg px-3 py-2 text-sm" />
        <input required name="nom" placeholder="Nom" className="champ rounded-lg px-3 py-2 text-sm" />
        <input required type="email" name="email" placeholder="E-mail ENSIIE" className="champ rounded-lg px-3 py-2 text-sm sm:col-span-2" />
        <input name="telephone" type="tel" placeholder="Téléphone (facultatif)" className="champ rounded-lg px-3 py-2 text-sm" />
        <input name="annee" placeholder="Année (ex : 1A, 2A, 3A)" className="champ rounded-lg px-3 py-2 text-sm" />
        <input name="sportPrefere" placeholder="Sport préféré (facultatif)" className="champ rounded-lg px-3 py-2 text-sm sm:col-span-2" />
        <button type="submit" className="bouton-or rounded-lg px-4 py-3 text-sm sm:col-span-2">
          Devenir adhérent
        </button>
      </form>
    </div>
  );
}
