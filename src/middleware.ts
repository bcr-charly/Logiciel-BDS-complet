import { NextRequest, NextResponse } from "next/server";
import { jwtVerify } from "jose";

const secret = new TextEncoder().encode(
  process.env.AUTH_SECRET ?? "dev-secret-bds-ensiie-local-ne-pas-utiliser-en-prod",
);

export async function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;
  if (pathname.startsWith("/admin") && pathname !== "/admin/login") {
    const token = req.cookies.get("bds_session")?.value;
    let valide = false;
    if (token) {
      try {
        await jwtVerify(token, secret);
        valide = true;
      } catch {
        valide = false;
      }
    }
    if (!valide) {
      return NextResponse.redirect(new URL("/admin/login", req.url));
    }
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/admin/:path*"],
};
