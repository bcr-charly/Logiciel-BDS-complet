"use server";

import { redirect } from "next/navigation";
import { prisma } from "@/lib/db";

export async function creerAdherent(formData: FormData) {
  const prenom = String(formData.get("prenom") ?? "").trim();
  const nom = String(formData.get("nom") ?? "").trim();
  const email = String(formData.get("email") ?? "").trim().toLowerCase();
  const telephone = String(formData.get("telephone") ?? "").trim() || null;
  const annee = String(formData.get("annee") ?? "").trim() || null;
  const sportPrefere = String(formData.get("sportPrefere") ?? "").trim() || null;

  if (!prenom || !nom || !email.includes("@")) {
    redirect("/adhesion?erreur=champs");
  }

  const existant = await prisma.adherent.findUnique({ where: { email } });
  if (existant) {
    redirect("/adhesion?existe=1");
  }

  await prisma.adherent.create({
    data: { prenom, nom, email, telephone, annee, sportPrefere },
  });
  redirect("/adhesion?ok=1");
}
