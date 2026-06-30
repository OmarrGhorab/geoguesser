import { SharedPanel } from "@/features/challenges/shared-panel";
import { getSharedChallenge, startChallenge } from "@/lib/api/challenges";
import { redirect } from "@/lib/i18n/navigation";

type SharedPageProps = {
  params: Promise<{ locale: string; challengeId: string }>;
};

export default async function SharedChallengePage({ params }: SharedPageProps) {
  const { locale, challengeId } = await params;
  const data = await getSharedChallenge(challengeId);

  async function startAction() {
    "use server";
    const response = await startChallenge(data.challenge.id);
    if (response.game?.id) {
      redirect({ href: `/game/${response.game.id}`, locale });
    }
  }

  return <SharedPanel data={data} startAction={startAction} />;
}
