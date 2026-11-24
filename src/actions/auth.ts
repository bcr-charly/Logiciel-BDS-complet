"use server";

import bcrypt from "bcryptjs";
import { redirect } from "next/navigation";
import { prisma } from "@/lib/db";
import { createSession, destroySession } from "@/lib/auth";

export async function login(formData: FormData) {
  const email = String(formData.get("email") ?? "").trim().toLowerCase();
  const password = String(formData.get("password") ?? "");

  const admin = await prisma.admin.findUnique({ where: { email } });
  if (!admin || !(await bcrypt.compare(password, admin.password))) {
    redirect("/admin/login?erreur=1");
  }

  await createSession({ sub: admin.id, email: admin.email, nom: admin.nom });
  redirect("/admin");
}

export async function logout() {
  await destroySession();
  redirect("/admin/login");
}
