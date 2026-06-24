import { getRequestConfig } from "next-intl/server";
import { defaultLocale, locales, type AppLocale } from "./routing";

export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale;
  const locale: AppLocale = locales.includes(requested as AppLocale)
    ? (requested as AppLocale)
    : defaultLocale;

  return {
    locale,
    messages: (await import(`@/messages/${locale}.json`)).default,
  };
});
