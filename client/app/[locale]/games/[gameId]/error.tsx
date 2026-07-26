"use client";

export default function GameError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <main className="grid min-h-dvh place-items-center bg-[#08051d] px-4 text-white">
      <div className="flex max-w-md flex-col items-center gap-4 text-center">
        <h1 className="text-xl font-black">Game unavailable</h1>
        <p className="text-sm text-white/70">
          This game could not be loaded. Check your connection and try again.
        </p>
        <button
          type="button"
          onClick={reset}
          className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-6 py-2.5 text-xs font-black uppercase"
        >
          Retry
        </button>
      </div>
    </main>
  );
}
