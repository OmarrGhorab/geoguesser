import { getTranslations } from "next-intl/server";
import { getRoom } from "@/lib/api/rooms";
import { Lobby } from "@/features/rooms/lobby";
import { HostControlsServer } from "@/features/rooms/host-controls-server";
import { RoomGame } from "@/features/rooms/room-game";

type RoomPageProps = {
  params: Promise<{
    roomCode: string;
  }>;
};

export default async function RoomPage({ params }: RoomPageProps) {
  const { roomCode } = await params;
  const [t, response] = await Promise.all([getTranslations("Rooms"), getRoom(roomCode)]);

  const hostControlLabels = {
    title: t("hostControls.title"),
    roundCount: t("hostControls.roundCount"),
    maxPlayers: t("hostControls.maxPlayers"),
    timerSeconds: t("hostControls.timerSeconds"),
    saveSettings: t("hostControls.saveSettings"),
    startRoom: t("hostControls.startRoom"),
    removePlayer: t("hostControls.removePlayer"),
    ready: t("hostControls.ready"),
    notReady: t("hostControls.notReady"),
    locked: t("hostControls.locked"),
  };

  if (response.room.status === "active" || response.room.status === "completed") {
    return (
      <RoomGame
        room={response.room}
        labels={{
          activeTitle: t("game.activeTitle"),
          round: t("game.round"),
          countdown: t("game.countdown"),
          untimed: t("game.untimed"),
          progress: t("game.progress"),
          submitted: t("game.submitted"),
          results: t("game.results"),
          score: t("game.score"),
          final: t("game.final"),
          waiting: t("game.waiting"),
          recovery: {
            reconnecting: t("recovery.reconnecting"),
            degraded: t("recovery.degraded"),
            disconnected: t("recovery.disconnected"),
            restored: t("recovery.restored"),
          },
        }}
      />
    );
  }

  return (
    <Lobby
      room={response.room}
      hostControls={<HostControlsServer room={response.room} currentPlayerId={response.room.current_player_id} labels={hostControlLabels} />}
      labels={{
        roomCode: t("roomCode"),
        copyInvite: t("copyInvite"),
        players: t("players"),
        host: t("host"),
        connected: t("connected"),
        disconnected: t("disconnected"),
        rounds: t("rounds"),
        timer: t("timer"),
        noTimer: t("noTimer"),
        hostControls: hostControlLabels,
      }}
    />
  );
}
