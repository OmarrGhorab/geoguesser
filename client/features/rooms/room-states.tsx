type RoomStateKind = "invalid" | "expired" | "full" | "kicked" | "unauthorized" | "loading" | "empty" | "disabled" | "success" | "unexpected";

type RoomStateMessageProps = {
  kind: RoomStateKind;
  labels: Record<RoomStateKind, string>;
};

export function RoomStateMessage({ kind, labels }: RoomStateMessageProps) {
  const role = kind === "success" || kind === "loading" ? "status" : "alert";
  return (
    <div className="rounded-sm border px-3 py-2 text-sm" role={role} aria-live={role === "status" ? "polite" : "assertive"}>
      {labels[kind]}
    </div>
  );
}
