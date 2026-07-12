import { getTranslations, setRequestLocale } from "next-intl/server";
import { Link } from "@/lib/i18n/navigation";
import type { AppLocale } from "@/lib/i18n/routing";

type HomePageProps = Readonly<{
  params: Promise<{ locale: string }>;
}>;

export default async function HomePage({ params }: HomePageProps) {
  const { locale } = await params;
  setRequestLocale(locale as AppLocale);
  const t = await getTranslations("Home");

  return (
    <main className="grid min-h-dvh place-items-center p-[var(--spacing-page)]">
      <div className="flex flex-col items-center gap-6 text-center">
        <h1 className="m-0 text-[clamp(2rem,6vw,4rem)] font-medium tracking-tight">
          {t("title")}
        </h1>
        <div className="flex flex-wrap items-center justify-center gap-3">
          <Link
            href="/sign-up"
            className="rounded-full bg-white px-6 py-2.5 text-sm font-medium text-black transition-colors hover:bg-neutral-200"
          >
            {t("signUpCta")}
          </Link>
          <Link
            href="/login"
            className="rounded-full border border-white/15 bg-transparent px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-white/10"
          >
            {t("logInCta")}
          </Link>
        </div>
      </div>
    </main>
  );
}
