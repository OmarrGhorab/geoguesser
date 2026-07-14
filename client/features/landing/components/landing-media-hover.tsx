"use client";

import { motion } from "motion/react";
import type { ReactNode } from "react";
import { HOVER_LIFT, HOVER_MS, HOVER_SCALE } from "@/features/landing/motion";
import { cn } from "@/lib/utils";

type LandingMediaHoverProps = {
  children: ReactNode;
  className?: string;
  /** Disable lift on very wide landscape art if needed */
  enabled?: boolean;
};

/**
 * Framer Motion micro-interaction only — hover lift / soft scale.
 * Does not change layout; uses transform only.
 */
export function LandingMediaHover({
  children,
  className,
  enabled = true,
}: LandingMediaHoverProps) {
  if (!enabled) {
    return <div className={className}>{children}</div>;
  }

  return (
    <motion.div
      className={cn("will-change-transform", className)}
      whileHover={{
        y: HOVER_LIFT,
        scale: HOVER_SCALE,
        transition: { duration: HOVER_MS, ease: [0.22, 1, 0.36, 1] },
      }}
      transition={{ duration: HOVER_MS, ease: [0.22, 1, 0.36, 1] }}
    >
      {children}
    </motion.div>
  );
}
