"use client";

import Image from "next/image";
import { motion } from "motion/react";

export type MissionFeatureCard = {
  key: "locations" | "compare" | "daily";
  src: string;
  label: string;
};

type MissionFeatureCardsProps = Readonly<{
  cards: MissionFeatureCard[];
}>;

/**
 * Three mission feature cards with staggered entrance animation.
 */
export function MissionFeatureCards({ cards }: MissionFeatureCardsProps) {
  return (
    <ul
      data-testid="mission-feature-cards"
      className="grid w-full max-w-4xl grid-cols-1 gap-4 sm:grid-cols-3 sm:gap-5"
    >
      {cards.map((card, index) => (
        <motion.li
          key={card.key}
          initial={{ opacity: 0, y: 36, scale: 0.92 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{
            duration: 0.55,
            delay: 0.12 + index * 0.14,
            ease: [0.22, 1, 0.36, 1],
          }}
        >
          <article
            data-mission-card={card.key}
            className="flex h-full flex-col items-center rounded-[1.35rem] border border-[#7B5CFF]/55 bg-[#120B35]/72 px-5 py-7 text-center shadow-[0_16px_40px_rgba(20,5,60,0.55),inset_0_1px_0_rgba(255,255,255,0.08)] backdrop-blur-md transition duration-300 hover:-translate-y-1 hover:border-[#9B84FF]/80 hover:shadow-[0_20px_48px_rgba(80,40,180,0.45)]"
          >
            <div className="relative mb-5 h-[5.5rem] w-full sm:h-[6.25rem]">
              <Image
                src={card.src}
                alt=""
                fill
                sizes="200px"
                className="object-contain drop-shadow-[0_12px_28px_rgba(0,0,0,0.45)]"
                priority
              />
            </div>
            <h2 className="text-[0.78rem] font-black tracking-[0.06em] text-white uppercase sm:text-[0.82rem]">
              {card.label}
            </h2>
          </article>
        </motion.li>
      ))}
    </ul>
  );
}
