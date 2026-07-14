"use client";

import { useEffect, type ReactNode } from "react";
import {
  createBackgroundParallax,
  createFooterReveal,
  createSectionReveal,
  ensureGsapPlugins,
  ScrollTrigger,
  type SectionRevealDirection,
} from "@/features/landing/motion";

type LandingScrollRootProps = {
  children: ReactNode;
};

/**
 * Boots GSAP ScrollTrigger for landing:
 * - post-hero section text/media reveals (replay on re-enter)
 * - footer fade
 * Hero motion is skipped via [data-landing-skip].
 */
export function LandingScrollRoot({ children }: LandingScrollRootProps) {
  useEffect(() => {
    ensureGsapPlugins();

    const cleanups: Array<() => void> = [];

    const animatedSections = document.querySelectorAll<HTMLElement>(
      "[data-landing-section]:not([data-landing-skip])",
    );

    animatedSections.forEach((section) => {
      const direction = (section.dataset.landingDirection ||
        "up") as SectionRevealDirection;

      cleanups.push(createSectionReveal({ section, direction }));

      const parallaxTarget = section.querySelector<HTMLElement>(
        "[data-landing-parallax]",
      );
      if (parallaxTarget) {
        cleanups.push(createBackgroundParallax(section, parallaxTarget));
      }
    });

    const footer = document.querySelector<HTMLElement>("[data-landing-footer]");
    if (footer) {
      cleanups.push(createFooterReveal(footer));
    }

    // Layout settle (images/fonts/split text) then refresh trigger positions.
    const refreshId = window.requestAnimationFrame(() => {
      ScrollTrigger.refresh();
    });
    // SplitText waits on document.fonts; refresh again after fonts are ready.
    let fontsRefreshId = 0;
    const fontsReady =
      "fonts" in document
        ? document.fonts.ready.then(() => {
            fontsRefreshId = window.requestAnimationFrame(() => {
              ScrollTrigger.refresh();
            });
          })
        : Promise.resolve();

    return () => {
      window.cancelAnimationFrame(refreshId);
      window.cancelAnimationFrame(fontsRefreshId);
      void fontsReady;
      cleanups.forEach((fn) => fn());
    };
  }, []);

  return children;
}
