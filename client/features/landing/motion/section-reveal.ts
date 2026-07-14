"use client";

import {
  CARD_STAGGER_MIN,
  CARD_Y,
  FLOAT_DURATION,
  FLOAT_Y,
  FOOTER_DURATION,
  FOOTER_Y,
  IMAGE_DURATION,
  IMAGE_TRAVEL,
  LANDING_EASE,
  LANDING_EASE_FLOAT,
  LANDING_EASE_SOFT,
  SECTION_DURATION,
  SECTION_START,
  SECTION_Y,
  TEXT_STAGGER,
} from "./constants";
import { gsap, prefersReducedMotion, ScrollTrigger } from "./gsap-client";

export type SectionRevealDirection = "up" | "left" | "right";

export type SectionRevealOptions = {
  section: HTMLElement;
  direction?: SectionRevealDirection;
  delay?: number;
};

function travelForDirection(direction: SectionRevealDirection) {
  switch (direction) {
    case "left":
      return { x: -SECTION_Y, y: SECTION_Y * 0.35 };
    case "right":
      return { x: SECTION_Y, y: SECTION_Y * 0.35 };
    default:
      return { x: 0, y: SECTION_Y };
  }
}

function imageTravelForDirection(direction: SectionRevealDirection) {
  switch (direction) {
    case "left":
      return { x: -IMAGE_TRAVEL, y: 16 };
    case "right":
      return { x: IMAGE_TRAVEL, y: 16 };
    default:
      return { x: 0, y: IMAGE_TRAVEL };
  }
}

/**
 * Replayable section reveal. Plays on enter (down or up).
 * Resets when the section fully leaves the viewport so it can animate again.
 */
export function createSectionReveal({
  section,
  direction = "up",
  delay = 0,
}: SectionRevealOptions): () => void {
  const reduced = prefersReducedMotion();
  const heading = section.querySelector<HTMLElement>("[data-landing-heading]");
  const paragraph = section.querySelector<HTMLElement>(
    "[data-landing-paragraph]",
  );
  const media = Array.from(
    section.querySelectorAll<HTMLElement>("[data-landing-media]"),
  );
  const cards = Array.from(
    section.querySelectorAll<HTMLElement>("[data-landing-card]"),
  );
  const buttons = Array.from(
    section.querySelectorAll<HTMLElement>("[data-landing-cta]"),
  );

  const targets = [heading, paragraph, ...media, ...cards, ...buttons].filter(
    (el): el is HTMLElement => Boolean(el),
  );

  if (targets.length === 0) {
    return () => undefined;
  }

  if (reduced) {
    gsap.set(targets, { clearProps: "all" });
    return () => undefined;
  }

  const travel = travelForDirection(direction);
  const imageTravel = imageTravelForDirection(direction);
  const floatTweens: gsap.core.Tween[] = [];

  const killFloats = () => {
    floatTweens.forEach((t) => t.kill());
    floatTweens.length = 0;
    media.forEach((el) => gsap.killTweensOf(el, "y"));
  };

  const setHidden = () => {
    killFloats();
    if (heading) {
      gsap.set(heading, {
        autoAlpha: 0,
        x: travel.x,
        y: travel.y,
        force3D: true,
      });
    }
    if (paragraph) {
      gsap.set(paragraph, {
        autoAlpha: 0,
        x: travel.x * 0.6,
        y: travel.y,
        force3D: true,
      });
    }
    buttons.forEach((el) => {
      gsap.set(el, { autoAlpha: 0, y: SECTION_Y * 0.5, force3D: true });
    });
    media.forEach((el) => {
      gsap.set(el, {
        autoAlpha: 0,
        x: imageTravel.x,
        y: imageTravel.y,
        force3D: true,
      });
    });
    cards.forEach((el) => {
      gsap.set(el, { autoAlpha: 0, y: CARD_Y, force3D: true });
    });
  };

  const ctx = gsap.context(() => {
    setHidden();

    const tl = gsap.timeline({
      paused: true,
      delay,
      defaults: { ease: LANDING_EASE, force3D: true, overwrite: "auto" },
    });

    let cursor = 0;

    // Heading first — clear rise + fade
    if (heading) {
      tl.to(
        heading,
        {
          autoAlpha: 1,
          x: 0,
          y: 0,
          duration: SECTION_DURATION,
        },
        cursor,
      );
      cursor += TEXT_STAGGER;
    }

    // Paragraph 150ms after heading
    if (paragraph) {
      tl.to(
        paragraph,
        {
          autoAlpha: 1,
          x: 0,
          y: 0,
          duration: SECTION_DURATION * 0.95,
        },
        cursor,
      );
      cursor += TEXT_STAGGER;
    }

    // CTAs last in the text cascade
    if (buttons.length > 0) {
      tl.to(
        buttons,
        {
          autoAlpha: 1,
          y: 0,
          duration: SECTION_DURATION * 0.85,
          stagger: CARD_STAGGER_MIN,
        },
        cursor,
      );
      cursor += TEXT_STAGGER * 0.5;
    }

    // Media after copy
    if (media.length > 0) {
      media.forEach((el, index) => {
        tl.to(
          el,
          {
            autoAlpha: 1,
            x: 0,
            y: 0,
            duration: IMAGE_DURATION,
            ease: LANDING_EASE_SOFT,
            onComplete: () => {
              const floatTween = gsap.to(el, {
                y: -FLOAT_Y,
                duration: FLOAT_DURATION,
                ease: LANDING_EASE_FLOAT,
                yoyo: true,
                repeat: -1,
                force3D: true,
              });
              floatTweens.push(floatTween);
            },
          },
          cursor + index * 0.1,
        );
      });
    }

    if (cards.length > 0) {
      tl.to(
        cards,
        {
          autoAlpha: 1,
          y: 0,
          duration: SECTION_DURATION,
          stagger: CARD_STAGGER_MIN,
        },
        0.12,
      );
    }

    const playIn = () => {
      killFloats();
      setHidden();
      tl.restart(true);
    };

    const resetOut = () => {
      tl.pause(0);
      setHidden();
    };

    const st = ScrollTrigger.create({
      trigger: section,
      start: SECTION_START,
      end: "bottom top",
      // Replay every time the section is entered from either direction.
      onEnter: playIn,
      onEnterBack: playIn,
      // Hide again when fully leaving so the next visit can animate.
      onLeave: resetOut,
      onLeaveBack: resetOut,
    });

    // Kick off if this section is already in the active range on first paint.
    if (st.isActive) {
      playIn();
    }
  }, section);

  return () => {
    killFloats();
    ctx.revert();
  };
}

export function createFooterReveal(footer: HTMLElement): () => void {
  if (prefersReducedMotion()) {
    gsap.set(footer, { clearProps: "all" });
    return () => undefined;
  }

  const ctx = gsap.context(() => {
    const setHidden = () => {
      gsap.set(footer, { autoAlpha: 0, y: FOOTER_Y, force3D: true });
    };
    setHidden();

    const tween = gsap.to(footer, {
      autoAlpha: 1,
      y: 0,
      duration: FOOTER_DURATION,
      ease: LANDING_EASE,
      force3D: true,
      paused: true,
    });

    ScrollTrigger.create({
      trigger: footer,
      start: SECTION_START,
      end: "bottom top",
      onEnter: () => {
        setHidden();
        tween.restart(true);
      },
      onEnterBack: () => {
        setHidden();
        tween.restart(true);
      },
      onLeave: () => {
        tween.pause(0);
        setHidden();
      },
      onLeaveBack: () => {
        tween.pause(0);
        setHidden();
      },
    });
  }, footer);

  return () => {
    ctx.revert();
  };
}

/**
 * Optional ultra-subtle parallax on large decorative backgrounds.
 * Max travel well under 40px; only transform.
 */
export function createBackgroundParallax(
  section: HTMLElement,
  media: HTMLElement,
): () => void {
  if (prefersReducedMotion()) return () => undefined;

  const ctx = gsap.context(() => {
    gsap.fromTo(
      media,
      { y: -12 },
      {
        y: 12,
        ease: "none",
        force3D: true,
        scrollTrigger: {
          trigger: section,
          start: "top bottom",
          end: "bottom top",
          scrub: true,
        },
      },
    );
  }, section);

  return () => {
    ctx.revert();
  };
}
