import type { AttemptSummary } from "./types";

export function resumeLabelKey(attempt?: AttemptSummary) {
  if (!attempt) {
    return "start";
  }
  if (attempt.status === "completed") {
    return "completed";
  }
  if (attempt.status === "expired") {
    return "expired";
  }
  return "resume";
}
