import Link from "next/link";

export default function Footer() {
  return (
    <footer className="mt-16 border-t border-bronze-sombre/60 bg-noir-charbon/60">
      <div className="ligne-or" />
      <div className="mx-auto flex max-w-6xl flex-col items-center gap-4 px-4 py-8 text-center sm:flex-row sm:justify-between sm:text-left">
        <div>
          <p className="texte-or text-lg font-extrabold tracking-widest">BDS ENSIIE</p>
          <p className="mt-1 text-sm text-or-champagne/80">
            Bureau des Sports — École nationale supérieure d'informatique pour l'industrie et
            l'entreprise
          </p>
        </div>
        <div className="flex items-center gap-5 text-xs uppercase tracking-widest">
          <Link href="/competitions" className="text-or-satin/90 transition hover:text-eclat">
            Compétitions
          </Link>
          <Link href="/boutique" className="text-or-satin/90 transition hover:text-eclat">
            Boutique
          </Link>
          <Link href="/admin" className="text-or-satin/90 transition hover:text-eclat">
            Admin
          </Link>
        </div>
      </div>
      <p className="pb-6 text-center text-xs text-reflet-satin">
        Noir satin &amp; or champagne — maillot officiel du BDS
      </p>
    </footer>
  );
}
