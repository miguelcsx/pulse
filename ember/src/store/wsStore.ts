import { create } from "zustand";

interface WSState {
  connected: boolean;
  roomPresence: Record<string, number>;
  setConnected: (connected: boolean) => void;
  updatePresence: (roomId: string, memberCount: number) => void;
}

export const useWsStore = create<WSState>()((set) => ({
  connected: false,
  roomPresence: {},

  setConnected: (connected) => {
    set({ connected });
  },

  updatePresence: (roomId, memberCount) => {
    set((state) => ({
      roomPresence: { ...state.roomPresence, [roomId]: memberCount },
    }));
  },
}));
