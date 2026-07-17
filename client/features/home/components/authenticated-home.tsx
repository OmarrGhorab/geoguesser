import type { AppLocale } from "@/lib/i18n/routing";
import type { AuthenticatedHomeCopy } from "@/features/home/types";
import type { AuthenticatedHomeData } from "@/features/home/schemas";
import { AuthenticatedHomeMain } from "./authenticated-home-main";
import { AuthenticatedHomeRail } from "./authenticated-home-rail";
import { AuthenticatedShell } from "./authenticated-shell";

type AuthenticatedHomeProps = Readonly<{
  locale: AppLocale;
  copy: AuthenticatedHomeCopy;
  data: AuthenticatedHomeData;
}>;

/**
 * Authenticated home dashboard: shell chrome + home main column + right rail.
 * `copy` is i18n labels; `data` is the GET /home snapshot.
 */
export function AuthenticatedHome({
  locale,
  copy,
  data,
}: AuthenticatedHomeProps) {
  return (
    <main className="h-dvh max-h-dvh overflow-hidden">
      <AuthenticatedShell
        locale={locale}
        copy={{
          brand: copy.brand,
          nav: copy.nav,
          premium: copy.premium,
          languageLabel: copy.languageLabel,
          aria: copy.aria,
          viewerName: data.viewer.display_name,
          viewerAvatarUrl: data.viewer.avatar_url ?? null,
        }}
        activeSidebarItem="home"
        rightRail={
          <AuthenticatedHomeRail locale={locale} copy={copy} data={data} />
        }
      >
        <AuthenticatedHomeMain locale={locale} copy={copy} data={data} />
      </AuthenticatedShell>
    </main>
  );
}

export type { AuthenticatedHomeCopy };
export type { AuthenticatedHomeData };
export { AuthenticatedNavbar } from "./authenticated-navbar";
export { AuthenticatedSidebar } from "./authenticated-sidebar";
export { AuthenticatedShell } from "./authenticated-shell";
export { AuthenticatedHomeMain } from "./authenticated-home-main";
export { AuthenticatedHomeRail } from "./authenticated-home-rail";
