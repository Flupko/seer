import { OnlineUpdate } from "@/socket/messages";
import { create } from "zustand";

export interface OnlineStore {
    onlineCount: number;
    updateOnlineCount: (onlineUpdate: OnlineUpdate) => void;
}

export const useOnlineStore = create<OnlineStore>((set) => ({
    onlineCount: 0,
    updateOnlineCount: (onlineUpdate: OnlineUpdate) => set({ onlineCount: onlineUpdate.usersOnlineCount }),
}));
