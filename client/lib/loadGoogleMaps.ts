/**
 * Singleton loader for the Google Maps JavaScript API.
 *
 * Instead of pulling a wrapper library like `@react-google-maps/api`, we
 * inject the API script exactly once and reuse the resulting promise for
 * every consumer. This honors the "no unnecessary libraries" constraint
 * while giving us full control over when/where the API loads.
 *
 * === Why a callback instead of script.onload ===
 *
 * The script's `onload` fires as soon as the bootstrap JS file is downloaded
 * — but with `loading=async`, Google's own loader finishes *after* that, so
 * `google.maps.Map` may not exist yet when `onload` runs. That caused
 * `google.maps.Map is not a constructor`.
 *
 * Google's documented, race-free pattern is to pass `callback=<globalFn>`:
 * the API invokes it only once `google.maps` is FULLY defined. We point that
 * callback at a one-time global that resolves our promise.
 *
 * The loader is idempotent:
 *   1. If `window.google.maps` already exists → resolve immediately.
 *   2. If a load is mid-flight → return the in-flight promise.
 *   3. Otherwise → inject `<script>` with `loading=async` + `callback=`.
 */

declare global {
  interface Window {
    /** Tracks a possibly-in-flight load so concurrent callers share it. */
    __googleMapsLoader?: Promise<typeof google>;
    /** The callback the Maps API will invoke once it's fully ready. */
    __initGoogleMaps?: () => void;
  }
}

const SCRIPT_ID = "google-maps-js-api";
/** Unique global callback name so it can't collide with anything else. */
const CALLBACK_NAME = "__initGoogleMaps";

/**
 * Loads the Maps JS API (once) and resolves with the `google` global once
 * `google.maps.Map` etc. are guaranteed to be defined.
 * @throws if no API key is configured via NEXT_PUBLIC_GOOGLE_MAPS_API_KEY.
 */
export function loadGoogleMaps(): Promise<typeof google> {
  // Browser-only guard; this module is only imported from client components,
  // but being defensive lets it never crash during SSR.
  if (typeof window === "undefined") {
    return Promise.reject(new Error("loadGoogleMaps called on the server"));
  }

  // Case 1: already fully loaded.
  if (window.google?.maps) {
    return Promise.resolve(window.google);
  }

  // Case 2: load already in progress.
  if (window.__googleMapsLoader) {
    return window.__googleMapsLoader;
  }

  // Case 3: start a fresh load.
  const apiKey = process.env.NEXT_PUBLIC_GOOGLE_MAPS_API_KEY;
  if (!apiKey) {
    return Promise.reject(
      new Error(
        "Missing NEXT_PUBLIC_GOOGLE_MAPS_API_KEY. Add it to .env.local and enable the Maps JavaScript API.",
      ),
    );
  }

  window.__googleMapsLoader = new Promise<typeof google>((resolve, reject) => {
    // Register the callback BEFORE appending the script. The API calls this
    // exactly once, only after google.maps.* constructors are all defined.
    window[CALLBACK_NAME] = () => {
      if (window.google?.maps) resolve(window.google);
      else reject(new Error("Maps callback fired but window.google.maps is missing"));
    };

    const params = new URLSearchParams({
      key: apiKey,
      v: "weekly",
      libraries: "marker",
      // loading=async is Google's recommended non-blocking load strategy.
      loading: "async",
      // The API invokes window[CALLBACK_NAME]() when fully ready.
      callback: CALLBACK_NAME,
    });
    const src = `https://maps.googleapis.com/maps/api/js?${params.toString()}`;

    const script = Object.assign(document.createElement("script"), {
      id: SCRIPT_ID,
      type: "text/javascript",
      async: true,
      src,
    });

    // onerror only fires for network/HTTP failures (not API logic errors);
    // the callback handles the success path.
    script.onerror = () =>
      reject(new Error("Failed to load the Google Maps JavaScript API script"));

    document.head.appendChild(script);
  });

  return window.__googleMapsLoader;
}
