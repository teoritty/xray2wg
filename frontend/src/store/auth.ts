import { create } from "zustand";

type AuthState = {
  signedIn: boolean;
  setSignedIn: (v: boolean) => void;
  clear: () => void;
};

export const useAuthStore = create<AuthState>()((set) => ({
  signedIn: false,
  setSignedIn: (v) => set({ signedIn: v }),
  clear: () => set({ signedIn: false }),
}));
