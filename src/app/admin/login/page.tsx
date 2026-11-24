import { login } from "@/actions/auth";

export const dynamic = "force-dynamic";

type SearchParams = Promise<Record<string, string | string[] | undefined>>;

export default async function AdminLoginPage({ searchParams }: { searchParams: SearchParams }) {
  const sp = await searchParams;
  const erreur = sp.erreur === "1";

  return (
    <div className="mx-auto max-w-md px-4 py-20">
      <div className="carte-satin rounded-xl p-8">
        <p className="badge-or inline-block rounded-full px-3 py-1 text-[11px]">Espace admin</p>
        <h1 className="texte-or mt-4 text-3xl font-extrabold uppercase tracking-widest">
          Connexion
        </h1>
        <p className="mt-2 text-sm text-or-champagne/80">
          Accès réservé aux administrateurs du BDS (site et base de données).
        </p>

        {erreur && (
          <p className="mt-4 rounded-lg border border-red-900/70 bg-red-950/40 px-4 py-3 text-sm text-red-200">
            E-mail ou mot de passe incorrect.
          </p>
        )}

        <form action={login} className="mt-6 grid gap-4">
          <input
            required
            type="email"
            name="email"
            placeholder="E-mail admin"
            className="champ rounded-lg px-3 py-2 text-sm"
          />
          <input
            required
            type="password"
            name="password"
            placeholder="Mot de passe"
            className="champ rounded-lg px-3 py-2 text-sm"
          />
          <button type="submit" className="bouton-or rounded-lg px-4 py-3 text-sm">
            Se connecter
          </button>
        </form>
      </div>
    </div>
  );
}
