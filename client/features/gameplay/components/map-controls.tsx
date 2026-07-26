"use client";

import { Crosshair, Flag, Minus, Plus } from "lucide-react";

export type MapControlsCopy = Readonly<{
  zoomIn: string;
  zoomOut: string;
  recenter: string;
  openReactions: string;
}>;

type MapControlsProps = Readonly<{
  copy: MapControlsCopy;
  onZoomIn: () => void;
  onZoomOut: () => void;
  onRecenter: () => void;
  onOpenReactions?: () => void;
  reactionOpen?: boolean;
  showReactions?: boolean;
}>;

function Control({
  label,
  onClick,
  children,
}: {
  label: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className="grid size-11 place-items-center rounded-full bg-black/65 text-white shadow-lg backdrop-blur transition hover:bg-black/85 focus-visible:outline focus-visible:outline-2 focus-visible:outline-white"
    >
      {children}
    </button>
  );
}

export function MapControls({
  copy,
  onZoomIn,
  onZoomOut,
  onRecenter,
  onOpenReactions,
  reactionOpen = false,
  showReactions = true,
}: MapControlsProps) {
  return (
    <>
      <div className="absolute bottom-7 left-4 z-20 flex flex-col gap-2">
        <Control label={copy.zoomIn} onClick={onZoomIn}>
          <Plus className="size-5" />
        </Control>
        <Control label={copy.zoomOut} onClick={onZoomOut}>
          <Minus className="size-5" />
        </Control>
        <Control label={copy.recenter} onClick={onRecenter}>
          <Crosshair className="size-5" />
        </Control>
      </div>
      {showReactions && onOpenReactions ? (
        <button
          type="button"
          aria-label={copy.openReactions}
          aria-pressed={reactionOpen}
          onClick={onOpenReactions}
          className="absolute bottom-7 left-20 z-20 grid size-11 place-items-center rounded-full bg-black/60 text-white shadow-lg backdrop-blur"
        >
          <Flag className="size-5" />
        </button>
      ) : null}
    </>
  );
}
