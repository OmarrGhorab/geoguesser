"use client";

import { motion, useReducedMotion } from "motion/react";
import type { ReactNode } from "react";

type LandingRevealProps = {
  id: string;
  labelledBy: string;
  className: string;
  children: ReactNode;
};

export function LandingReveal({
  id,
  labelledBy,
  className,
  children,
}: LandingRevealProps) {
  const shouldReduceMotion = useReducedMotion();

  return (
    <motion.section
      id={id}
      aria-labelledby={labelledBy}
      className={className}
      initial={shouldReduceMotion ? false : { opacity: 0, y: 24 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, amount: 0.18, margin: "0px 0px -40px 0px" }}
      transition={
        shouldReduceMotion
          ? { duration: 0 }
          : { duration: 0.6, ease: [0.22, 1, 0.36, 1] }
      }
    >
      {children}
    </motion.section>
  );
}
