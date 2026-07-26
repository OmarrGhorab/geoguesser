"use client";

import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from "react";
import {
  loadGoogleMaps,
  type GoogleMapsNamespace,
} from "@/features/gameplay/google-maps";
import { normalizedHeading } from "@/features/mission/compass";

export type SceneMedia = Readonly<{
  panorama_id?: string;
  url?: string;
  attribution?: string | null;
}>;

export type SceneViewerCopy = Readonly<{
  mapUnavailable: string;
  retryMap: string;
}>;

export type SceneViewerHandle = {
  changeZoom: (delta: number) => void;
  recenter: () => void;
};

type SceneViewerProps = Readonly<{
  media: SceneMedia;
  googleMapsApiKey: string;
  copy: SceneViewerCopy;
  retryAction?: () => Promise<void>;
  onHeadingChange?: (heading: number) => void;
  onMapErrorChange?: (mapError: boolean) => void;
  className?: string;
}>;

type StreetViewPanorama = InstanceType<
  GoogleMapsNamespace["StreetViewPanorama"]
>;

export const SceneViewer = forwardRef<SceneViewerHandle, SceneViewerProps>(
  function SceneViewer(
    {
      media,
      googleMapsApiKey,
      copy,
      retryAction,
      onHeadingChange,
      onMapErrorChange,
      className,
    },
    ref,
  ) {
    const panoramaRef = useRef<HTMLDivElement>(null);
    const panoramaInstanceRef = useRef<StreetViewPanorama | null>(null);
    const onHeadingChangeRef = useRef(onHeadingChange);
    const onMapErrorChangeRef = useRef(onMapErrorChange);
    const [mapError, setMapError] = useState(!googleMapsApiKey);

    onHeadingChangeRef.current = onHeadingChange;
    onMapErrorChangeRef.current = onMapErrorChange;

    useImperativeHandle(ref, () => ({
      changeZoom(delta: number) {
        const panorama = panoramaInstanceRef.current;
        if (panorama) {
          panorama.setZoom(
            Math.max(0, Math.min(5, panorama.getZoom() + delta)),
          );
        }
      },
      recenter() {
        panoramaInstanceRef.current?.setPov({ heading: 0, pitch: 0 });
      },
    }));

    useEffect(() => {
      onMapErrorChangeRef.current?.(mapError);
    }, [mapError]);

    useEffect(() => {
      if (!media.panorama_id || !googleMapsApiKey || !panoramaRef.current) {
        if (!googleMapsApiKey) setMapError(true);
        panoramaInstanceRef.current = null;
        return;
      }

      let active = true;
      let povListener: { remove(): void } | null = null;

      loadGoogleMaps(googleMapsApiKey)
        .then((maps) => {
          if (!active || !panoramaRef.current || !media.panorama_id) return;
          setMapError(false);
          panoramaInstanceRef.current = new maps.StreetViewPanorama(
            panoramaRef.current,
            {
              pano: media.panorama_id,
              pov: { heading: 0, pitch: 0 },
              zoom: 0,
              addressControl: false,
              fullscreenControl: false,
              motionTracking: false,
              showRoadLabels: false,
            },
          );
          const panorama = panoramaInstanceRef.current;
          onHeadingChangeRef.current?.(
            normalizedHeading(panorama.getPov().heading),
          );
          povListener = panorama.addListener("pov_changed", () => {
            onHeadingChangeRef.current?.(
              normalizedHeading(panorama.getPov().heading),
            );
          });
        })
        .catch(() => {
          if (active) setMapError(true);
        });

      return () => {
        active = false;
        povListener?.remove();
        panoramaInstanceRef.current = null;
      };
    }, [googleMapsApiKey, media.panorama_id]);

    const hasPlayableMedia = Boolean(media.panorama_id || media.url);
    const showFallback = !hasPlayableMedia || (mapError && !media.url);

    return (
      <>
        <div
          ref={panoramaRef}
          className={
            className ?? "absolute inset-0 bg-cover bg-center"
          }
          style={
            media.url
              ? { backgroundImage: `url(${JSON.stringify(media.url)})` }
              : undefined
          }
        />
        {showFallback ? (
          <div className="absolute inset-0 grid place-items-center bg-[radial-gradient(circle_at_center,#25205d,#08051d)]">
            <div className="flex flex-col items-center gap-3 rounded-3xl bg-black/35 px-6 py-5 text-center">
              <p className="font-bold text-white/75">{copy.mapUnavailable}</p>
              {!hasPlayableMedia && retryAction ? (
                <form action={retryAction}>
                  <button
                    type="submit"
                    className="rounded-full bg-gradient-to-b from-[#8b72ff] to-[#5538d4] px-7 py-2.5 text-sm font-black tracking-wide uppercase"
                  >
                    {copy.retryMap}
                  </button>
                </form>
              ) : null}
            </div>
          </div>
        ) : null}
      </>
    );
  },
);
