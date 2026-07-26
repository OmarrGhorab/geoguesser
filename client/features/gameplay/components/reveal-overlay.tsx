"use client";

const REACTIONS = [
  {
    emoji: "🪝",
    label: "Hooked",
    position: "top-[7%] left-[50%] -translate-x-1/2",
  },
  { emoji: "🤯", label: "Mind blown", position: "top-[25%] right-[16%]" },
  { emoji: "😮", label: "Surprised", position: "bottom-[21%] right-[16%]" },
  {
    emoji: "😍",
    label: "Love it",
    position: "bottom-[5%] left-[50%] -translate-x-1/2",
  },
  { emoji: "🔥", label: "On fire", position: "bottom-[21%] left-[16%]" },
  { emoji: "5K", label: "Perfect score", position: "top-[25%] left-[16%]" },
] as const;

export type RevealOverlayCopy = Readonly<{
  reactionsTitle: string;
  closeReactions: string;
  reactionsPrompt: string;
  reactionSent: string;
}>;

type RevealOverlayProps = Readonly<{
  open: boolean;
  copy: RevealOverlayCopy;
  reaction: string | null;
  onClose: () => void;
  onReaction: (emoji: string) => void;
}>;

/** Reactions wheel dialog extracted from Daily Mission play. */
export function RevealOverlay({
  open,
  copy,
  reaction,
  onClose,
  onReaction,
}: RevealOverlayProps) {
  if (!open) return null;

  return (
    <div
      className="absolute inset-0 z-35 grid place-items-center bg-black/30 backdrop-blur-md"
      role="dialog"
      aria-modal="true"
      aria-label={copy.reactionsTitle}
      onClick={onClose}
    >
      <div
        className="relative size-[min(74vw,21rem)]"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="absolute inset-[18%] rounded-full border-8 border-[#6f7180] bg-[conic-gradient(#242632_0deg_58deg,#6e6e75_59deg_61deg,#242632_62deg_118deg,#6e6e75_119deg_121deg,#242632_122deg_178deg,#6e6e75_179deg_181deg,#242632_182deg_238deg,#6e6e75_239deg_241deg,#242632_242deg_298deg,#6e6e75_299deg_301deg,#242632_302deg_360deg)] shadow-2xl" />
        {REACTIONS.map((item, index) => (
          <button
            key={item.label}
            type="button"
            aria-label={item.label}
            onClick={() => onReaction(item.emoji)}
            className={`absolute ${item.position} grid size-16 place-items-center rounded-full text-3xl transition hover:scale-110 focus-visible:outline focus-visible:outline-2 focus-visible:outline-white`}
          >
            <span
              className={
                index === 5
                  ? "text-2xl font-black text-[#ffad21] italic [text-shadow:2px_2px_0_#9d4200]"
                  : ""
              }
            >
              {item.emoji}
            </span>
          </button>
        ))}
        <button
          type="button"
          aria-label={copy.closeReactions}
          onClick={onClose}
          className="absolute inset-[40%] grid place-items-center rounded-full border-2 border-white bg-[#4b4a55] text-lg"
        >
          ×
        </button>
      </div>
      <p className="absolute bottom-[14%] max-w-52 text-center text-sm font-black text-white italic">
        {reaction ? `${reaction} ${copy.reactionSent}` : copy.reactionsPrompt}
      </p>
    </div>
  );
}
