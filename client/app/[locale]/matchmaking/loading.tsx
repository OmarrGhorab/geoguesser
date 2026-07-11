import { MatchmakingLoadingSkeleton } from "@/features/matchmaking/matchmaking-states";

export default function MatchmakingLoading() {
  return (
    <main className="px-4 py-8">
      <MatchmakingLoadingSkeleton />
    </main>
  );
}
