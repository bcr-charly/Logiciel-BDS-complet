"use server";

import { redirect } from "next/navigation";
import { prisma } from "@/lib/db";

export async function commander(formData: FormData) {
  const produitId = String(formData.get("produitId") ?? "");
  const email = String(formData.get("email") ?? "").trim().toLowerCase();
  const prenom = String(formData.get("prenom") ?? "").trim();
  const nom = String(formData.get("nom") ?? "").trim();
  const quantiteBrute = Number.parseInt(String(formData.get("quantite") ?? "1"), 10);
  const quantite = Number.isFinite(quantiteBrute) ? Math.max(1, quantiteBrute) : 1;

  if (!produitId || !email.includes("@") || !prenom || !nom) {
    redirect("/boutique?erreur=champs");
  }

  const produit = await prisma.produit.findUnique({ where: { id: produitId } });
  if (!produit || !produit.actif) {
    redirect("/boutique?erreur=produit");
  }
  if (produit.stock < quantite) {
    redirect("/boutique?erreur=stock");
  }

  const adherent = await prisma.adherent.upsert({
    where: { email },
    update: { prenom, nom },
    create: { prenom, nom, email },
  });

  await prisma.$transaction([
    prisma.produit.update({
      where: { id: produitId },
      data: { stock: { decrement: quantite } },
    }),
    prisma.commande.create({
      data: {
        adherentId: adherent.id,
        email,
        total: produit.prix * quantite,
        lignes: {
          create: { produitId, quantite, prixUnitaire: produit.prix },
        },
      },
    }),
  ]);

  redirect("/boutique?commande=1");
}
