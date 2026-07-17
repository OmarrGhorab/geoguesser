export const COMPASS_TICK_DEGREES = 11.25;
export const COMPASS_TICK_WIDTH_PX = 14;
export const COMPASS_CENTER_TICK = 32;

const CARDINAL_LABELS = new Map<number, string>([
  [0, "N"],
  [4, "NE"],
  [8, "E"],
  [12, "SE"],
  [16, "S"],
  [20, "SW"],
  [24, "W"],
  [28, "NW"],
]);

export function normalizedHeading(heading: number): number {
  return ((heading % 360) + 360) % 360;
}

export function compassTrackOffset(heading: number): number {
  const centeredTick =
    COMPASS_CENTER_TICK + normalizedHeading(heading) / COMPASS_TICK_DEGREES;
  return -(centeredTick * COMPASS_TICK_WIDTH_PX + COMPASS_TICK_WIDTH_PX / 2);
}

export function compassTickLabel(index: number): string | null {
  const cycleIndex = (((index - COMPASS_CENTER_TICK) % 32) + 32) % 32;
  return CARDINAL_LABELS.get(cycleIndex) ?? null;
}
