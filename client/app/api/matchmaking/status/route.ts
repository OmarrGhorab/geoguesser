import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import { env } from "@/lib/env";

export async function GET() {
  const cookieStore = await cookies();
  const cookieHeader = cookieStore.toString();

  try {
    const response = await fetch(`${env.BACKEND_API_URL}/matchmaking/status`, {
      method: "GET",
      headers: cookieHeader ? { Cookie: cookieHeader } : undefined,
      cache: "no-store",
    });

    const retryAfter = response.headers.get("Retry-After");
    const contentType = response.headers.get("Content-Type") || "application/json";

    if (response.status === 204) {
      return new NextResponse(null, { status: 204 });
    }

    const body = await response.text();
    const headers = new Headers({ "Content-Type": contentType });
    if (retryAfter) {
      headers.set("Retry-After", retryAfter);
    }

    // Forward status and payload without leaking backend-only headers.
    return new NextResponse(body, {
      status: response.status,
      headers,
    });
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "matchmaking_unavailable",
          message: "Matchmaking is temporarily unavailable.",
        },
      },
      { status: 503 },
    );
  }
}
