# GeoGuessr

A minimal, modern GeoGuessr-style guessing game built with **Next.js 16 (App Router)**, **TypeScript**, and **Tailwind CSS v4**. No backend, no database — entirely client-side.

The player sees a Google **Street View** panorama of a real-world place, drops a marker on a world map to guess where it is, and scores up to 5000 points based on how close they were.

## Setup

### 1. Get a Google Maps API key

1. Open the [Google Cloud Console](https://console.cloud.google.com/google/maps-apis) and create / select a project.
2. Create an API key under **Credentials**.
3. Enable **one** API for that key:
   - **Maps JavaScript API** (interactive guess map *and* the Street View panorama clue)
4. (Recommended) Restrict the key to your domain(s) in production.

### 2. Configure the key

Copy the example env file and paste your key:

```bash
cp .env.example .env.local
```

```env
NEXT_PUBLIC_GOOGLE_MAPS_API_KEY=your_api_key_here
```

### 3. Run

```bash
npm install
npm run dev
```

Open http://localhost:3000.

## How it works

| Concern | Approach |
| --- | --- |
| Clue | Google **Street View Panorama** (`google.maps.StreetViewPanorama`) — drag to look around, walk to adjacent spots; no labels, true guessing |
| Guess map | Google Maps **JavaScript API**, click-to-place marker |
| Distance | Great-circle **Haversine** distance (Earth radius 6371 km) |
| Scoring | Exponential decay: `score = 5000 × e^(−km/500)` — rewards near-misses, tapers smoothly |
| Places | Curated 150-location test pool in `lib/locations.ts`; Vali can replace it with generated Street View coordinates |

### Scoring reference

| Distance | Score |
| --- | --- |
| 0 km | 5000 |
| 50 km | 4524 |
| 100 km | 4094 |
| 250 km | 3033 |
| 500 km | 1839 |
| 1000 km | 677 |

## Project structure

```
app/
  layout.tsx        Root layout + metadata
  page.tsx          Game orchestration & state (client component)
  globals.css       Tailwind v4 + base styles
components/
  StreetView.tsx    Interactive Street View panorama (JS API)
  GuessMap.tsx      JS-API interactive map (click / reveal)
  ResultCard.tsx    Post-submit result panel
  ScoreBar.tsx      Title + score + round
lib/
  locations.ts      Location type + 150-place dataset + no-repeat random picker
  haversine.ts      Great-circle distance
  scoring.ts        Distance → score
  loadGoogleMaps.ts Singleton Maps JS API loader
  types.ts          Shared domain types
```

## Generating places with Vali

Vali is a separate CLI for generating GeoGuessr/map-making coordinate sets. Its GitHub repo contains the tool source, but not the full downloaded Street View data pool, so generation happens outside this app first.

```bash
dotnet tool install -g vali
vali download
vali create-file
vali generate --file "my-map.json"
```

Vali writes a `*-locations.json` file shaped like:

```json
[
  { "lat": 48.8584, "lng": 2.2945, "heading": 90, "pitch": 0, "zoom": 1 }
]
```

To use that in this app, convert each item to the `Location` shape in `lib/locations.ts` by adding an `id`, `name`, `city`, `country`, `region`, and `difficulty`. The current Google Maps Street View loader will resolve nearby outdoor panoramas from the coordinates.

## Notes

- Score resets on page reload (client-only state by design).
- No animations, no authentication, no theme toggle — kept intentionally minimal.
- Only runtime deps: `next`, `react`, `react-dom`. `@types/google.maps` is dev-only for typings; the API is loaded via a hand-written loader rather than a wrapper library.
