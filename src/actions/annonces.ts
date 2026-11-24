"use server";

import { redirect } from "next/navigation";
import { revalidatePath } from "next/cache";
import { prisma } from "@/lib/db";
import { getSession } from "@/lib/auth";

async function exigerAdmin() {
  const session = await getSession();
  if (!session) redirect("/admin/login");
  return session;
}

export async function creerAnnonce(formData: FormData) {
  await exigerAdmin();
  const titre = String(formData.get("titre") ?? "").trim();
  const contenu = String(formData.get("contenu") ?? "").trim();
  if (!titre || !contenu) redirect("/admin?erreur=champs");
  await prisma.annonce.create({ data: { titre, contenu } });
  revalidatePath("/");
  revalidatePath("/admin");
  redirect("/admin?annonce=1");
}

export async function supprimerAnnonce(formData: FormData) {
  await exigerAdmin();
  const id = String(formData.get("id") ?? "");
  if (id) await prisma.annonce.delete({ where: { id } });
  revalidatePath("/");
  revalidatePath("/admin");
  redirect("/admin?supprime=1");
}

export async function basculerPublication(formData: FormData) {
  await exigerAdmin();
  const id = String(formData.get("id") ?? "");
  const annonce = await prisma.annonce.findUnique({ where: { id } });
  if (annonce) {
    await prisma.annonce.update({ where: { id }, data: { publiee: !annonce.publiee } });
  }
  revalidatePath("/");
  revalidatePath("/admin");
  redirect("/admin");
}
