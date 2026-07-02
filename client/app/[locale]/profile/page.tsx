import { getCurrentProfile, getGameHistory } from "@/lib/api/profile";
import { ApiError } from "@/lib/api/errors";
import { ProfileSummary } from "@/features/profile/profile-summary";
import { GameHistory } from "@/features/profile/game-history";
import { ProfileUnauthorizedPanel, ProfileUnexpectedErrorPanel } from "@/features/profile/profile-states";

type ProfilePageProps = {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ cursor?: string }>;
};

export default async function ProfilePage({ params, searchParams }: ProfilePageProps) {
  const { locale } = await params;
  const { cursor } = await searchParams;

  let data;
  try {
    data = await getCurrentProfile();
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
      return <ProfileUnauthorizedPanel />;
    }
    if (error instanceof ApiError) {
      return <ProfileUnexpectedErrorPanel />;
    }
    throw error;
  }

  const progress = cursor ? await getGameHistory(data.profile.user_id, { cursor }) : { games: data.progress.recent_games, page: data.progress.page };

  return (
    <>
      <ProfileSummary profile={data.profile} stats={data.stats} />
      <div className="mx-auto w-full max-w-2xl px-4 pb-8">
        <GameHistory games={progress.games} page={progress.page} basePath={`/${locale}/profile`} />
      </div>
    </>
  );
}
