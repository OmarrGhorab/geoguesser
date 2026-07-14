"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { cn } from "@/lib/utils";

type LandingHeroVideoProps = {
  src: string;
  className?: string;
  opacity?: number;
};

/**
 * Hero background video that avoids flashing a stale poster image.
 * Renders a solid base color until the video has enough data to play,
 * then fades the video in.
 */
export function LandingHeroVideo({
  src,
  className,
  opacity,
}: LandingHeroVideoProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [ready, setReady] = useState(false);

  const markReady = useCallback(() => {
    setReady(true);
  }, []);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;

    // Cached / already buffered on revisit
    if (video.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) {
      setReady(true);
    }

    const play = () => {
      void video.play().catch(() => {
        // Autoplay can fail if the browser blocks it; video stays muted.
      });
    };

    if (video.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) {
      play();
    }
  }, [src]);

  const targetOpacity = opacity ?? 1;

  return (
    <>
      {/* Brand base while the video buffers — never flash a stale poster image */}
      <div className="absolute inset-0 bg-[#020522]" aria-hidden="true" />
      <video
        ref={videoRef}
        className={cn(
          "absolute inset-0 size-full object-cover transition-opacity duration-500 ease-out",
          className,
        )}
        style={{ opacity: ready ? targetOpacity : 0 }}
        autoPlay
        loop
        muted
        playsInline
        preload="metadata"
        aria-hidden="true"
        onLoadedData={markReady}
        onCanPlay={markReady}
        onPlaying={markReady}
      >
        <source src={src} type="video/mp4" />
      </video>
    </>
  );
}
