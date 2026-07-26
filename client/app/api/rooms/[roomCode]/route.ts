import { NextResponse } from "next/server";
import { z } from "zod";
import { apiJson } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";
import { roomSnapshotSchema } from "@/features/rooms/schemas";

const paramsSchema = z.object({
  roomCode: z
    .string()
    .min(4)
    .max(12)
    .regex(/^[A-Za-z0-9]+$/),
});

type RouteContext = { params: Promise<{ roomCode: string }> };

/** Same-origin room snapshot for client polling (cookie session). */
export async function GET(_request: Request, context: RouteContext) {
  const params = paramsSchema.safeParse(await context.params);
  if (!params.success) {
    return NextResponse.json(
      { error: { code: "validation_failed", message: "Invalid room code." } },
      { status: 400, headers: { "Cache-Control": "no-store" } },
    );
  }

  try {
    const snapshot = await apiJson(
      `/rooms/${encodeURIComponent(params.data.roomCode.toUpperCase())}`,
      roomSnapshotSchema,
      { requiresAuth: true },
    );
    return NextResponse.json(snapshot, {
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
      { error: { code: "unavailable", message: "Room unavailable." } },
      { status: 503, headers: { "Cache-Control": "no-store" } },
    );
  }
}
