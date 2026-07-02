import type { ReactElement } from "react";
import { render } from "@testing-library/react";
import { NextIntlClientProvider } from "next-intl";
import enMessages from "@/messages/en.json";
import arMessages from "@/messages/ar.json";

export function renderWithIntl(element: ReactElement, locale: "en" | "ar" = "en") {
  const messages = locale === "ar" ? arMessages : enMessages;
  return render(
    <NextIntlClientProvider locale={locale} messages={messages}>
      {element}
    </NextIntlClientProvider>,
  );
}
