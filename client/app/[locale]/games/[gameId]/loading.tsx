export default function RankedGameLoading() {
  return (
    <main className="mx-auto w-full max-w-3xl animate-pulse px-4 py-8" aria-busy="true" aria-label="Loading game">
      <div className="mb-3 h-8 w-1/3 rounded bg-zinc-200 dark:bg-zinc-800" />
      <div className="h-4 w-2/3 rounded bg-zinc-200 dark:bg-zinc-800" />
    </main>
  );
}
