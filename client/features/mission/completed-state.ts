type AttemptState = Readonly<{
  status: string;
  game_id?: string | null;
}> | null;

export function resolveCompletedDailyGameId(
  selectedDate: string,
  missionDate: string,
  attempt: AttemptState,
  lastCompletedGameId?: string | null,
): string | null {
  if (selectedDate !== missionDate) {
    return null;
  }
  return (
    lastCompletedGameId ??
    (attempt?.status === "completed" ? attempt.game_id : null) ??
    null
  );
}

export function shouldShowPastDailyEmptyState(
  selectedDate: string,
  todayDate: string,
  completedGameId: string | null,
): boolean {
  return selectedDate < todayDate && completedGameId === null;
}
