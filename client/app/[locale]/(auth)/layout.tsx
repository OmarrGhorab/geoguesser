import { getTranslations, setRequestLocale } from "next-intl/server";
import type { AppLocale } from "@/lib/i18n/routing";

type AuthLayoutProps = Readonly<{
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}>;

export default async function AuthLayout({
  children,
  params,
}: AuthLayoutProps) {
  const { locale } = await params;
  const appLocale = locale as AppLocale;
  setRequestLocale(appLocale);

  const t = await getTranslations({
    locale: appLocale,
    namespace: "Auth.Common",
  });

  return (
    <div className="auth-page-atmosphere text-foreground selection:bg-auth-accent/30 relative flex min-h-dvh w-full flex-col overflow-hidden font-sans antialiased selection:text-white lg:flex-row">
      {/* Shared left media panel — persists across auth routes (unchanged video) */}
      <div className="relative z-10 hidden w-full flex-col justify-end p-3 lg:flex lg:min-h-dvh lg:w-1/2">
        <div className="relative h-full min-h-[calc(100dvh-1.5rem)] w-full overflow-hidden rounded-[28px] border border-white/15 shadow-2xl">
          <video
            className="absolute inset-0 h-full w-full object-cover"
            src="/authentication/auth-video.mp4"
            autoPlay
            muted
            loop
            playsInline
            preload="metadata"
            aria-hidden="true"
            tabIndex={-1}
          />
          <span className="sr-only">{t("videoAriaLabel")}</span>
        </div>
      </div>

      {/* Right form panel — extracted reference artwork + live form chrome */}
      <div className="auth-form-atmosphere relative z-10 flex min-h-dvh w-full flex-col items-center overflow-x-hidden overflow-y-auto p-6 sm:p-10 lg:min-h-dvh lg:w-1/2">
        <div className="relative z-10 my-auto flex w-full flex-col items-center justify-center py-8 lg:py-12">
          {children}
        </div>
      </div>
    </div>
  );
}
