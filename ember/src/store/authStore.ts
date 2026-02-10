import { create } from "zustand";
import type { User } from "@pulse/drift/types";

interface AuthState {
  accessToken: string | null;
  user: User | null;
  isAuthenticated: () => boolean;
  setTokens: (accessToken: string) => void;
  setUser: (user: User) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  accessToken: null,
  user: (() => {
    try {
      const raw = localStorage.getItem("pulse_user");
      return raw ? (JSON.parse(raw) as User) : null;
    } catch {
      return null;
    }
  })(),

  isAuthenticated: () => get().accessToken !== null,

  setTokens: (accessToken) => {
    set({ accessToken });
  },

  setUser: (user) => {
    localStorage.setItem("pulse_user", JSON.stringify(user));
    set({ user });
  },

  logout: () => {
    localStorage.removeItem("pulse_user");
    set({ accessToken: null, user: null });
  },
}));
