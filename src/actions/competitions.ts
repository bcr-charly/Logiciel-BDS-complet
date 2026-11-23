"use server";

import { redirect } from "next/navigation";
import { prisma } from "@/lib/db";

export async function inscrire(formData: FormData) {
  const competitionId = String(formData.get("competitionId") ?? "");
  const prenom = String(formData.get("prenom") ?? "").trim();
  const nom = String(formData.get("nom") ?? "").trim();
  const email = String(formData.get("email") ?? "").trim().toLowerCase();
  const telephone = String(formData.get("telephone") ?? "").trim() || null;

  if (!competitionId || !prenom || !nom || !email.includes("@")) {
    redirect("/competitions?erreur=champs");
  }

  const competition = await prisma.competition.findUnique({
    where: { id: competitionId },
    include: { _count: { select: { inscriptions: true } } },
  });

  if (!competition || competition.cloturee) {
    redirect("/competitions?erreur=cloturee");
  }
  if (competition.places != null && competition._count.inscriptions >= competition.places) {
    redirect("/competitions?erreur=complet");
  }

  const adherent = await prisma.adherent.upsert({
    where: { email },
    update: { prenom, nom, telephone: telephone ?? undefined },
    create: { prenom, nom, email, telephone },
  });

  let dejaInscrit = false;
  try {
    await prisma.inscription.create({
      data: { adherentId: adherent.id, competitionId },
    });
  } catch {
    dejaInscrit = true;
  }

  if (dejaInscrit) {
    redirect("/competitions?erreur=deja");
  }
  redirect("/competitions?inscrit=1");
}
