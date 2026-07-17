import type { ReactNode } from "react";
import type { AppLocale } from "@/lib/i18n/routing";
import type {
  AuthenticatedChromeCopy,
  AuthenticatedSidebarItem,
} from "@/features/home/types";
import type { AuthenticatedTopNavId } from "@/features/home/nav-config";
import { AuthenticatedNavbar } from "./authenticated-navbar";
import { AuthenticatedSidebar } from "./authenticated-sidebar";
import { PAGE_BG } from "./shared";

type AuthenticatedShellProps = Readonly<{
  locale: AppLocale;
  copy: AuthenticatedChromeCopy;
  children: ReactNode;
  rightRail?: ReactNode;
  activeSidebarItem?: AuthenticatedSidebarItem;
  activeTopNavId?: AuthenticatedTopNavId;
  showPremiumPromo?: boolean;
}>;

export function AuthenticatedShell({
  locale,
  copy,
  children,
  rightRail,
  activeSidebarItem = "home",
  activeTopNavId,
  showPremiumPromo = true,
}: AuthenticatedShellProps) {
  const bodyGrid = rightRail
    ? "mx-auto grid h-full min-h-0 w-full max-w-[1560px] grid-cols-1 lg:grid-cols-[10.5rem_minmax(0,1fr)] xl:grid-cols-[10.5rem_minmax(0,1fr)_18.5rem]"
    : "mx-auto grid h-full min-h-0 w-full max-w-[1560px] grid-cols-1 lg:grid-cols-[10.5rem_minmax(0,1fr)]";

  return (
    <div
      className={`flex h-dvh max-h-dvh flex-col overflow-hidden text-white ${PAGE_BG}`}
    >
      <AuthenticatedNavbar
        locale={locale}
        brand={copy.brand}
        nav={copy.nav}
        viewerName={copy.viewerName}
        viewerAvatarUrl={copy.viewerAvatarUrl}
        aria={copy.aria}
        activeTopNavId={activeTopNavId}
      />

      <div
        className={`${bodyGrid} min-h-0 flex-1 gap-2.5 overflow-hidden px-3 pt-2 pb-2 sm:px-4 lg:gap-3.5 lg:px-5 lg:pt-3 lg:pb-3`}
      >
        <AuthenticatedSidebar
          locale={locale}
          nav={copy.nav}
          premium={copy.premium}
          dashboardAriaLabel={copy.aria.dashboard}
          activeItem={activeSidebarItem}
          showPremiumPromo={showPremiumPromo}
        />

        <div className="min-h-0 min-w-0 overflow-hidden">{children}</div>

        {rightRail ? (
          <aside className="hidden min-h-0 overflow-hidden xl:block">
            {rightRail}
          </aside>
        ) : null}
      </div>
    </div>
  );
}
