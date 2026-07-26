import { NextResponse } from "next/server";
import { z } from "zod";
import { proxyMutation } from "@/lib/api/mutation-proxy";
import {
  guessResultSchema,
  submitGuessBodySchema,
} from "@/features/play/schemas";

const paramsSchema = z.object({
  gameId: z.string().uuid(),
  roundId: z.string().uuid(),
});

type RouteContext = {
  params: Promise<{ gameId: string; roundId: string }>;
};

export async function POST(request: Request, context: RouteContext) {
  const params = paramsSchema.safeParse(await context.params);
  if (!params.success) {
    return NextResponse.json(
      {
        error: {
          code: "validation_failed",
          message: "Invalid game or round id.",
        },
      },
      { status: 400, headers: { "Cache-Control": "no-store" } },
    );
  }

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

  const body = submitGuessBodySchema.safeParse(json);
  if (!body.success) {
    return NextResponse.json(
      {
        error: {
          code: "validation_failed",
          message: "Invalid guess payload.",
        },
      },
      { status: 400, headers: { "Cache-Control": "no-store" } },
    );
  }

  const { latitude, longitude, idempotencyKey } = body.data;
  return proxyMutation(request, {
    path: `/games/${params.data.gameId}/rounds/${params.data.roundId}/guesses`,
    method: "POST",
    body: { latitude, longitude },
    idempotencyKey,
    schema: guessResultSchema,
  });
}
