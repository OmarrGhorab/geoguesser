import type { Room } from "./types";
import { HostControls } from "./host-controls";
import { removeRoomPlayerAction, startRoomAction, updateRoomSettingsAction } from "./actions";

type HostControlsServerProps = {
  room: Room;
  currentPlayerId?: string | null;
  labels: Parameters<typeof HostControls>[0]["labels"];
};

export function HostControlsServer({ room, currentPlayerId, labels }: HostControlsServerProps) {
  async function updateSettings(formData: FormData) {
    "use server";
    await updateRoomSettingsAction(room.code, formData);
  }

  async function start() {
    "use server";
    await startRoomAction(room.code);
  }

  async function removePlayer(playerId: string) {
    "use server";
    await removeRoomPlayerAction(room.code, playerId);
  }

  return <HostControls room={room} currentPlayerId={currentPlayerId} labels={labels} updateSettingsAction={updateSettings} startAction={start} removePlayerAction={removePlayer} />;
}
