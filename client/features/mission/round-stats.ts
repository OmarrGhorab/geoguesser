export function formatRoundTimer(seconds: number): string {
  const safeSeconds = Math.max(0, Math.floor(seconds));
  return `${Math.floor(safeSeconds / 60)}:${String(safeSeconds % 60).padStart(2, "0")}`;
}

export function roundStatValue(
  roundNumber: number,
  currentRoundNumber: number,
  timerLabel: string,
  completedRoundLabels: Readonly<Record<number, string>>,
): string {
  const completedLabel = completedRoundLabels[roundNumber];
  if (completedLabel) return completedLabel;
  if (roundNumber === currentRoundNumber) return timerLabel;
  if (roundNumber < currentRoundNumber) return "✓";
  return "";
}
