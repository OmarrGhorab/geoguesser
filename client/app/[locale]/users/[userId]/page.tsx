import { getPublicProfile, getGameHistory } from "@/lib/api/profile";
import { ApiError } from "@/lib/api/errors";
import { PublicStats } from "@/features/profile/public-stats";
import { GameHistory } from "@/features/profile/game-history";
import { ProfileNotFoundPanel, ProfileUnexpectedErrorPanel } from "@/features/profile/profile-states";

type UserProfilePageProps = {
  params: Promise<{ locale: string; userId: string }>;
  searchParams: Promise<{ cursor?: string }>;
};

export default async function UserProfilePage({ params, searchParams }: UserProfilePageProps) {
  const { locale, userId } = await params;
  const { cursor } = await searchParams;

  let data;
  try {
    data = await getPublicProfile(userId);
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 400)) {
      return <ProfileNotFoundPanel />;
    }
    if (error instanceof ApiError) {
      return <ProfileUnexpectedErrorPanel />;
    }
    throw error;
  }

  let history;
  try {
    history = await getGameHistory(userId, { cursor });
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 400)) {
      return <ProfileNotFoundPanel />;
    }
    if (error instanceof ApiError) {
      return <ProfileUnexpectedErrorPanel />;
    }
    throw error;
  }

  return (
    <>
      <PublicStats profile={data.profile} stats={data.stats} />
      <div className="mx-auto w-full max-w-2xl px-4 pb-8">
        <GameHistory games={history.games} page={history.page} basePath={`/${locale}/users/${userId}`} />
      </div>
    </>
  );
}
