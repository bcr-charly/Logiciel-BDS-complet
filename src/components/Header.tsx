import Link from "next/link";

const liens = [
  { href: "/", label: "Accueil" },
  { href: "/competitions", label: "Compétitions" },
  { href: "/boutique", label: "Boutique" },
  { href: "/adhesion", label: "Adhésion" },
  { href: "/admin", label: "Admin" },
];

export default function Header() {
  return (
    <header className="sticky top-0 z-40 border-b border-bronze-sombre/60 bg-noir-profond/90 backdrop-blur">
      <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-3">
        <Link href="/" className="flex items-baseline gap-2">
          <span className="texte-or text-2xl font-extrabold tracking-widest">BDS</span>
          <span className="hidden text-xs uppercase tracking-[0.35em] text-or-champagne sm:block">
            ENSIIE
          </span>
        </Link>
        <nav className="flex flex-wrap items-center gap-x-5 gap-y-1 text-xs uppercase tracking-widest sm:text-sm">
          {liens.map((lien) => (
            <Link
              key={lien.href}
              href={lien.href}
              className="text-or-satin/90 transition hover:text-eclat"
            >
              {lien.label}
            </Link>
          ))}
        </nav>
      </div>
      <div className="ligne-or" />
    </header>
  );
}
