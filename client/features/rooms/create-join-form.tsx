import { createRoom, joinRoom } from "@/lib/api/rooms";
import { redirect } from "@/lib/i18n/navigation";

type CreateJoinFormProps = {
  locale: string;
  labels: {
    title: string;
    create: string;
    join: string;
    mapId: string;
    roomCode: string;
    displayName: string;
  };
};

export function CreateJoinForm({ locale, labels }: CreateJoinFormProps) {
  async function createRoomAction(formData: FormData) {
    "use server";

    const mapId = String(formData.get("map_id") ?? "");
    const displayName = String(formData.get("display_name") ?? "");
    const response = await createRoom({
      map_id: mapId,
      visibility: "private",
      round_count: 5,
      timer_seconds: null,
      max_players: 8,
      display_name: displayName || undefined,
    });

    redirect({ href: `/rooms/${response.room.code}`, locale });
  }

  async function joinRoomAction(formData: FormData) {
    "use server";

    const code = String(formData.get("code") ?? "");
    const displayName = String(formData.get("display_name") ?? "");
    const response = await joinRoom({ code, display_name: displayName || undefined });

    redirect({ href: `/rooms/${response.room.code}`, locale });
  }

  return (
    <main className="mx-auto grid min-h-screen w-full max-w-5xl gap-8 px-4 py-8 md:grid-cols-2">
      <section className="flex flex-col gap-4">
        <h1 className="text-3xl font-semibold tracking-normal">{labels.title}</h1>
        <form action={createRoomAction} className="flex flex-col gap-3">
          <label className="flex flex-col gap-1 text-sm font-medium">
            {labels.mapId}
            <input className="rounded-sm border px-3 py-2" name="map_id" required />
          </label>
          <label className="flex flex-col gap-1 text-sm font-medium">
            {labels.displayName}
            <input className="rounded-sm border px-3 py-2" maxLength={32} minLength={2} name="display_name" />
          </label>
          <button className="rounded-sm bg-zinc-950 px-3 py-2 text-sm font-medium text-white focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-zinc-950" type="submit" aria-label={labels.create}>
            {labels.create}
          </button>
        </form>
      </section>

      <section className="flex flex-col justify-end gap-4">
        <form action={joinRoomAction} className="flex flex-col gap-3">
          <label className="flex flex-col gap-1 text-sm font-medium">
            {labels.roomCode}
            <input className="rounded-sm border px-3 py-2 uppercase" maxLength={10} minLength={6} name="code" required />
          </label>
          <label className="flex flex-col gap-1 text-sm font-medium">
            {labels.displayName}
            <input className="rounded-sm border px-3 py-2" maxLength={32} minLength={2} name="display_name" />
          </label>
          <button className="rounded-sm border px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-zinc-950" type="submit" aria-label={labels.join}>
            {labels.join}
          </button>
        </form>
      </section>
    </main>
  );
}
