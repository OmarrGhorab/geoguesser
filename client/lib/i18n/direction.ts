import type { AppLocale } from "./routing";

export function getDirection(locale: AppLocale) {
  return locale === "ar" ? "rtl" : "ltr";
}
