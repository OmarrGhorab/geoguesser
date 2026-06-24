"use client";

import { create } from "zustand";

type DistanceUnit = "kilometers" | "miles";

type PreferencesState = {
  distanceUnit: DistanceUnit;
  setDistanceUnit: (distanceUnit: DistanceUnit) => void;
};

export const usePreferencesStore = create<PreferencesState>((set) => ({
  distanceUnit: "kilometers",
  setDistanceUnit: (distanceUnit) => set({ distanceUnit }),
}));
