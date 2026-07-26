import { NextResponse } from "next/server";
import { z } from "zod";
import { proxyMutation } from "@/lib/api/mutation-proxy";

const bodySchema = z.object({
  channel_kind: z.enum(["match", "party"]),
  channel_id: z.string().uuid(),
});

const ticketResponseSchema = z.object({
  ticket: z.string().min(1),
  expires_at: z.string().min(1),
  websocket_url: z.string().min(1),
});

export async function POST(request: Request) {
  let json: unknown;
  try {
    json = await request.json();
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "validation_failed",
          message: "Invalid JSON body.",
        },
      },
      { status: 400, headers: { "Cache-Control": "no-store" } },
    );
  }

  const body = bodySchema.safeParse(json);
  if (!body.success) {
    return NextResponse.json(
      {
        error: {
          code: "validation_failed",
          message: "channel_kind and channel_id are required.",
        },
      },
      { status: 400, headers: { "Cache-Control": "no-store" } },
    );
  }

  return proxyMutation(request, {
    path: "/realtime/tickets",
    method: "POST",
    body: body.data,
    schema: ticketResponseSchema,
  });
}
