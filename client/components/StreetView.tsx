"use client";

import { useEffect, useRef, useState } from "react";
import { loadGoogleMaps } from "@/lib/loadGoogleMaps";
import type { Location } from "@/lib/locations";

/**
 * The "clue": an interactive Google Street View panorama of the current
 * landmark, rendered via the Maps JavaScript API's `StreetViewPanorama`.
 *
 * Unlike the Embed-API iframe (which shows one fixed snapshot), the JS-API
 * panorama is fully interactive — the player can drag to look around in
 * 360°, pan/zoom, and "walk" to adjacent panoramas. That's the real
 * GeoGuessr experience: infer the place from your surroundings, then move
 * to gather more clues.
 *
 * Fairness controls: we disable `addressControl` and `showRoadLabels` so the
 * API never prints the country/city/road on screen — the player still has to
 * guess from what they can see.
 *
 * The panorama is recreated whenever the landmark changes (the effect keys on
 * `location.id`), and the injected DOM is torn down on cleanup so tiles don't
 * leak across rounds.
 *
 * Why StreetViewService rather than passing `position` directly: many landmark
 * coordinates (monument centers, plazas) have no Street View coverage at the
 * exact point, and a bare `position` renders a black screen instead of
 * snapping to a nearby road. We query `getPanorama` within a 1km radius,
 * take the nearest outdoor panorama, and load it by explicit panorama ID —
 * reliable and (thanks to `source: OUTDOOR`) never a business photosphere.
 */
export type StreetViewProps = {
  location: Location;
};

export default function StreetView({ location }: StreetViewProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const panoramaRef = useRef<google.maps.StreetViewPanorama | null>(null);
  // Set when the API can't find a panorama for this coordinate.
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    let panorama: google.maps.StreetViewPanorama | null = null;
    // Capture the live DOM node once per effect run; the ref may have been
    // swapped to a different node by the time cleanup runs.
    const container = containerRef.current;

    loadGoogleMaps()
      .then((google) => {
        if (cancelled || !container) return;

        panorama = new google.maps.StreetViewPanorama(container, {
          // Don't auto-load on a bare position: many landmark coordinates
          // (monument centers, plazas) have no coverage at the exact point,
          // and the panorama renders black instead of snapping to a nearby
          // road. We instead resolve the nearest panorama via the
          // StreetViewService below and feed its explicit ID in.
          visible: false,
          // Fairness: never reveal the address, road names, or a close button
          // that would pop the player back to the map.
          addressControl: false,
          showRoadLabels: false,
          enableCloseButton: false,
          // Immersive navigation: look around + walk to adjacent panos.
          panControl: true,
          linksControl: true,
          motionTracking: false,
          motionTrackingControl: false,
        });
        panoramaRef.current = panorama;

        const randomHeading = Math.floor(Math.random() * 360);

        // Find the nearest available panorama to the landmark. `OUTDOOR`
        // excludes indoor/business photospheres (museums, shops) so the
        // player sees actual street context. 1km covers most cases where the
        // nearest road is a short walk from the monument.
        const service = new google.maps.StreetViewService();
        service.getPanorama(
          {
            location: { lat: location.lat, lng: location.lng },
            radius: 1000,
            source: google.maps.StreetViewSource.OUTDOOR,
          },
          (data, status) => {
            if (cancelled) return;
            if (status === "OK" && data?.location?.pano) {
              panorama!.setPano(data.location.pano);
              panorama!.setPov({ heading: randomHeading, pitch: 0 });
              panorama!.setVisible(true);
              setFailed(false);
            } else {
              // No coverage within the radius — show the fallback message
              // instead of a black void.
              setFailed(true);
            }
          },
        );
      })
      .catch((err) => {
        // Missing API key, network failure, etc.
        console.error(err);
        if (!cancelled) setFailed(true);
      });

    return () => {
      cancelled = true;
      // Hide + detach the DOM the API injected so nothing leaks into the next
      // round's panorama.
      panorama?.setVisible(false);
      if (container) container.innerHTML = "";
      panoramaRef.current = null;
    };
  }, [location.id, location.lat, location.lng]);

  return (
    // Full-viewport background layer. The panorama sits behind every overlay
    // (page.tsx renders this as the z-0 base). No card chrome — the panorama
    // *is* the interface.
    <div className="absolute inset-0 z-0 h-full w-full bg-zinc-800">
      <div ref={containerRef} className="h-full w-full" />

      {failed && (
        <div className="absolute inset-0 flex items-center justify-center bg-zinc-800 p-6 text-center text-sm text-zinc-300">
          No Street View coverage for this location. Check your Maps API key,
          or skip to the next round.
        </div>
      )}
    </div>
  );
}
