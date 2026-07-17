"use client";

import { useActionState } from "react";
import { claimMissionAction, initialClaimMissionState } from "../actions";

type Props = Readonly<{
  locale: string;
  missionId: string;
  copy: {
    claim: string;
    claiming: string;
    success: string;
    error: string;
  };
}>;

export function MissionClaimForm({ locale, missionId, copy }: Props) {
  const [state, action, pending] = useActionState(
    claimMissionAction,
    initialClaimMissionState,
  );
  const succeeded = state.status === "success";

  return (
    <form action={action}>
      <input type="hidden" name="locale" value={locale} />
      <input type="hidden" name="missionId" value={missionId} />
      <button
        type="submit"
        disabled={pending || succeeded}
        className="rounded-full bg-[#7654e8] px-4 py-2 text-xs font-black uppercase transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
      >
        {pending ? copy.claiming : copy.claim}
      </button>
      <p
        className={`mt-2 max-w-48 text-xs ${state.status === "error" ? "text-red-300" : "text-emerald-300"}`}
        aria-live="polite"
        role={state.status === "error" ? "alert" : "status"}
      >
        {state.status === "error"
          ? copy.error
          : state.status === "success"
            ? copy.success
            : ""}
      </p>
    </form>
  );
}
