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
    <div className="bg-auth-bg text-foreground flex min-h-dvh w-full flex-col font-sans antialiased selection:bg-white/20 selection:text-white lg:flex-row">
      {/* Shared left media panel — persists across auth routes */}
      <div className="relative hidden w-full flex-col justify-end p-4 lg:flex lg:min-h-dvh lg:w-1/2">
        <div className="border-border relative h-full min-h-[calc(100dvh-2rem)] w-full overflow-hidden rounded-[32px] border shadow-2xl">
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

      {/* Right form panel — page content changes here */}
      <div className="flex w-full flex-col items-center justify-center p-6 sm:p-12 lg:w-1/2">
        {children}
      </div>
    </div>
  );
}
