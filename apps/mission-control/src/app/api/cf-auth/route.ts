import { NextRequest, NextResponse } from "next/server";

// Server-side API route: reads Cf-Access-Authenticated-User-Email header
// from Cloudflare Zero Trust and exchanges it for JWT tokens via the backend.
// The backend's /auth/proxy-login trusts this header (only reachable via cloudflared).

export async function POST(req: NextRequest) {
  const cfEmail = req.headers.get("cf-access-authenticated-user-email");
  if (!cfEmail) {
    return NextResponse.json({ error: "not behind cloudflare access" }, { status: 401 });
  }

  // Call backend internally (server-side, inside cluster)
  const internalApi =
    process.env.API_INTERNAL_URL ||
    process.env.NEXT_PUBLIC_API_URL ||
    "http://localhost:8080";

  try {
    const res = await fetch(`${internalApi}/api/v1/auth/proxy-login`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Cf-Access-Authenticated-User-Email": cfEmail,
      },
    });

    if (!res.ok) {
      const body = await res.text();
      return NextResponse.json({ error: body }, { status: res.status });
    }

    const tokens = await res.json();
    return NextResponse.json(tokens);
  } catch {
    return NextResponse.json({ error: "backend unreachable" }, { status: 502 });
  }
}
