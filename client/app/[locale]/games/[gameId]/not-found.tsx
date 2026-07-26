import Link from "next/link";
import type { Route } from "next";

export default function GameNotFound() {
  return (
    <main className="grid min-h-dvh place-items-center bg-[#08051d] px-4 text-white">
      <div className="flex max-w-md flex-col items-center gap-4 text-center">
        <h1 className="text-xl font-black">Game not found</h1>
        <p className="text-sm text-white/70">
          This game does not exist or you no longer have access.
        </p>
        <Link
          href={"/" as Route}
          className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-6 py-2.5 text-xs font-black uppercase"
        >
          Back home
        </Link>
      </div>
    </main>
  );
}
