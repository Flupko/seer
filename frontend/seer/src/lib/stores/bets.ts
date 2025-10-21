import { BetUpdate } from "@/socket/messages";
import { create } from "zustand";

interface BetStore {
    latestBets: BetUpdate[];
    latestParity: number;
    addLatestBet: (bet: BetUpdate) => void;

    highBets: BetUpdate[];
    highParity: number;
    addHighBet: (bet: BetUpdate) => void;
}

export const useBetStore = create<BetStore>((set, get) => ({
    latestBets: [],
    latestParity: 0,
    addLatestBet: (bet: BetUpdate) => {
        const currentBets = get().latestBets;
        const updatedBets = [bet, ...currentBets].slice(0, 10); // Keep only the latest 10 bets
        set({ latestBets: updatedBets, latestParity: (get().latestParity + 1) % 2 });
    },

    highBets: [],
    highParity: 0,
    addHighBet: (bet: BetUpdate) => {
        const currentBets = get().highBets;
        const updatedBets = [bet, ...currentBets].slice(0, 10); // Keep only the top 10 high bets
        set({ highBets: updatedBets, highParity: (get().highParity + 1) % 2 });
    }
}));