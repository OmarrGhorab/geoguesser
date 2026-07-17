import "server-only";

import { apiJson } from "@/lib/api/client";
import { missionsResponseSchema, type Mission } from "./schemas";

export async function getMissions(): Promise<Mission[]> {
  return (
    await apiJson("/missions", missionsResponseSchema, { requiresAuth: true })
  ).missions;
}
