import type { AppLocale } from "@/lib/i18n/routing";
import type { AuthenticatedHomeCopy } from "@/features/home/types";
import { AuthenticatedHomeMain } from "./authenticated-home-main";
import { AuthenticatedHomeRail } from "./authenticated-home-rail";
import { AuthenticatedShell } from "./authenticated-shell";

type AuthenticatedHomeProps = Readonly<{
  locale: AppLocale;
  copy: AuthenticatedHomeCopy;
}>;

/**
 * Authenticated home dashboard: shell chrome + home main column + right rail.
 * For other authenticated pages, use `AuthenticatedShell` directly with your content.
 */
export function AuthenticatedHome({ locale, copy }: AuthenticatedHomeProps) {
  return (
    <main className="h-dvh max-h-dvh overflow-hidden">
      <AuthenticatedShell
        locale={locale}
        copy={copy}
        activeSidebarItem="home"
        rightRail={<AuthenticatedHomeRail locale={locale} copy={copy} />}
      >
        <AuthenticatedHomeMain copy={copy} />
      </AuthenticatedShell>
    </main>
  );
}

export type { AuthenticatedHomeCopy };
export { AuthenticatedNavbar } from "./authenticated-navbar";
export { AuthenticatedSidebar } from "./authenticated-sidebar";
export { AuthenticatedShell } from "./authenticated-shell";
export { AuthenticatedHomeMain } from "./authenticated-home-main";
export { AuthenticatedHomeRail } from "./authenticated-home-rail";
