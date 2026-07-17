type GoogleLatLngLiteral = { lat: number; lng: number };

type GoogleMouseEvent = {
  latLng: { lat(): number; lng(): number } | null;
};

export type GoogleMap = {
  setCenter(position: GoogleLatLngLiteral): void;
  setZoom(zoom: number): void;
  getZoom(): number | undefined;
  fitBounds(bounds: GoogleLatLngBounds): void;
};

export type GoogleLatLngBounds = {
  extend(position: GoogleLatLngLiteral): void;
};

export type GoogleMarker = {
  map: GoogleMap | null;
  position: GoogleLatLngLiteral | null;
};

export type GoogleMapsNamespace = {
  Map: new (
    element: HTMLElement,
    options: {
      center: GoogleLatLngLiteral;
      zoom: number;
      disableDefaultUI?: boolean;
      clickableIcons?: boolean;
      gestureHandling?: string;
      mapId?: string;
      keyboardShortcuts?: boolean;
      zoomControl?: boolean;
    },
  ) => GoogleMap & {
    addListener(
      event: "click",
      handler: (event: GoogleMouseEvent) => void,
    ): {
      remove(): void;
    };
  };
  marker: {
    AdvancedMarkerElement: new (options: {
      map: GoogleMap;
      position: GoogleLatLngLiteral;
      title?: string;
      content?: Node;
    }) => GoogleMarker;
  };
  LatLngBounds: new () => GoogleLatLngBounds;
  Polyline: new (options: {
    map: GoogleMap;
    path: GoogleLatLngLiteral[];
    geodesic?: boolean;
    strokeColor?: string;
    strokeOpacity?: number;
    strokeWeight?: number;
    icons?: Array<{
      icon: { path: string; scale?: number; strokeColor?: string };
      offset: string;
      repeat?: string;
    }>;
  }) => { setMap(map: GoogleMap | null): void };
  event: {
    trigger(instance: unknown, eventName: string): void;
  };
  importLibrary(name: "marker"): Promise<unknown>;
  StreetViewPanorama: new (
    element: HTMLElement,
    options: {
      pano: string;
      pov: { heading: number; pitch: number };
      zoom: number;
      addressControl?: boolean;
      fullscreenControl?: boolean;
      motionTracking?: boolean;
      showRoadLabels?: boolean;
    },
  ) => {
    getZoom(): number;
    getPov(): { heading: number; pitch: number };
    setZoom(zoom: number): void;
    setPov(pov: { heading: number; pitch: number }): void;
    addListener(event: "pov_changed", handler: () => void): { remove(): void };
  };
};

type GoogleMapsRuntime = Omit<GoogleMapsNamespace, "marker"> & {
  marker?: GoogleMapsNamespace["marker"];
};

declare global {
  interface Window {
    google?: { maps: GoogleMapsRuntime };
    __worldguessGoogleMapsReady?: () => void;
  }
}

let mapsPromise: Promise<GoogleMapsNamespace> | null = null;

async function requireMarkerLibrary(
  maps: GoogleMapsRuntime,
): Promise<GoogleMapsNamespace> {
  if (!maps.marker?.AdvancedMarkerElement) {
    await maps.importLibrary("marker");
  }
  if (!maps.marker?.AdvancedMarkerElement) {
    throw new Error("Google Maps marker library is unavailable");
  }
  return maps as GoogleMapsNamespace;
}

export function loadGoogleMaps(apiKey: string): Promise<GoogleMapsNamespace> {
  if (window.google?.maps) return requireMarkerLibrary(window.google.maps);
  if (mapsPromise) return mapsPromise;

  mapsPromise = new Promise((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(
      "script[data-worldguess-google-maps]",
    );
    const onReady = () => {
      if (window.google?.maps) {
        requireMarkerLibrary(window.google.maps).then(resolve, reject);
      } else reject(new Error("Google Maps loaded without a maps namespace"));
      delete window.__worldguessGoogleMapsReady;
    };
    const onError = () => {
      mapsPromise = null;
      delete window.__worldguessGoogleMapsReady;
      reject(new Error("Google Maps failed to load"));
    };

    if (existing) {
      existing.addEventListener("error", onError, { once: true });
      return;
    }

    window.__worldguessGoogleMapsReady = onReady;
    const script = document.createElement("script");
    const params = new URLSearchParams({
      key: apiKey,
      v: "weekly",
      loading: "async",
      libraries: "marker",
      callback: "__worldguessGoogleMapsReady",
    });
    script.src = `https://maps.googleapis.com/maps/api/js?${params.toString()}`;
    script.async = true;
    script.dataset.worldguessGoogleMaps = "true";
    script.addEventListener("error", onError, { once: true });
    document.head.append(script);
  });

  return mapsPromise;
}
