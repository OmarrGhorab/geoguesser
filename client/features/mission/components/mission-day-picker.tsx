"use client";

import { useMemo, useState, useTransition } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import type { Route } from "next";
import { ChevronLeft, ChevronRight } from "lucide-react";
import type { DailyMissionCopy } from "@/features/mission/types";

type MissionDayPickerProps = Readonly<{
  copy: Pick<
    DailyMissionCopy,
    "today" | "prevDay" | "nextDay" | "aria" | "weekdays" | "months"
  >;
  selectedDate: string;
  todayDate: string;
}>;

function startOfDay(d: Date) {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate());
}

function isSameDay(a: Date, b: Date) {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

function parseDateKey(value: string) {
  const [year, month, day] = value.split("-").map(Number);
  return new Date(year, month - 1, day, 12);
}

function toDateKey(value: Date) {
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function buildMonthGrid(year: number, monthIndex: number) {
  const first = new Date(year, monthIndex, 1);
  const startOffset = (first.getDay() + 6) % 7; // Monday-first
  const daysInMonth = new Date(year, monthIndex + 1, 0).getDate();
  const cells: Array<Date | null> = [];
  for (let i = 0; i < startOffset; i += 1) cells.push(null);
  for (let day = 1; day <= daysInMonth; day += 1) {
    cells.push(new Date(year, monthIndex, day));
  }
  while (cells.length % 7 !== 0) cells.push(null);
  return cells;
}

/**
 * TODAY control + prev/next + dropdown calendar for mission day selection.
 */
export function MissionDayPicker({
  copy,
  selectedDate,
  todayDate,
}: MissionDayPickerProps) {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [, startTransition] = useTransition();
  const today = useMemo(() => startOfDay(parseDateKey(todayDate)), [todayDate]);
  const selected = useMemo(
    () => startOfDay(parseDateKey(selectedDate)),
    [selectedDate],
  );
  const [open, setOpen] = useState(false);
  const [viewYear, setViewYear] = useState(selected.getFullYear());
  const [viewMonth, setViewMonth] = useState(selected.getMonth());

  const label = isSameDay(selected, today)
    ? copy.today
    : selected.toLocaleDateString(undefined, {
        month: "short",
        day: "numeric",
      });

  const grid = useMemo(
    () => buildMonthGrid(viewYear, viewMonth),
    [viewYear, viewMonth],
  );

  const navigateToDay = (day: Date) => {
    setViewYear(day.getFullYear());
    setViewMonth(day.getMonth());
    const params = new URLSearchParams(searchParams.toString());
    const key = toDateKey(day);
    if (key === todayDate) params.delete("date");
    else params.set("date", key);
    const query = params.toString();
    startTransition(() =>
      router.replace((query ? `${pathname}?${query}` : pathname) as Route),
    );
  };

  const shiftDay = (delta: number) => {
    const next = new Date(selected);
    next.setDate(selected.getDate() + delta);
    const clamped = startOfDay(next);
    if (clamped.getTime() > today.getTime()) return;
    navigateToDay(clamped);
  };

  const selectDay = (day: Date) => {
    const d = startOfDay(day);
    if (d.getTime() > today.getTime()) return;
    navigateToDay(d);
    setOpen(false);
  };

  const shiftMonth = (delta: number) => {
    const next = new Date(viewYear, viewMonth + delta, 1);
    setViewYear(next.getFullYear());
    setViewMonth(next.getMonth());
  };

  return (
    <div
      role="group"
      aria-label={copy.aria.dayPicker}
      className="absolute top-5 left-1/2 z-[100] -translate-x-1/2 sm:top-6"
    >
      <div className="relative z-[100] flex items-center gap-1 rounded-full border border-white/15 bg-[#1A1040]/90 px-2 py-1.5 shadow-[0_10px_30px_rgba(40,10,90,0.55)] backdrop-blur-md">
        <button
          type="button"
          aria-label={copy.prevDay}
          onClick={() => shiftDay(-1)}
          className="grid size-9 place-items-center rounded-full text-white/80 transition hover:bg-white/10 hover:text-white"
        >
          <ChevronLeft className="size-5" />
        </button>
        <button
          type="button"
          data-testid="mission-today-control"
          aria-expanded={open}
          aria-haspopup="dialog"
          onClick={() => setOpen((v) => !v)}
          className="flex min-w-[7.5rem] items-center justify-center px-3 text-sm font-black tracking-[0.14em] text-white uppercase"
        >
          {label}
        </button>
        <button
          type="button"
          aria-label={copy.nextDay}
          onClick={() => shiftDay(1)}
          disabled={isSameDay(selected, today)}
          className="grid size-9 place-items-center rounded-full text-white/80 transition hover:bg-white/10 hover:text-white disabled:cursor-not-allowed disabled:opacity-35"
        >
          <ChevronRight className="size-5" />
        </button>
      </div>

      {open ? (
        <div
          data-testid="mission-calendar"
          role="dialog"
          aria-label={copy.aria.dayPicker}
          className="absolute top-[calc(100%+0.65rem)] left-1/2 z-[110] w-[18.5rem] -translate-x-1/2 rounded-2xl border border-white/12 bg-[#120B35]/98 p-3 shadow-[0_24px_60px_rgba(10,0,40,0.75)] backdrop-blur-xl"
        >
          <div className="mb-2 flex items-center justify-between px-1">
            <button
              type="button"
              aria-label={copy.prevDay}
              onClick={() => shiftMonth(-1)}
              className="grid size-8 place-items-center rounded-lg text-white/80 transition hover:bg-white/10"
            >
              <ChevronLeft className="size-4" />
            </button>
            <p className="text-sm font-bold tracking-wide text-white">
              {`${copy.months[viewMonth] ?? ""} ${viewYear}`}
            </p>
            <button
              type="button"
              aria-label={copy.nextDay}
              onClick={() => shiftMonth(1)}
              className="grid size-8 place-items-center rounded-lg text-white/80 transition hover:bg-white/10"
            >
              <ChevronRight className="size-4" />
            </button>
          </div>

          <div className="mb-1 grid grid-cols-7 gap-1 text-center text-[0.62rem] font-semibold tracking-wide text-white/45 uppercase">
            {copy.weekdays.map((d) => (
              <span key={d} className="py-1">
                {d}
              </span>
            ))}
          </div>

          <div className="grid grid-cols-7 gap-1">
            {grid.map((cell, index) => {
              if (!cell) {
                return <span key={`empty-${index}`} className="size-9" />;
              }
              const isSelected = isSameDay(cell, selected);
              const isToday = isSameDay(cell, today);
              const isFuture = cell.getTime() > today.getTime();
              return (
                <button
                  key={cell.toISOString()}
                  type="button"
                  disabled={isFuture}
                  onClick={() => selectDay(cell)}
                  data-selected={isSelected ? "true" : "false"}
                  data-today={isToday ? "true" : "false"}
                  className={`grid size-9 place-items-center rounded-full text-sm transition ${
                    isSelected
                      ? "bg-[#6B4EFF] font-bold text-white shadow-[0_0_14px_rgba(107,78,255,0.55)]"
                      : isToday
                        ? "border border-[#7DD3FC]/70 font-semibold text-white"
                        : "text-white/80 hover:bg-white/10"
                  } disabled:cursor-not-allowed disabled:text-white/25 disabled:hover:bg-transparent`}
                >
                  {cell.getDate()}
                </button>
              );
            })}
          </div>
        </div>
      ) : null}
    </div>
  );
}
