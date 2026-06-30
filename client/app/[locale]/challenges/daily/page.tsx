import { DailyPanel } from "@/features/challenges/daily-panel";
import { ChallengeUnavailablePanel } from "@/features/challenges/unavailable-panel";
import { getDailyChallenge, startDailyChallenge } from "@/lib/api/challenges";
import { ApiError } from "@/lib/api/errors";
import { redirect } from "@/lib/i18n/navigation";

type DailyPageProps = {
  params: Promise<{ locale: string }>;
};

export default async function DailyChallengePage({ params }: DailyPageProps) {
  const { locale } = await params;
  let data;
  try {
    data = await getDailyChallenge();
  } catch (error) {
    if (error instanceof ApiError) {
      return <ChallengeUnavailablePanel />;
    }
    throw error;
  }

  async function startAction() {
    "use server";
    const response = await startDailyChallenge();
    if (response.game?.id) {
      redirect({ href: `/game/${response.game.id}`, locale });
    }
  }

  return <DailyPanel data={data} startAction={startAction} />;
}
