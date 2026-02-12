import { create } from "zustand";
import { WS_MESSAGE_TYPES } from "@pulse/drift/constants";

interface WSState {
  connected: boolean;
  roomPresence: Record<string, number>;
  setConnected: (connected: boolean) => void;
  updatePresence: (roomId: string, memberCount: number) => void;

  /** Set by useWebSocket when the connection opens/closes. */
  _send: ((data: string) => void) | null;
  setSend: (send: ((data: string) => void) | null) => void;

  joinRoom: (roomId: string) => void;
  leaveRoom: (roomId: string) => void;
}

export const useWsStore = create<WSState>()((set, get) => ({
  connected: false,
  roomPresence: {},
  _send: null,

  setConnected: (connected) => {
    set({ connected });
  },

  updatePresence: (roomId, memberCount) => {
    set((state) => ({
      roomPresence: { ...state.roomPresence, [roomId]: memberCount },
    }));
  },

  setSend: (send) => {
    set({ _send: send });
  },

  joinRoom: (roomId) => {
    get()._send?.(JSON.stringify({ type: WS_MESSAGE_TYPES.JOIN_ROOM, room_id: roomId }));
  },

  leaveRoom: (roomId) => {
    get()._send?.(JSON.stringify({ type: WS_MESSAGE_TYPES.LEAVE_ROOM, room_id: roomId }));
  },
}));
