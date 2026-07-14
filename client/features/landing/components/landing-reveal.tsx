import type { ReactNode } from "react";
import type { SectionRevealDirection } from "@/features/landing/motion";

type LandingRevealProps = {
  id: string;
  labelledBy: string;
  className: string;
  children: ReactNode;
  /** Skip GSAP scroll reveal (hero). */
  skipMotion?: boolean;
  /** Alternate entrance direction for post-hero sections. */
  direction?: SectionRevealDirection;
};

/** Semantic section shell — motion attributes only, no layout changes. */
export function LandingReveal({
  id,
  labelledBy,
  className,
  children,
  skipMotion = false,
  direction = "up",
}: LandingRevealProps) {
  return (
    <section
      id={id}
      aria-labelledby={labelledBy}
      className={className}
      data-landing-section=""
      data-landing-direction={direction}
      {...(skipMotion ? { "data-landing-skip": "" } : {})}
    >
      {children}
    </section>
  );
}
