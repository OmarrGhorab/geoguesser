import Image from "next/image";
import type { Mission } from "../schemas";
import { MissionClaimForm } from "./mission-claim-form";

type Props = Readonly<{
  locale: string;
  missions: Mission[];
  copy: {
    title: string;
    subtitle: string;
    daily: string;
    weekly: string;
    claim: string;
    claiming: string;
    claimed: string;
    claimSuccess: string;
    claimError: string;
    complete: string;
    empty: string;
    items: Record<string, { title: string; description: string }>;
  };
  progressById: Record<string, string>;
  rewardById: Record<string, string>;
}>;

export function MissionsBoard({
  locale,
  missions,
  copy,
  progressById,
  rewardById,
}: Props) {
  const groups = (["daily", "weekly"] as const).map((cadence) => ({
    cadence,
    missions: missions.filter((mission) => mission.cadence === cadence),
  }));
  return (
    <div className="h-full overflow-y-auto rounded-2xl border border-white/[0.07] bg-[#08071c]/80 px-4 py-8 text-white sm:px-8">
      <section className="mx-auto max-w-5xl">
        <p className="text-xs font-black tracking-[0.2em] text-[#b9a8ff] uppercase">
          {copy.subtitle}
        </p>
        <h1 className="mt-2 text-3xl font-black italic sm:text-4xl">
          {copy.title}
        </h1>
        <div className="mt-8 space-y-8">
          {groups.map(({ cadence, missions: items }) => (
            <section key={cadence}>
              <h2 className="mb-3 text-lg font-bold">
                {cadence === "daily" ? copy.daily : copy.weekly}
              </h2>
              {items.length === 0 ? (
                <p className="rounded-2xl border border-white/10 p-5 text-sm text-white/60">
                  {copy.empty}
                </p>
              ) : (
                <div className="grid gap-3 md:grid-cols-2">
                  {items.map((mission) => {
                    const complete = mission.status === "completed";
                    const claimed = mission.status === "claimed";
                    const itemCopy = copy.items[mission.mission_key] ?? {
                      title: mission.mission_key,
                      description: "",
                    };
                    return (
                      <article
                        key={mission.id}
                        className="rounded-2xl border border-white/10 bg-gradient-to-br from-[#20134f] to-[#0e1030] p-4 shadow-[0_14px_36px_rgba(0,0,0,.25)]"
                      >
                        <div className="flex gap-4">
                          <Image
                            src={`/mission/icons/${mission.icon_key}.png`}
                            alt=""
                            width={72}
                            height={72}
                            className="size-[72px] object-contain"
                          />
                          <div className="min-w-0 flex-1">
                            <h3 className="font-bold">{itemCopy.title}</h3>
                            <p className="mt-1 text-xs text-white/60">
                              {itemCopy.description}
                            </p>
                            <div
                              className="mt-3 h-2 overflow-hidden rounded-full bg-black/25"
                              role="progressbar"
                              aria-label={itemCopy.title}
                              aria-valuemin={0}
                              aria-valuemax={mission.target_value}
                              aria-valuenow={mission.current_value}
                            >
                              <div
                                className="h-full rounded-full bg-gradient-to-r from-[#7d5cff] to-[#f5b942]"
                                style={{
                                  width: `${Math.min(100, (mission.current_value / mission.target_value) * 100)}%`,
                                }}
                              />
                            </div>
                            <p className="mt-1 text-xs text-white/65">
                              {progressById[mission.id]}
                            </p>
                          </div>
                        </div>
                        <div className="mt-4 flex items-center justify-between">
                          <span className="flex items-center gap-1.5 font-black text-[#f5c542]">
                            <Image
                              src="/mission/icons/xp-reward.png"
                              alt=""
                              width={26}
                              height={26}
                              className="size-6 object-contain"
                            />
                            {rewardById[mission.id]}
                          </span>
                          {complete ? (
                            <MissionClaimForm
                              locale={locale}
                              missionId={mission.id}
                              copy={{
                                claim: copy.claim,
                                claiming: copy.claiming,
                                success: copy.claimSuccess,
                                error: copy.claimError,
                              }}
                            />
                          ) : (
                            <span className="text-xs font-bold text-white/55">
                              {claimed ? copy.claimed : copy.complete}
                            </span>
                          )}
                        </div>
                      </article>
                    );
                  })}
                </div>
              )}
            </section>
          ))}
        </div>
      </section>
    </div>
  );
}
