import { useTranslations } from "next-intl";
import { Link } from "@/lib/i18n/navigation";

export function ProfileLoadingSkeleton() {
  return (
    <div role="status" aria-live="polite" className="mx-auto flex w-full max-w-2xl flex-col gap-4 px-4 py-8">
      <span className="sr-only">Loading</span>
      <div className="h-8 w-48 animate-pulse rounded-md bg-slate-200" aria-hidden="true" />
      <div className="h-24 animate-pulse rounded-md bg-slate-200" aria-hidden="true" />
      <div className="h-40 animate-pulse rounded-md bg-slate-200" aria-hidden="true" />
    </div>
  );
}

export function ProfileUnauthorizedPanel() {
  const t = useTranslations("Profile.states");

  return (
    <main className="mx-auto flex min-h-[60vh] max-w-xl flex-col items-center justify-center gap-4 px-4 text-center">
      <h1 className="text-2xl font-bold text-slate-950">{t("unauthorizedTitle")}</h1>
      <p className="text-sm text-slate-700">{t("unauthorizedMessage")}</p>
      <Link
        href="/"
        className="inline-flex rounded-md bg-slate-950 px-4 py-2 text-sm font-semibold text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
      >
        {t("backToPlay")}
      </Link>
    </main>
  );
}

export function ProfileNotFoundPanel() {
  const t = useTranslations("Profile.states");

  return (
    <main className="mx-auto flex min-h-[60vh] max-w-xl flex-col items-center justify-center gap-4 px-4 text-center">
      <h1 className="text-2xl font-bold text-slate-950">{t("notFoundTitle")}</h1>
      <p className="text-sm text-slate-700">{t("notFoundMessage")}</p>
      <Link
        href="/"
        className="inline-flex rounded-md bg-slate-950 px-4 py-2 text-sm font-semibold text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
      >
        {t("backToPlay")}
      </Link>
    </main>
  );
}

export function ProfileUnexpectedErrorPanel() {
  const t = useTranslations("Profile.states");

  return (
    <main className="mx-auto flex min-h-[60vh] max-w-xl flex-col items-center justify-center gap-4 px-4 text-center" role="alert">
      <h1 className="text-2xl font-bold text-slate-950">{t("unexpectedTitle")}</h1>
      <p className="text-sm text-slate-700">{t("unexpectedMessage")}</p>
    </main>
  );
}
