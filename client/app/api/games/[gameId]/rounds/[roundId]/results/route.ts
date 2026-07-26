import { NextResponse } from "next/server";
import { z } from "zod";
import { apiJson } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";

const paramsSchema = z.object({
  gameId: z.string().uuid(),
  roundId: z.string().uuid(),
});

/** Shared multiplayer round results — participant-only after reveal. */
const sharedRoundResultsSchema = z
  .object({
    round_id: z.string().uuid().optional(),
    round_number: z.number().int().optional(),
    actual_location: z
      .object({
        latitude: z.number(),
        longitude: z.number(),
        country_code: z.string().optional(),
      })
      .passthrough(),
    guesses: z
      .array(
        z
          .object({
            game_player_id: z.string().uuid().optional(),
            guess: z
              .object({
                latitude: z.number(),
                longitude: z.number(),
                distance_meters: z.number(),
                score: z.number(),
                timed_out: z.boolean().optional(),
              })
              .passthrough(),
          })
          .passthrough(),
      )
      .optional(),
  })
  .passthrough();

type RouteContext = {
  params: Promise<{ gameId: string; roundId: string }>;
};

export async function GET(_request: Request, context: RouteContext) {
  const params = paramsSchema.safeParse(await context.params);
  if (!params.success) {
    return NextResponse.json(
      { error: { code: "validation_failed", message: "Invalid ids." } },
      { status: 400, headers: { "Cache-Control": "no-store" } },
    );
  }

  try {
    const data = await apiJson(
      `/games/${params.data.gameId}/rounds/${params.data.roundId}/results`,
      sharedRoundResultsSchema,
      { requiresAuth: true },
    );
    return NextResponse.json(data, {
      status: 200,
      headers: { "Cache-Control": "no-store" },
    });
  } catch (error) {
    if (error instanceof ApiError) {
      return NextResponse.json(
        { error: { code: error.code, message: error.message } },
        { status: error.status, headers: { "Cache-Control": "no-store" } },
      );
    }
    return NextResponse.json(
      { error: { code: "unavailable", message: "Results unavailable." } },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }
}
