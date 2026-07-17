import type { AppLocale } from "@/lib/i18n/routing";

export function formatNumber(locale: AppLocale, value: number): string {
  return new Intl.NumberFormat(locale === "ar" ? "ar" : "en").format(value);
}

export function formatAverageScore(locale: AppLocale, value: number): string {
  return new Intl.NumberFormat(locale === "ar" ? "ar" : "en", {
    maximumFractionDigits: 1,
  }).format(value);
}

/** Format ISO date (YYYY-MM-DD) for display. */
export function formatDate(
  locale: AppLocale,
  isoDate: string | null | undefined,
): string {
  if (!isoDate) return "";
  const date = new Date(`${isoDate}T12:00:00.000Z`);
  if (Number.isNaN(date.getTime())) return isoDate;
  return new Intl.DateTimeFormat(locale === "ar" ? "ar" : "en", {
    year: "numeric",
    month: "long",
    day: "numeric",
    timeZone: "UTC",
  }).format(date);
}

/** Weekday short labels for a week containing the challenge date (Mon–Sun). */
export function weekContaining(
  isoDate: string | null | undefined,
): { days: number[]; todayDay: number; monthIndex: number; year: number } {
  const base = isoDate
    ? new Date(`${isoDate}T12:00:00.000Z`)
    : new Date();
  const utcDay = base.getUTCDay(); // 0=Sun
  const mondayOffset = utcDay === 0 ? -6 : 1 - utcDay;
  const monday = new Date(base);
  monday.setUTCDate(base.getUTCDate() + mondayOffset);

  const days: number[] = [];
  for (let i = 0; i < 7; i += 1) {
    const d = new Date(monday);
    d.setUTCDate(monday.getUTCDate() + i);
    days.push(d.getUTCDate());
  }

  return {
    days,
    todayDay: base.getUTCDate(),
    monthIndex: base.getUTCMonth(),
    year: base.getUTCFullYear(),
  };
}

export function formatPlayedToday(
  locale: AppLocale,
  template: string,
  participants: number,
): string {
  const count = formatNumber(locale, participants);
  return template.replace("{count}", count);
}

export function formatWelcome(template: string, displayName: string): string {
  return template.replace("{name}", displayName);
}
