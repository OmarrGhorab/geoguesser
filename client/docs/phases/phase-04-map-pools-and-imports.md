# Phase 4 - Map Pools And Imports

Goal: support multiple sets of locations instead of one hard-coded pool.

## Required Features

- Map pool model:
  - id
  - name
  - description
  - locations
  - tags
  - difficulty
- Built-in pools:
  - World
  - Landmarks
  - Europe
  - Americas
  - Asia
  - Africa
  - Oceania
- Map pool picker.
- Import JSON coordinates.
- Validate imported locations.
- Reject invalid latitude/longitude.

## Vali Import Support

- Accept Vali `*-locations.json` shape:

```json
[
  { "lat": 48.8584, "lng": 2.2945, "heading": 90, "pitch": 0, "zoom": 1 }
]
```

- Convert imported items into internal `Location` objects.
- Auto-generate placeholder names if needed.
- Preserve heading, pitch, zoom, and pano id if available.

## Done When

- Player can choose a pool before starting.
- Imported coordinates can be played.
- Bad imports show useful validation errors.
