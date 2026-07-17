import { notFound, redirect } from "next/navigation";
import { z } from "zod";
import { dailyMissionPlayHref } from "@/features/mission/routes";
import type { AppLocale } from "@/lib/i18n/routing";

type LegacyGamePageProps = Readonly<{
  params: Promise<{ locale: string; gameId: string }>;
}>;

export default async function LegacyGamePage({ params }: LegacyGamePageProps) {
  const { locale, gameId } = await params;
  const parsedGameId = z.string().uuid().safeParse(gameId);
  if (!parsedGameId.success) notFound();
  redirect(dailyMissionPlayHref(locale as AppLocale, parsedGameId.data));
}
