"use client";

import { useGSAP } from "@gsap/react";
import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { SplitText as GSAPSplitText } from "gsap/SplitText";
import {
  useEffect,
  useRef,
  type CSSProperties,
  type ElementType,
  type JSX,
} from "react";

gsap.registerPlugin(ScrollTrigger, GSAPSplitText, useGSAP);

type SplitTextTag = keyof Pick<
  JSX.IntrinsicElements,
  "h1" | "h2" | "h3" | "h4" | "h5" | "h6" | "p" | "span" | "div"
>;

export type SplitTextProps = {
  text?: string;
  className?: string;
  delay?: number;
  duration?: number;
  ease?: string;
  splitType?: string;
  from?: gsap.TweenVars;
  to?: gsap.TweenVars;
  threshold?: number;
  rootMargin?: string;
  textAlign?: CSSProperties["textAlign"];
  tag?: SplitTextTag;
  id?: string;
  /**
   * When true (default), re-play when the section re-enters the viewport
   * (scroll up or down). When false, animate only once.
   */
  replay?: boolean;
  onLetterAnimationComplete?: () => void;
};

type SplitHostElement = HTMLElement & {
  _rbsplitInstance?: GSAPSplitText | null;
};

async function waitForFonts(): Promise<void> {
  if (typeof document === "undefined" || !("fonts" in document)) return;
  if (document.fonts.status === "loaded") return;
  await document.fonts.ready;
}

function scrollStartFromProps(threshold: number, rootMargin: string): string {
  const startPct = (1 - threshold) * 100;
  const marginMatch = /^(-?\d+(?:\.\d+)?)(px|em|rem|%)?$/.exec(rootMargin);
  const marginValue = marginMatch ? parseFloat(marginMatch[1]) : 0;
  const marginUnit = marginMatch ? marginMatch[2] || "px" : "px";
  const sign =
    marginValue === 0
      ? ""
      : marginValue < 0
        ? `-=${Math.abs(marginValue)}${marginUnit}`
        : `+=${marginValue}${marginUnit}`;
  return `top ${startPct}%${sign}`;
}

export default function SplitText({
  text = "",
  className = "",
  delay = 50,
  duration = 1.25,
  ease = "power3.out",
  splitType = "chars",
  from = { opacity: 0, y: 40 },
  to = { opacity: 1, y: 0 },
  threshold = 0.1,
  rootMargin = "-100px",
  textAlign,
  tag = "p",
  id,
  replay = true,
  onLetterAnimationComplete,
}: SplitTextProps) {
  const ref = useRef<SplitHostElement | null>(null);
  const onCompleteRef = useRef(onLetterAnimationComplete);

  useEffect(() => {
    onCompleteRef.current = onLetterAnimationComplete;
  }, [onLetterAnimationComplete]);

  useGSAP(
    () => {
      if (!ref.current || !text) return;

      const el = ref.current;
      let cancelled = false;
      let splitInstance: GSAPSplitText | null = null;
      let tween: gsap.core.Tween | null = null;
      let trigger: ScrollTrigger | null = null;

      const cleanup = () => {
        trigger?.kill();
        trigger = null;
        tween?.kill();
        tween = null;
        if (splitInstance) {
          try {
            splitInstance.revert();
          } catch {
            /* noop */
          }
          splitInstance = null;
        }
        el._rbsplitInstance = null;
      };

      const setup = async () => {
        await waitForFonts();
        if (cancelled || !ref.current) return;

        const reducedMotion =
          typeof window !== "undefined" &&
          window.matchMedia("(prefers-reduced-motion: reduce)").matches;

        if (reducedMotion) {
          gsap.set(el, { clearProps: "all" });
          onCompleteRef.current?.();
          return;
        }

        cleanup();

        // Prefer the landing section so text re-triggers with the rest of the
        // section (media / snap), not only when this node itself crosses.
        const sectionTrigger =
          el.closest<HTMLElement>("[data-landing-section]") ?? el;
        const start = scrollStartFromProps(threshold, rootMargin);

        let targets: Element[] | undefined;
        const assignTargets = (self: GSAPSplitText) => {
          if (splitType.includes("chars") && self.chars.length) {
            targets = self.chars;
          }
          if (!targets && splitType.includes("words") && self.words.length) {
            targets = self.words;
          }
          if (!targets && splitType.includes("lines") && self.lines.length) {
            targets = self.lines;
          }
          if (!targets) {
            targets = self.chars.length
              ? self.chars
              : self.words.length
                ? self.words
                : self.lines;
          }
        };

        splitInstance = new GSAPSplitText(el, {
          type: splitType,
          smartWrap: true,
          autoSplit: splitType === "lines",
          linesClass: "split-line",
          wordsClass: "split-word",
          charsClass: "split-char",
          reduceWhiteSpace: false,
          onSplit: (self) => {
            assignTargets(self);
            if (!targets?.length) return;

            // SplitText may re-split on resize; drop prior tween/trigger first.
            trigger?.kill();
            trigger = null;
            tween?.kill();
            tween = null;

            gsap.set(targets, { ...from, force3D: true });

            tween = gsap.fromTo(
              targets,
              { ...from },
              {
                ...to,
                duration,
                ease,
                stagger: delay / 1000,
                paused: true,
                force3D: true,
                willChange: "transform, opacity",
                onComplete: () => {
                  if (targets) {
                    gsap.set(targets, { clearProps: "willChange" });
                  }
                  onCompleteRef.current?.();
                },
              },
            );

            const playIn = () => {
              if (!targets?.length || !tween) return;
              gsap.set(targets, { ...from, force3D: true });
              tween.restart(true);
            };

            const resetOut = () => {
              if (!targets?.length || !tween) return;
              tween.pause(0);
              gsap.set(targets, { ...from, force3D: true });
            };

            if (replay) {
              trigger = ScrollTrigger.create({
                trigger: sectionTrigger,
                start,
                end: "bottom top",
                onEnter: playIn,
                onEnterBack: playIn,
                onLeave: resetOut,
                onLeaveBack: resetOut,
              });
            } else {
              trigger = ScrollTrigger.create({
                trigger: sectionTrigger,
                start,
                once: true,
                onEnter: playIn,
              });
            }

            if (trigger.isActive) {
              playIn();
            }

            return tween;
          },
        });

        if (cancelled) {
          cleanup();
          return;
        }

        el._rbsplitInstance = splitInstance;
      };

      void setup();

      return () => {
        cancelled = true;
        cleanup();
      };
    },
    {
      dependencies: [
        text,
        delay,
        duration,
        ease,
        splitType,
        JSON.stringify(from),
        JSON.stringify(to),
        threshold,
        rootMargin,
        replay,
      ],
      scope: ref,
    },
  );

  const Tag = (tag || "p") as ElementType;
  // Keep layout close to a normal heading/paragraph — avoid shrink-to-fit
  // inline-block + forced center alignment shifting section copy.
  const style: CSSProperties = {
    overflow: "hidden",
    display: "block",
    whiteSpace: "normal",
    wordWrap: "break-word",
    ...(textAlign ? { textAlign } : null),
  };

  return (
    <Tag
      ref={ref}
      id={id}
      style={style}
      className={`split-parent ${className}`}
    >
      {text}
    </Tag>
  );
}
