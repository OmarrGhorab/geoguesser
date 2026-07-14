/** Premium landing motion tokens — subtle, cinematic, non-bouncy. */

export const LANDING_EASE = "power3.out" as const;
export const LANDING_EASE_SOFT = "power2.out" as const;
export const LANDING_EASE_FLOAT = "sine.inOut" as const;

export const SECTION_START = "top 80%";

/** Global section rise */
export const SECTION_DURATION = 0.8;
export const SECTION_Y = 40;

/** Text cascade (seconds between stages) */
export const TEXT_STAGGER = 0.15;

/** Card / multi-item stagger */
export const CARD_STAGGER_MIN = 0.08;
export const CARD_STAGGER_MAX = 0.12;
export const CARD_Y = 30;

/** Image reveal */
export const IMAGE_TRAVEL = 40;
export const IMAGE_DURATION = 0.85;

/** Post-reveal float (barely noticeable) */
export const FLOAT_Y = 6;
export const FLOAT_DURATION = 7;

/** Footer */
export const FOOTER_Y = 20;
export const FOOTER_DURATION = 0.75;

/** Hover micro-interactions (Framer Motion) */
export const HOVER_LIFT = -6;
export const HOVER_SCALE = 1.02;
export const HOVER_MS = 0.25;
