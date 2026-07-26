import { notFound } from "next/navigation";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { ApiError } from "@/lib/api/errors";
import type { AppLocale } from "@/lib/i18n/routing";
import { getRoom } from "@/features/rooms/data";
import { RoomLobbyScreen } from "@/features/rooms/components/room-lobby-screen";

type RoomPageProps = Readonly<{
  params: Promise<{ locale: string; roomCode: string }>;
}>;

export default async function RoomPage({ params }: RoomPageProps) {
  const { locale, roomCode } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const code = roomCode.trim().toUpperCase();
  if (!/^[A-Z0-9]{4,12}$/.test(code)) notFound();

  let snapshot;
  try {
    snapshot = await getRoom(code);
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 403)) {
      notFound();
    }
    throw error;
  }

  const t = await getTranslations("Rooms");

  return (
    <RoomLobbyScreen
      locale={appLocale}
      room={snapshot.room}
      googleMapsApiKey={process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY ?? ""}
      copy={{
        title: t("lobby.title"),
        codeLabel: t("lobby.codeLabel"),
        statusLabel: t("lobby.statusLabel"),
        playersLabel: t("lobby.playersLabel"),
        host: t("lobby.host"),
        you: t("lobby.you"),
        leave: t("lobby.leave"),
        cancel: t("lobby.cancel"),
        start: t("lobby.start"),
        starting: t("lobby.starting"),
        needPlayers: t("lobby.needPlayers"),
        back: t("back"),
        copyCode: t("lobby.copyCode"),
        copied: t("lobby.copied"),
        waiting: t("lobby.waiting"),
        activeHint: t("lobby.activeHint"),
        submitted: t("lobby.submitted"),
        waitingOthers: t("lobby.waitingOthers"),
        placePin: t("lobby.placePin"),
        submitGuess: t("lobby.submitGuess"),
        submitting: t("lobby.submitting"),
        next: t("lobby.next"),
        distance: t("lobby.distance"),
        correctLocation: t("lobby.correctLocation"),
        timeExpired: t("lobby.timeExpired"),
        mapLabel: t("lobby.mapLabel"),
        total: t("lobby.total"),
        zoomIn: t("lobby.zoomIn"),
        zoomOut: t("lobby.zoomOut"),
        recenter: t("lobby.recenter"),
        expandMap: t("lobby.expandMap"),
        mapUnavailable: t("lobby.mapUnavailable"),
        retryMap: t("lobby.retryMap"),
        selectLocation: t("lobby.selectLocation"),
        live: t("lobby.live"),
        polling: t("lobby.polling"),
        errors: {
          hostActionRequired: t("errors.hostActionRequired"),
          unavailable: t("errors.unavailable"),
          needPlayers: t("lobby.needPlayers"),
          invalidGuess: t("lobby.invalidGuess"),
          generic: t("errors.generic"),
        },
      }}
    />
  );
}
