import { create } from "zustand";
import type { RoomContext } from "@pulse/drift/types";

interface FeedContextState {
  activeRoom: RoomContext | null;
  contextSheetOpen: boolean;
  setActiveRoom: (room: RoomContext | null) => void;
  openSheet: () => void;
  closeSheet: () => void;
}

export const useFeedContextStore = create<FeedContextState>((set) => ({
  activeRoom: null,
  contextSheetOpen: false,
  setActiveRoom: (room) => set({ activeRoom: room }),
  openSheet: () => set({ contextSheetOpen: true }),
  closeSheet: () => set({ contextSheetOpen: false }),
}));
