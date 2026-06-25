# GeoGuess

This repo is organized as a small full-stack workspace:

```text
client/   Next.js frontend
backend/  Go API backend
docs/     Planning and architecture documentation
```

Project work follows the Spec Kit constitution in
`.specify/memory/constitution.md`. Feature plans and reviews must satisfy its
code quality, testing, UX consistency, performance, and operational readiness
gates.

Run the frontend from the repo root:

```bash
npm run dev
```

Or work directly inside `client/`:

```bash
cd client
npx pnpm@10.24.0 install
npx pnpm@10.24.0 dev
```

The original frontend README now lives at `client/README.md`.

Common verification commands:

```powershell
npm run backend:test
npm run lint
npm run typecheck
npm run build
```
